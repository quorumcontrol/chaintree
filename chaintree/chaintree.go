package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
	"github.com/quorumcontrol/chaintree/typecaster"
	"log"
)

const (
	ErrUnknownTransactionType = 1
	ErrRetryableError = 2
	ErrInvalidTree = 3
	ErrUnknown = 4

	TreeLabel = "tree"
	ChainLabel = "chain"
)

type CodedError interface {
	error
	GetCode() int
}

type ErrorCode struct {
	Code  int
	Memo string
}

func init() {
	typecaster.AddType(Chain{})
	typecaster.AddType(ChainEntry{})
	typecaster.AddType(BlockWithHeaders{})
	typecaster.AddType(cid.Cid{})
}


func (e *ErrorCode) GetCode() int {
	return e.Code
}

func (e *ErrorCode) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Memo)
}

// TransactorFunc mutates a  ChainTree and returns whether the transaction is valid
// or if there was an error processing the transactor. Errors should be retried,
// valid means it isn't a valid transaction
type TransactorFunc func(tree *dag.BidirectionalTree, transaction *Transaction) (valid bool, err CodedError)

// Validator funcs are run
type BlockValidatorFunc func(tree *dag.BidirectionalTree, blockWithHeaders *BlockWithHeaders) (valid bool, err CodedError)


/*
A Chain Tree is a DAG that starts with the following root node:
{
  chain: *cidLink
  tree: *cidLink
}

The chain is for history and the tree is for data. This produces a content-addressable
data structure that has its history of change built into the merkle-DAG.

Validators are given the tip of the whole chain tree (chain and tree). Transactions are only given the
tip of the tree.
 */
type ChainTree struct {
	Dag *dag.BidirectionalTree
	Transactors map[string]TransactorFunc
	BlockValidators []BlockValidatorFunc
	Metadata interface{}
}

func NewChainTree(dag *dag.BidirectionalTree, blockValidators []BlockValidatorFunc, transactors map[string]TransactorFunc) (*ChainTree, error) {
	ct := &ChainTree{
		Dag: dag,
		BlockValidators: blockValidators,
		Transactors: transactors,
	}

	root,err := ct.Dag.Get(ct.Dag.Tip).AsJSONish()
	if err != nil {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error: missing root %v", err)}
	}

	if len(root) == 0 || (hasKey(root, ChainLabel) && hasKey(root, TreeLabel)) {
		return ct, nil
	} else {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error, invalid root: %v", root)}
	}
}

func hasKey(m map[string]interface{}, k string) bool {
	_,ok := m[k]
	if ok {
		return true
	}
	return false
}

