package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
)

const (
	ErrUnknownTransactionType = 1
	ErrRetryableError = 2
	ErrInvalidTree = 3
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

	

	return true, nil
}

type Transaction struct {
	Type string `refmt:"type" json:"type" cbor:"type"`
	Payload interface{} `refmt:"payload" json:"payload" cbor:"payload"`
}

type Block struct {
	PreviousTip *cid.Cid `refmt:"parents" json:"parents" cbor:"parents"`
	Transactions []*Transaction `refmt:"transactions" json:"transactions" cbor:"transactions"`
}

type BlockWithHeaders struct {
	Block
	Headers map[string]interface{} `refmt:"headers" json:"headers" cbor:"headers"`
}

type BlockWithHeadersId *cid.Cid
type BlockId *cid.Cid
type ChainEntryId *cid.Cid

type Chain struct {
	Genesis ChainEntryId `refmt:"genesis" json:"genesis" cbor:"genesis"`
	End ChainEntryId `refmt:"end" json:"end" cbor:"end"`
	dag *dag.BidirectionalTree
}

type ChainEntry struct {
	PreviousTip *cid.Cid
	BlocksWithHeaders []BlockWithHeadersId
	Previous ChainEntryId
	Next ChainEntryId
}

