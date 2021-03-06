package chaintree

import (
	"context"
	"fmt"
	"strings"
	"testing"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hasCoolHeader(_ *dag.Dag, blockWithHeaders *BlockWithHeaders) (valid bool, err CodedError) {
	headerVal, ok := blockWithHeaders.Headers["cool"].(string)
	if ok {
		return headerVal == "cool", nil
	}
	return false, nil
}

func setData(_ string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr CodedError) {
	payload, err := transaction.EnsureSetDataPayload()
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: "not a SetData transaction"}
	}

	var val interface{}
	err = cbornode.DecodeInto(payload.Value, &val)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding data value: %v", err)}
	}

	newTree, err = tree.Set(context.Background(), strings.Split(payload.Path, "/"), val)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

func TestChainTree_Id(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sw := &safewrap.SafeWrap{}

	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
		"id":    "test",
	})

	store := nodestore.MustMemoryStore(ctx)
	dag, err := dag.NewDagWithNodes(ctx, store, root, tree, chain)
	require.Nil(t, err)
	chainTree, err := NewChainTree(
		ctx,
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[transactions.Transaction_Type]TransactorFunc{
			transactions.Transaction_SETDATA: setData,
		},
	)
	assert.Nil(t, err)

	id, err := chainTree.Id(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "test", id)

}

func TestHeightValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	treeNode := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chainNode := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chainNode.Cid(),
		"tree":  treeNode.Cid(),
	})

	assert.Nil(t, sw.Err)

	store := nodestore.MustMemoryStore(ctx)
	dag, err := dag.NewDagWithNodes(ctx, store, root, treeNode, chainNode)
	require.Nil(t, err)

	tree, err := NewChainTree(
		ctx,
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[transactions.Transaction_Type]TransactorFunc{
			transactions.Transaction_SETDATA: setData,
		},
	)
	require.Nil(t, err)

	t.Run("first block fails with a non-zero height", func(t *testing.T) {
		txn, err := NewSetDataTransaction("down/in/the/thing", "hi")
		require.Nil(t, err)
		block := &BlockWithHeaders{
			Block: Block{
				Height:       1,
				Transactions: []*transactions.Transaction{txn},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err := tree.ProcessBlock(ctx, block)
		require.NotNil(t, err)
		require.False(t, valid)
	})

	t.Run("first block succeeds with a zero height, next requires a 1", func(t *testing.T) {
		txn, err := NewSetDataTransaction("down/in/the/thing", "hi")
		require.Nil(t, err)
		block := &BlockWithHeaders{
			Block: Block{
				Height:       0,
				Transactions: []*transactions.Transaction{txn},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err := tree.ProcessBlock(ctx, block)
		require.Nil(t, err)
		require.True(t, valid)
		height, _, err := tree.Dag.Resolve(ctx, []string{"height"})
		require.Nil(t, err)
		assert.Equal(t, 0, height)

		// next fail with a zero
		txn2, err := NewSetDataTransaction("down/in/the/thing", "different")
		require.Nil(t, err)
		block2 := &BlockWithHeaders{
			Block: Block{
				Height:       0,
				Transactions: []*transactions.Transaction{txn2},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err = tree.ProcessBlock(ctx, block2)
		require.NotNil(t, err)
		require.False(t, valid)

		// then succeed with a 1
		txn2, err = NewSetDataTransaction("down/in/the/thing", "different")
		require.Nil(t, err)
		block2 = &BlockWithHeaders{
			Block: Block{
				PreviousTip:  &tree.Dag.Tip,
				Height:       1,
				Transactions: []*transactions.Transaction{txn2},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err = tree.ProcessBlock(ctx, block2)
		require.Nil(t, err)
		require.True(t, valid)

		height, _, err = tree.Dag.Resolve(ctx, []string{"height"})
		require.Nil(t, err)
		assert.Equal(t, 1, height)
	})

}

func TestBuildingUpAChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	treeNode := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chainNode := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chainNode.Cid(),
		"tree":  treeNode.Cid(),
	})

	assert.Nil(t, sw.Err)

	store := nodestore.MustMemoryStore(ctx)
	dag, err := dag.NewDagWithNodes(ctx, store, root, treeNode, chainNode)
	require.Nil(t, err)

	tree, err := NewChainTree(
		ctx,
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[transactions.Transaction_Type]TransactorFunc{
			transactions.Transaction_SETDATA: setData,
		},
	)
	require.Nil(t, err)
	startingTip := tree.Dag.Tip

	txn, err := NewSetDataTransaction("down/in/the/thing", "hi")
	require.Nil(t, err)
	block := &BlockWithHeaders{
		Block: Block{
			Transactions: []*transactions.Transaction{txn},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid, err := tree.ProcessBlock(ctx, block)
	require.Nil(t, err)
	require.True(t, valid)

	_, _, err = tree.Dag.Resolve(ctx, []string{"chain", "end"})
	require.Nil(t, err)
	//assert.Equal(t, blockCid, entry.([]interface{})[0].(cid.Cid))

	txn2, err := NewSetDataTransaction("down/in/the/thing", "hi")
	require.Nil(t, err)
	block2 := &BlockWithHeaders{
		Block: Block{
			Height:       1,
			PreviousTip:  &tree.Dag.Tip,
			Transactions: []*transactions.Transaction{txn2},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid, err = tree.ProcessBlock(ctx, block2)
	require.Nil(t, err)
	assert.True(t, valid)

	block1Cid := sw.WrapObject(block).Cid()
	assert.Nil(t, sw.Err)

	entry, remain, err := tree.Dag.Resolve(ctx, []string{"chain", "end"})
	assert.Nil(t, err)
	assert.Len(t, remain, 0)
	t.Log("previousBlock", entry.(map[string]interface{}), "block1Cid", block1Cid.String())
	assert.True(t, entry.(map[string]interface{})["previousBlock"].(cid.Cid).Equals(block1Cid))

	require.NotEqual(t, tree.Dag.Tip.String(), startingTip.String())
	require.NotEqual(t, tree.root.cid.String(), startingTip.String())

	// check `.At` can retrieve a previous state of chaintree
	startingTree, err := tree.At(ctx, &startingTip)
	require.Nil(t, err)
	assert.Equal(t, startingTip.String(), startingTree.Dag.Tip.String())
	assert.Equal(t, startingTip.String(), startingTree.root.cid.String())
}

func TestBlockProcessing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})

	assert.Nil(t, sw.Err)

	validTxn, err := NewSetDataTransaction("down/in/the/thing", "hi")
	assert.Nil(t, err)

	for _, test := range []struct {
		description string
		shouldValid bool
		shouldErr   bool
		block       *BlockWithHeaders
		validator   func(tree *ChainTree)
	}{
		{
			description: "a valid set data",
			shouldValid: true,
			shouldErr:   false,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*transactions.Transaction{validTxn},
				},
				Headers: map[string]interface{}{
					"cool": "cool",
				},
			},
			validator: func(tree *ChainTree) {
				val, _, err := tree.Dag.Resolve(ctx, strings.Split("tree/down/in/the/thing", "/"))
				assert.Nil(t, err, "valid data set resolution")
				assert.Equal(t, "hi", val)
			},
		},
		{
			description: "a block that fails block validators",
			shouldValid: false,
			shouldErr:   false,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*transactions.Transaction{validTxn},
				},
				Headers: map[string]interface{}{
					"cool": "NOT COOl!",
				},
			},
			validator: func(tree *ChainTree) {
				val, _, err := tree.Dag.Resolve(ctx, strings.Split("tree/down/in/the/thing", "/"))
				assert.Nil(t, val)
				assert.Nil(t, err)
			},
		},
	} {
		store := nodestore.MustMemoryStore(ctx)
		dag, err := dag.NewDagWithNodes(ctx, store, root, tree, chain)
		require.Nil(t, err)

		tree, err := NewChainTree(
			ctx,
			dag,
			[]BlockValidatorFunc{hasCoolHeader},
			map[transactions.Transaction_Type]TransactorFunc{
				transactions.Transaction_SETDATA: setData,
			},
		)
		assert.Nil(t, err)
		valid, err := tree.ProcessBlock(ctx, test.block)
		if !test.shouldErr {
			assert.Nil(t, err, test.description)
		}

		if test.shouldValid {
			assert.True(t, valid, test.description)
			wrappedBlock := sw.WrapObject(test.block)
			assert.Nil(t, sw.Err, test.description)
			node, err := tree.Dag.Get(ctx, wrappedBlock.Cid())
			assert.Nil(t, err)
			assert.NotNil(t, node, test.description)
		}

		if test.validator != nil {
			test.validator(tree)
		}
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	sw := &safewrap.SafeWrap{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	store := nodestore.MustMemoryStore(ctx)
	dag, err := dag.NewDagWithNodes(ctx, store, root, tree, chain)
	require.Nil(b, err)

	chainTree, err := NewChainTree(
		ctx,
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[transactions.Transaction_Type]TransactorFunc{
			transactions.Transaction_SETDATA: setData,
		},
	)
	require.Nil(b, err)

	txn, err := NewSetDataTransaction("down/in/the/thing", "hi")
	require.Nil(b, err)

	block := &BlockWithHeaders{
		Block: Block{
			Transactions: []*transactions.Transaction{txn},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}
	valid, err := chainTree.ProcessBlock(ctx, block)
	require.Nil(b, err)
	require.True(b, valid)

	for n := 0; n < b.N; n++ {
		_, _, err = chainTree.Dag.Resolve(ctx, []string{"tree", "down", "in", "the", "thing"})
	}
	require.Nil(b, err)
}
