package chaintree

import (
	"fmt"
	"reflect"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
)

const (
	ErrUnknownTransactionType = 1
	ErrRetryableError         = 2
	ErrInvalidTree            = 3
	ErrUnknown                = 4
	ErrBadHeight              = 5
	ErrBadTip                 = 6

	TreeLabel     = "tree"
	ChainLabel    = "chain"
	ChainEndLabel = "end"
)

func init() {
	cbornode.RegisterCborType(RootNode{})
	cbornode.RegisterCborType(Chain{})
	cbornode.RegisterCborType(BlockWithHeaders{})
	cbornode.RegisterCborType(Block{})
	cbornode.RegisterCborType(signatures.PublicKey{})
	cbornode.RegisterCborType(signatures.Signature{})
	cbornode.RegisterCborType(transactions.Transaction{})
	cbornode.RegisterCborType(transactions.SetDataPayload{})
	cbornode.RegisterCborType(transactions.SetOwnershipPayload{})
	cbornode.RegisterCborType(transactions.EstablishTokenPayload{})
	cbornode.RegisterCborType(transactions.TokenMonetaryPolicy{})
	cbornode.RegisterCborType(transactions.MintTokenPayload{})
	cbornode.RegisterCborType(transactions.SendTokenPayload{})
	cbornode.RegisterCborType(transactions.ReceiveTokenPayload{})
	cbornode.RegisterCborType(transactions.TokenPayload{})
	cbornode.RegisterCborType(transactions.StakePayload{})
	// protobuf generated types have internal fields with `struct{}` values
	// and cannot be marshalled without registering that type first
	// structs are commonly registered across other repos, so ensure it
	// hasn't already been registered
	structRtid := reflect.ValueOf(reflect.TypeOf(struct{}{})).Pointer()
	if _, ok := cbornode.CborAtlas.Get(structRtid); !ok {
		cbornode.RegisterCborType(struct{}{})
	}

	typecaster.AddType(RootNode{})
	typecaster.AddType(Chain{})
	typecaster.AddType(BlockWithHeaders{})
	typecaster.AddType(Block{})
	typecaster.AddType(signatures.PublicKey{})
	typecaster.AddType(signatures.Signature{})
	typecaster.AddType(transactions.Transaction{})
	typecaster.AddType(transactions.SetDataPayload{})
	typecaster.AddType(transactions.SetOwnershipPayload{})
	typecaster.AddType(transactions.EstablishTokenPayload{})
	typecaster.AddType(transactions.TokenMonetaryPolicy{})
	typecaster.AddType(transactions.MintTokenPayload{})
	typecaster.AddType(transactions.SendTokenPayload{})
	typecaster.AddType(transactions.ReceiveTokenPayload{})
	typecaster.AddType(transactions.TokenPayload{})
	// protobuf generated types have internal fields that `struct{}` and
	// cannot be cast without registering that type first
	typecaster.AddType(struct{}{})

	typecaster.AddType(cid.Cid{})
}

type CodedError interface {
	error
	GetCode() int
}

type ErrorCode struct {
	Code int
	Memo string
}

type RootNode struct {
	Chain  *cid.Cid `refmt:"chain"`
	Tree   *cid.Cid `refmt:"tree"`
	Id     string   `refmt:"id"`
	Height uint64   `refmt:"height" json:"height" cbor:"height"`
	cid    cid.Cid
}

type Block struct {
	PreviousTip  *cid.Cid                    `refmt:"previousTip,omitempty" json:"previousTip,omitempty" cbor:"previousTip,omitempty"`
	Height       uint64                      `refmt:"height" json:"height" cbor:"height"`
	Transactions []*transactions.Transaction `refmt:"transactions" json:"transactions" cbor:"transactions"`
}

type BlockWithHeaders struct {
	Block
	PreviousBlock *cid.Cid               `refmt:"previousBlock,omitempty" json:"previousBlock,omitempty" cbor:"previousBlock,omitempty"`
	Headers       map[string]interface{} `refmt:"headers" json:"headers" cbor:"headers"`
}

type Chain struct {
	End *cid.Cid `refmt:"end" json:"end" cbor:"end"`
}

