package chaintree

import (
	"testing"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/stretchr/testify/assert"
	"github.com/quorumcontrol/chaintree/typecaster"
	"fmt"
	"strings"
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

func TestIsSigned(t *testing.T) {
	sw := &dag.SafeWrap{}


	data := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	root := sw.WrapObject(map[string]interface{}{
		"something": "ishere",
		"data": data.Cid(),
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
						&Transaction{
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
				val,_,err := tree.Dag.Resolve(strings.Split("down/in/the/thing", "/"))
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
						&Transaction{
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
				_,_,err := tree.Dag.Resolve(strings.Split("down/in/the/thing", "/"))
				assert.Equal(t, dag.ErrMissingPath, err.(*dag.ErrorCode).Code)
			},
		},
		{
			description: "a block that fails a validation",
			shouldValid: false,
			shouldErr: false,
			block: &BlockWithHeaders{
				Block: Block{
					Transactions: []*Transaction{
						&Transaction{
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
				_,_,err := tree.Dag.Resolve(strings.Split("down/in/the/thing", "/"))
				assert.Equal(t, dag.ErrMissingPath, err.(*dag.ErrorCode).Code)
			},
		},
	} {
		tree := &ChainTree{
			Dag: dag.NewBidirectionalTree(root.Cid(), root,data),
			BlockValidators: []BlockValidatorFunc{hasCoolHeader},
			Transactors: map[string]TransactorFunc{
				"SET_DATA": setData,
			},
		}
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