func (ct *ChainTree) ProcessBlock(blockWithHeaders *BlockWithHeaders) (valid bool, err error) {
	if blockWithHeaders == nil {
		return false, &ErrorCode{Code: ErrUnknown, Memo: "must have a block to process"}
	}

	// first validate the block
	for _,validator := range ct.BlockValidators {
		valid,err := validator(ct.Dag, blockWithHeaders)
		if err != nil || !valid {
			return valid,err
		}
	}

	newTree := ct.Dag.Copy()

	rootNode := newTree.Get(newTree.Tip)
	if rootNode == nil {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: "error missing root"}
	}

	root,err := rootNode.AsMap()
	if err != nil {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error converting root: %v", err)}
	}

	treeLink,ok := root[TreeLabel]
	if !ok {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: "error getting treeLink"}
	}

	newTree.Tip = treeLink.(*cid.Cid)

	for _,transaction := range blockWithHeaders.Transactions {
		transactor,ok := ct.Transactors[transaction.Type]
		if !ok {
			return false, &ErrorCode{Code: ErrUnknownTransactionType, Memo: fmt.Sprintf("unknown transaction type: %v", transaction.Type)}
		}
		valid,err := transactor(newTree, transaction)
		if err != nil || !valid {
			return valid,err
		}
	}

	ct.Dag.SetAsLink([]string{TreeLabel}, newTree)
	// now add the block itself

	/*
	if there are no chain entries, then the PreviousTip should be nil
	if there are chain entries than the tip should be either the current tip OR
		the PreviousTip of the last ChainEntry
	 */

	chainNode := ct.Dag.Get(root[ChainLabel].(*cid.Cid))
	chainMap,err := chainNode.AsMap()
	if err != nil {
		return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting map: %v", err)}
	}

	sw := &dag.SafeWrap{}

	wrappedBlock := sw.WrapObject(blockWithHeaders)

	endLink,ok := chainMap["end"]
	if !ok {
		if tip := blockWithHeaders.Block.PreviousTip; tip != "" {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("invalid previous tip: %v, expecting nil", tip)}
		}

		log.Printf("wrapped block Cid: %v", wrappedBlock.Cid())

		lastEntry := &ChainEntry{
			PreviousTip: "",
			BlocksWithHeaders: []*cid.Cid{wrappedBlock.Cid()},
		}
		entryNode := sw.WrapObject(lastEntry)
		chainMap["end"] = entryNode.Cid()
		newChainNode := sw.WrapObject(chainMap)

		ct.Dag.AddNodes(entryNode)
		ct.Dag.Swap(chainNode.Node.Cid(), newChainNode)

	} else {
		log.Println("we have an end")
		endNode := ct.Dag.Get(endLink.(*cid.Cid))
		if endNode == nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("missing end node in chain tree")}
		}

		endMap,err := endNode.AsMap()
		if err != nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting map: %v", err)}
		}

		lastEntry := &ChainEntry{}

		err = typecaster.ToType(endMap, lastEntry)
		if err != nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting lastEntry: %v", err)}
		}

		switch tip := blockWithHeaders.PreviousTip; tip{
		case rootNode.Node.Cid().String():
			log.Printf("previous tip of block == rootNode")
			newEntry := &ChainEntry{
				PreviousTip: ct.Dag.Tip.String(),
				BlocksWithHeaders: []*cid.Cid{wrappedBlock.Cid()},
				Previous: endNode.Node.Cid(),
			}

			entryNode := sw.WrapObject(newEntry)

			chainMap["end"] = entryNode.Cid()
			ct.Dag.AddNodes(entryNode)
			log.Printf("setting end to: %v", entryNode.Cid().String())

			wrappedChainMap := sw.WrapObject(chainMap)

			if sw.Err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping object: %v", err)}
			}

			log.Printf("chain map: %v", chainMap)

			err = ct.Dag.Swap(chainNode.Node.Cid(), wrappedChainMap)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping object: %v", err)}
			}
			log.Printf("after swap of chain map")
		case endMap["previousTip"].(string):
			log.Printf("previous tip of block == ending previousTip")

			lastEntry.BlocksWithHeaders = append(lastEntry.BlocksWithHeaders, wrappedBlock.Cid())
			entryNode := sw.WrapObject(lastEntry)
			err = ct.Dag.Swap(endNode.Node.Cid(), entryNode)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping object: %v", err)}
			}
		default:
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error, tip must be either current tip or same previousTip as last ChainEntry, tip: %v endMap: %v, rootNode: %v", tip, endMap["previousTip"], rootNode.Node.Cid())}
		}
	}

	//ct.Dag.Prune()

	return true, nil
}

type Transaction struct {
	Type string `refmt:"type" json:"type" cbor:"type"`
	Payload interface{} `refmt:"payload" json:"payload" cbor:"payload"`
}

type Block struct {
	// this is an interface because nil pointers aren't encoded correctly
	PreviousTip string `refmt:"previousTip,omitempty" json:"previousTip,omitempty" cbor:"previousTip,omitempty"`
	Transactions []*Transaction `refmt:"transactions" json:"transactions" cbor:"transactions"`
}

type BlockWithHeaders struct {
	Block
	Headers map[string]interface{} `refmt:"headers" json:"headers" cbor:"headers"`
}

type Chain struct {
	Genesis *cid.Cid `refmt:"genesis" json:"genesis" cbor:"genesis"`
	End *cid.Cid `refmt:"end" json:"end" cbor:"end"`
}

type ChainEntry struct {
	// this is an interface because nil pointers aren't encoded correctly
	PreviousTip string `refmt:"previousTip,omitempty" json:"previousTip,omitempty" cbor:"previousTip,omitempty"`
	BlocksWithHeaders []*cid.Cid	`refmt:"blocksWithHeaders" json:"blocksWithHeaders" cbor:"blocksWithHeaders"`
	// this is an interface because nil pointers aren't encoded correctly
	Previous interface{} `refmt:"previous" json:"previous" cbor:"previous"`
}