func (e *ErrorCode) GetCode() int {
	return e.Code
}

func (e *ErrorCode) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Memo)
}

// TransactorFunc mutates a ChainTree and returns whether the transaction is valid
// or if there was an error processing the transactor. Errors should be retried,
// valid == false means it isn't a valid transaction.
type TransactorFunc func(chainTreeDID string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, err CodedError)

// BlockValidatorFuncs are run on the block level rather than the per transaction level
type BlockValidatorFunc func(chainTree *dag.Dag, blockWithHeaders *BlockWithHeaders) (valid bool, err CodedError)

/*
A Chain Tree is a DAG that starts with the following root node:
{
  id: string
  chain: *cidLink
  tree: *cidLink
}

The chain is for history and the tree is for data. This produces a content-addressable
data structure that has its history of change built into the merkle-DAG.

Validators are given the tip of the whole chain tree (chain and tree). Transactions are only given the
tip of the tree.
*/
type ChainTree struct {
	Dag             *dag.Dag
	Transactors     map[transactions.Transaction_Type]TransactorFunc
	BlockValidators []BlockValidatorFunc
	Metadata        interface{}
	root            *RootNode
}

func NewChainTree(dag *dag.Dag, blockValidators []BlockValidatorFunc, transactors map[transactions.Transaction_Type]TransactorFunc) (*ChainTree, error) {
	ct := &ChainTree{
		Dag:             dag,
		BlockValidators: blockValidators,
		Transactors:     transactors,
	}

	_, err := ct.getRoot()
	if err != nil {
		return nil, err
	}
	return ct, nil
}

// Id returns the ID of a chain tree (the ID node in the root of the chaintree)
func (ct *ChainTree) Id() (string, error) {
	root, err := ct.getRoot()
	if err != nil {
		return "", err
	}
	return root.Id, nil
}

// Tree returns just the tree portion of the ChainTree as a pointer to its DAG
func (ct *ChainTree) Tree() (*dag.Dag, error) {
	root, err := ct.getRoot()
	if err != nil {
		return nil, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting root node: %v", err.Error())}
	}
	if root.Tree == nil {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: "tree link is nil"}
	}
	return ct.Dag.WithNewTip(*root.Tree), nil
}

