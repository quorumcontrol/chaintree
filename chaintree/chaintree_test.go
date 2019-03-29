package chaintree

import (
	"fmt"
	"strings"
	"testing"

	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const errInvalidPayload = 999

func init() {
	typecaster.AddType(setDataPayload{})
}

func hasCoolHeader(_ *dag.Dag, blockWithHeaders *BlockWithHeaders) (valid bool, err CodedError) {
	headerVal, ok := blockWithHeaders.Headers["cool"].(string)
	if ok {
		return headerVal == "cool", nil
	}
	return false, nil
}

type setDataPayload struct {
	Path  string
	Value interface{}
}

func setData(tree *dag.Dag, transaction *Transaction) (newTree *dag.Dag, valid bool, codedErr CodedError) {
	payload := &setDataPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: errInvalidPayload, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	newTree, err = tree.Set(strings.Split(payload.Path, "/"), payload.Value)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

func TestChainTree_Id(t *testing.T) {
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

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := dag.NewDagWithNodes(store, root, tree, chain)
	require.Nil(t, err)
	chainTree, err := NewChainTree(
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[string]TransactorFunc{
			"SET_DATA": setData,
		},
	)
	assert.Nil(t, err)

	id, err := chainTree.Id()
	assert.Nil(t, err)
	assert.Equal(t, "test", id)

}

func TestHeightValidation(t *testing.T) {
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

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := dag.NewDagWithNodes(store, root, treeNode, chainNode)
	require.Nil(t, err)

	tree, err := NewChainTree(
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[string]TransactorFunc{
			"SET_DATA": setData,
		},
	)
	require.Nil(t, err)

	t.Run("first block fails with a non-zero height", func(t *testing.T) {
		block := &BlockWithHeaders{
			Block: Block{
				Height: 1,
				Transactions: []*Transaction{
					{
						Type: "SET_DATA",
						Payload: map[string]string{
							"path":  "down/in/the/thing",
							"value": "hi",
						},
					},
				},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err := tree.ProcessBlock(block)
		require.NotNil(t, err)
		require.False(t, valid)
	})

	t.Run("first block succeeds with a zero height, next requires a 1", func(t *testing.T) {
		block := &BlockWithHeaders{
			Block: Block{
				Height: 0,
				Transactions: []*Transaction{
					{
						Type: "SET_DATA",
						Payload: map[string]string{
							"path":  "down/in/the/thing",
							"value": "hi",
						},
					},
				},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err := tree.ProcessBlock(block)
		require.Nil(t, err)
		require.True(t, valid)
		height, _, err := tree.Dag.Resolve([]string{"height"})
		require.Nil(t, err)
		assert.Equal(t, uint64(0), height)

		// next fail with a zero
		block2 := &BlockWithHeaders{
			Block: Block{
				Height: 0,
				Transactions: []*Transaction{
					{
						Type: "SET_DATA",
						Payload: map[string]string{
							"path":  "down/in/the/thing",
							"value": "different",
						},
					},
				},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err = tree.ProcessBlock(block2)
		require.NotNil(t, err)
		require.False(t, valid)

		// then succeed with a 1
		block2 = &BlockWithHeaders{
			Block: Block{
				PreviousTip: &tree.Dag.Tip,
				Height:      1,
				Transactions: []*Transaction{
					{
						Type: "SET_DATA",
						Payload: map[string]string{
							"path":  "down/in/the/thing",
							"value": "different",
						},
					},
				},
			},
			Headers: map[string]interface{}{
				"cool": "cool",
			},
		}

		valid, err = tree.ProcessBlock(block2)
		require.Nil(t, err)
		require.True(t, valid)

		height, _, err = tree.Dag.Resolve([]string{"height"})
		require.Nil(t, err)
		assert.Equal(t, uint64(1), height)
	})

}

func TestBuildingUpAChain(t *testing.T) {
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

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := dag.NewDagWithNodes(store, root, treeNode, chainNode)
	require.Nil(t, err)

	tree, err := NewChainTree(
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[string]TransactorFunc{
			"SET_DATA": setData,
		},
	)
	require.Nil(t, err)

	block := &BlockWithHeaders{
		Block: Block{
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path":  "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid, err := tree.ProcessBlock(block)
	require.Nil(t, err)
	require.True(t, valid)

	_, _, err = tree.Dag.Resolve([]string{"chain", "end"})
	require.Nil(t, err)
	//assert.Equal(t, blockCid, entry.([]interface{})[0].(cid.Cid))

	block2 := &BlockWithHeaders{
		Block: Block{
			Height:      1,
			PreviousTip: &tree.Dag.Tip,
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path":  "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid, err = tree.ProcessBlock(block2)
	require.Nil(t, err)
	assert.True(t, valid)

	block1Cid := sw.WrapObject(block).Cid()
	assert.Nil(t, sw.Err)

	entry, remain, err := tree.Dag.Resolve([]string{"chain", "end"})
	assert.Nil(t, err)
	assert.Len(t, remain, 0)
	t.Log("previousTip", entry.(map[string]interface{}), "block1Cid", block1Cid.String())
	assert.True(t, entry.(map[string]interface{})["previousBlock"].(cid.Cid).Equals(block1Cid))
}

func TestBlockProcessing(t *testing.T) {
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
					Transactions: []*Transaction{
						{
							Type: "SET_DATA",
							Payload: map[string]string{
								"path":  "down/in/the/thing",
								"value": "hi",
							},
						},
					},
				},
				Headers: map[string]interface{}{
					"cool": "cool",
				},
			},
			validator: func(tree *ChainTree) {
				val, _, err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
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
					Transactions: []*Transaction{
						{
							Type: "SET_DATA",
							Payload: map[string]string{
								"path":  "down/in/the/thing",
								"value": "hi",
							},
						},
					},
				},
				Headers: map[string]interface{}{
					"cool": "NOT COOl!",
				},
			},
			validator: func(tree *ChainTree) {
				val, _, err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
				assert.Nil(t, val)
				assert.Nil(t, err)
			},
		},
		{
			description: "a block that has a bad transaction",
			shouldValid: false,
			shouldErr:   true,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*Transaction{
						{
							Type:    "SET_DATA",
							Payload: "broken payload",
						},
					},
				},
				Headers: map[string]interface{}{
					"cool": "cool",
				},
			},
			validator: func(tree *ChainTree) {
				val, _, err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
				assert.Nil(t, val)
				assert.Nil(t, err)
			},
		},
	} {
		store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
		dag, err := dag.NewDagWithNodes(store, root, tree, chain)
		require.Nil(t, err)

		tree, err := NewChainTree(
			dag,
			[]BlockValidatorFunc{hasCoolHeader},
			map[string]TransactorFunc{
				"SET_DATA": setData,
			},
		)
		assert.Nil(t, err)
		valid, err := tree.ProcessBlock(test.block)
		if !test.shouldErr {
			assert.Nil(t, err, test.description)
		}

		if test.shouldValid {
			assert.True(t, valid, test.description)
			wrappedBlock := sw.WrapObject(test.block)
			assert.Nil(t, sw.Err, test.description)
			node, err := tree.Dag.Get(wrappedBlock.Cid())
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

	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := dag.NewDagWithNodes(store, root, tree, chain)
	require.Nil(b, err)

	chainTree, err := NewChainTree(
		dag,
		[]BlockValidatorFunc{hasCoolHeader},
		map[string]TransactorFunc{
			"SET_DATA": setData,
		},
	)
	require.Nil(b, err)

	block := &BlockWithHeaders{
		Block: Block{
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path":  "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}
	valid, err := chainTree.ProcessBlock(block)
	require.Nil(b, err)
	require.True(b, valid)

	for n := 0; n < b.N; n++ {
		_, _, err = chainTree.Dag.Resolve([]string{"tree", "down", "in", "the", "thing"})
	}
	require.Nil(b, err)

}
