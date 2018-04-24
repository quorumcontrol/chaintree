package chaintree

import (
	"testing"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/stretchr/testify/assert"
	"github.com/quorumcontrol/chaintree/typecaster"
	"fmt"
	"strings"
	"github.com/ipfs/go-cid"
)

const errInvalidPayload = 999

func init() {
	typecaster.AddType(setDataPayload{})
}

func hasCoolHeader(tree *dag.BidirectionalTree, blockWithHeaders *BlockWithHeaders) (valid bool, err CodedError) {
	headerVal,ok :=  blockWithHeaders.Headers["cool"].(string)
	if ok {
		return headerVal == "cool", nil
	}
	return false, nil
}

type setDataPayload struct {
	Path string
	Value interface{}
}


func setData(tree *dag.BidirectionalTree, transaction *Transaction) (valid bool, codedErr CodedError) {
	payload := &setDataPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return false, &ErrorCode{Code: errInvalidPayload, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	err = tree.Set(strings.Split(payload.Path, "/"), payload.Value)
	if err != nil {
		return false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return true, nil
}

func TestBuildingUpAChain(t *testing.T) {
	sw := &dag.SafeWrap{}


	treeNode := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chainNode := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chainNode.Cid(),
		"tree": treeNode.Cid(),
	})

	assert.Nil(t, sw.Err)
	tree,err := NewChainTree(
		dag.NewBidirectionalTree(root.Cid(), root,treeNode,chainNode),
		[]BlockValidatorFunc{hasCoolHeader},
		map[string]TransactorFunc{
			"SET_DATA": setData,
		},
	)
	assert.Nil(t, err)

	block := &BlockWithHeaders{
		Block: Block{
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path": "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid,err := tree.ProcessBlock(block)
	assert.Nil(t, err)
	assert.True(t,valid)

	//blockCid := sw.WrapObject(block).Cid()
	assert.Nil(t, sw.Err)

	//tree.Dag.Dump()

	entry,_,err := tree.Dag.Resolve([]string{"chain", "end", "blocksWithHeaders"})
	assert.Nil(t, err)
	//assert.Equal(t, blockCid, entry.([]interface{})[0].(cid.Cid))

	currAndOldTip := tree.Dag.Tip.String()

	block2 := &BlockWithHeaders{
		Block: Block{
			PreviousTip: currAndOldTip,
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path": "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid,err = tree.ProcessBlock(block2)
	assert.Nil(t, err)
	assert.True(t,valid)

	block2Cid := sw.WrapObject(block2).Cid()
	assert.Nil(t, sw.Err)
	//defer func() {
	//	if r := recover(); r != nil {
	//		t.Log(spew.Sdump(entry))
	//		t.Logf("Recovered in f: %v", r)
	//		t.Log(tree.Dag.Dump())
	//	}
	//}()
	entry,_,err = tree.Dag.Resolve([]string{"chain", "end", "blocksWithHeaders"})
	assert.Nil(t, err)
	assert.Equal(t, block2Cid, entry.([]interface{})[0].(*cid.Cid))


	// you can build on the same segment of the chain
	block3 := &BlockWithHeaders{
		Block: Block{
			PreviousTip: currAndOldTip,
			Transactions: []*Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path": "down/in/the/thing",
						"value": "hi",
					},
				},
			},
		},
		Headers: map[string]interface{}{
			"cool": "cool",
		},
	}

	valid,err = tree.ProcessBlock(block3)
	assert.Nil(t, err)
	assert.True(t,valid)

	block3Cid := sw.WrapObject(block3).Cid()
	assert.Nil(t, sw.Err)

	entry,_,err = tree.Dag.Resolve([]string{"chain", "end", "blocksWithHeaders"})
	assert.Nil(t, err)
	assert.Len(t, entry, 2)
	assert.Equal(t, block3Cid, entry.([]interface{})[1].(*cid.Cid))
}

func TestBlockProcessing(t *testing.T) {
	sw := &dag.SafeWrap{}


	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree": tree.Cid(),
	})

	assert.Nil(t, sw.Err)

	for _,test := range []struct{
		description string
		shouldValid bool
		shouldErr bool
		block *BlockWithHeaders
		validator func(tree *ChainTree)
	} {
		{
			description: "a valid set data",
			shouldValid: true,
			shouldErr: false,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*Transaction{
						{
							Type: "SET_DATA",
							Payload: map[string]string{
								"path": "down/in/the/thing",
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
				val,_,err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
				assert.Nil(t, err, "valid data set resolution")
				assert.Equal(t, "hi", val)
			},
		},
		{
			description: "a block that fails block validators",
			shouldValid: false,
			shouldErr: false,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*Transaction{
						{
							Type: "SET_DATA",
							Payload: map[string]string{
								"path": "down/in/the/thing",
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
				_,_,err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
				assert.Equal(t, dag.ErrMissingPath, err.(*dag.ErrorCode).Code)
			},
		},
		{
			description: "a block that has a bad transaction",
			shouldValid: false,
			shouldErr: true,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*Transaction{
						{
							Type: "SET_DATA",
							Payload: "broken payload",
						},
					},
				},
				Headers: map[string]interface{}{
					"cool": "cool",
				},
			},
			validator: func(tree *ChainTree) {
				_,_,err := tree.Dag.Resolve(strings.Split("tree/down/in/the/thing", "/"))
				assert.Equal(t, dag.ErrMissingPath, err.(*dag.ErrorCode).Code)
			},
		},
	} {
		tree,err := NewChainTree(
			dag.NewBidirectionalTree(root.Cid(), root,tree,chain),
			[]BlockValidatorFunc{hasCoolHeader},
			map[string]TransactorFunc{
				"SET_DATA": setData,
			},
		)
		assert.Nil(t, err)
		valid,err := tree.ProcessBlock(test.block)
		if !test.shouldErr {
			assert.Nil(t, err, test.description)
		}

		if test.shouldValid {
			assert.True(t, valid, test.description)
		}

		if test.validator != nil {
			test.validator(tree)
		}
	}
}