func (ct *ChainTree) ProcessBlockImmutable(blockWithHeaders *BlockWithHeaders) (newChainTree *ChainTree, valid bool, err error) {
	if blockWithHeaders == nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: "must have a block to process"}
	}

	newChainTree, err = NewChainTree(ct.Dag, ct.BlockValidators, ct.Transactors)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error creating new ChainTree: %v", err)}
	}

	// first validate the block
	for _, validator := range newChainTree.BlockValidators {
		valid, err := validator(newChainTree.Dag, blockWithHeaders)
		if err != nil || !valid {
			return nil, valid, err
		}
	}

	root, err := newChainTree.getRoot()
	if err != nil {
		return nil, false, err
	}

	if root.Tree == nil {
		return nil, false, &ErrorCode{Code: ErrInvalidTree, Memo: "error getting treeLink"}
	}

	newTree := newChainTree.Dag.WithNewTip(*root.Tree)

	for _, transaction := range blockWithHeaders.Transactions {
		transactor, ok := newChainTree.Transactors[transaction.Type]
		if !ok {
			return nil, false, &ErrorCode{Code: ErrUnknownTransactionType, Memo: fmt.Sprintf("unknown transaction type: %v", transaction.Type)}
		}

		chainTreeDID, err := ct.Id()
		if err != nil {
			return nil, false, fmt.Errorf("error getting ID of chaintree: %v", err)
		}

		newTree, valid, err = transactor(chainTreeDID, newTree, transaction)
		if err != nil || !valid {
			return nil, valid, err
		}
	}

	unmarshaledTreeTip, err := newTree.Get(newTree.Tip)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting new tree tip: %v", err)}
	}
	newTreeMap := make(map[string]interface{})
	err = cbornode.DecodeInto(unmarshaledTreeTip.RawData(), &newTreeMap)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error decoding new tree root into map: %v", err)}
	}

	newChainTree.Dag, err = newChainTree.Dag.SetAsLink([]string{TreeLabel}, newTreeMap)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error setting as link: %v", err)}
	}

	chainNode, err := newChainTree.Dag.Get(*root.Chain)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting node: %v", err)}
	}
	chain := &Chain{}
	err = cbornode.DecodeInto(chainNode.RawData(), chain)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting map: %v", err)}
	}

	/*
		if there are no chain entries, then the PreviousTip should be nil
		if there are chain entries than the tip should be either the current tip
	*/

	// if this is the first block
	if chain.End == nil {
		if height := blockWithHeaders.Block.Height; height != 0 {
			return nil, false, &ErrorCode{Code: ErrBadHeight, Memo: fmt.Sprintf("first block must have a height of 0, had: %d", height)}
		}
		if tip := blockWithHeaders.Block.PreviousTip; tip != nil {
			return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("invalid previous tip: %v, expecting nil", tip)}
		}

		wrappedBlock, err := newChainTree.Dag.CreateNode(blockWithHeaders)
		if err != nil {
			return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping block: %v", err)}
		}

		endCid := wrappedBlock.Cid()
		chain.End = &endCid

		newChainTree.Dag, err = newChainTree.Dag.SetAsLink([]string{ChainLabel}, chain)
		if err != nil {
			return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error updating: %v", err)}
		}

		newChainTree.Dag, err = newChainTree.Dag.Set([]string{"height"}, uint64(0))
		if err != nil {
			return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error setting height: %v", err)}
		}
		return newChainTree, true, nil
	}

	// otherwise we have an existing chain in this chaintree

	endNode, err := newChainTree.Dag.Get(*chain.End)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting end node: %v", err)}
	}
	if endNode == nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("missing end node in chain tree")}
	}

	lastEntry := &BlockWithHeaders{}

	err = cbornode.DecodeInto(endNode.RawData(), lastEntry)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting lastEntry: %v", err)}
	}

	if tip := blockWithHeaders.PreviousTip; tip == nil || !tip.Equals(root.cid) {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error, tip must be current tip, tip: %v endMap: %v, rootNode: %v", tip, lastEntry.PreviousTip, root.cid)}
	}

	if height := blockWithHeaders.Block.Height; height != (lastEntry.Height + uint64(1)) {
		return nil, false, &ErrorCode{Code: ErrBadHeight, Memo: fmt.Sprintf("block must have a height of %d, had: %d", (lastEntry.Height + uint64(1)), height)}
	}

	blockWithHeaders.PreviousBlock = chain.End

	wrappedBlock, err := newChainTree.Dag.CreateNode(blockWithHeaders)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping block: %v", err)}
	}

	newEnd := wrappedBlock.Cid()
	chain.End = &newEnd

	newChainTree.Dag, err = newChainTree.Dag.SetAsLink([]string{ChainLabel}, chain)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping object: %v", err)}
	}
	newChainTree.Dag, err = newChainTree.Dag.Set([]string{"height"}, blockWithHeaders.Block.Height)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error setting root height: %v", err)}
	}
	return newChainTree, true, nil
}

// ProcessBlock takes a signed block, runs all the validators and if those succeeds
// it runs the transactors. If all transactors succeed, then the tree
// of the Chain Tree is updated and the block is appended to the chain part
// of the Chain Tree
func (ct *ChainTree) ProcessBlock(blockWithHeaders *BlockWithHeaders) (valid bool, err error) {
	newChainTree, valid, err := ct.ProcessBlockImmutable(blockWithHeaders)
	if err != nil || !valid {
		return valid, err
	}

	ct.Dag = newChainTree.Dag

	return true, nil
}

func (ct *ChainTree) getRoot() (*RootNode, error) {
	if ct.root != nil && ct.root.cid.Equals(ct.Dag.Tip) {
		return ct.root, nil
	}
	unmarshaledRoot, err := ct.Dag.Get(ct.Dag.Tip)
	if unmarshaledRoot == nil || err != nil {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error: missing root: %v", err)}
	}

	root := &RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.RawData(), root)
	if err != nil {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error converting root: %v", err)}
	}
	root.cid = ct.Dag.Tip
	ct.root = root
	return root, nil
}
