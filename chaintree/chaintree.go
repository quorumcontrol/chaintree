package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
	"github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/typecaster"
)

const (
	ErrUnknownTransactionType = 1
	ErrRetryableError = 2
	ErrInvalidTree = 3
	ErrUnknown = 4

	TreeLabel = "tree"
	ChainLabel = "chain"
)

func init() {
	cbornode.RegisterCborType(RootNode{})
	cbornode.RegisterCborType(Chain{})
	cbornode.RegisterCborType(ChainEntry{})
	cbornode.RegisterCborType(BlockWithHeaders{})
	cbornode.RegisterCborType(Block{})
	cbornode.RegisterCborType(Transaction{})

	typecaster.AddType(RootNode{})
	typecaster.AddType(Chain{})
	typecaster.AddType(ChainEntry{})
	typecaster.AddType(BlockWithHeaders{})
	typecaster.AddType(Block{})
	typecaster.AddType(Transaction{})
	typecaster.AddType(cid.Cid{})
}

type CodedError interface {
	error
	GetCode() int
}

type ErrorCode struct {
	Code  int
	Memo string
}

type RootNode struct {
	Chain *cid.Cid `refmt:"chain"`
	Tree *cid.Cid `refmt:"tree"`
	Id string `refmt:"id"`
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
	// this is a string so that CID links aren't automatically adjusted
	PreviousTip string `refmt:"previousTip,omitempty" json:"previousTip,omitempty" cbor:"previousTip,omitempty"`
	BlocksWithHeaders []*cid.Cid	`refmt:"blocksWithHeaders" json:"blocksWithHeaders" cbor:"blocksWithHeaders"`
	Previous *cid.Cid `refmt:"previous" json:"previous" cbor:"previous"`
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

	root := &RootNode{}

	unmarshaledRoot := ct.Dag.Get(ct.Dag.Tip)
	if unmarshaledRoot == nil {
		return nil, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error: missing root")}
	}

	err := cbornode.DecodeInto(unmarshaledRoot.Node.RawData(), root)
	if err == nil {
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

func (ct *ChainTree) Id() (string,error) {
	root := &RootNode{}

	unmarshaledRoot := ct.Dag.Get(ct.Dag.Tip)
	if unmarshaledRoot == nil {
		return "", &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error: missing root")}
	}

	err := cbornode.DecodeInto(unmarshaledRoot.Node.RawData(), root)
	if err == nil {
		return root.Id, nil
	} else {
		return "", &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error, invalid root: %v", root)}
	}
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

	unmarshaledRoot := newTree.Get(newTree.Tip)
	if unmarshaledRoot == nil {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: "error missing root"}
	}

	root := &RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.Node.RawData(), root)
	if err != nil {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: fmt.Sprintf("error converting root: %v", err)}
	}

	if root.Tree == nil {
		return false, &ErrorCode{Code: ErrInvalidTree, Memo: "error getting treeLink"}
	}

	newTree.Tip = root.Tree

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

	chainNode := ct.Dag.Get(root.Chain)
	chainMap,err := chainNode.AsMap()
	if err != nil {
		return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting map: %v", err)}
	}

	sw := &dag.SafeWrap{}

	wrappedBlock := sw.WrapObject(blockWithHeaders)
	if sw.Err != nil {
		return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping block: %v", sw.Err)}
	}

	endLink,ok := chainMap["end"]
	if !ok {
		if tip := blockWithHeaders.Block.PreviousTip; tip != "" {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("invalid previous tip: %v, expecting nil", tip)}
		}

		//log.Printf("wrapped block Cid: %v", wrappedBlock.Cid())

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
		//log.Println("we have an end")
		link := endLink.(cid.Cid)
		endNode := ct.Dag.Get(&link)
		if endNode == nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("missing end node in chain tree")}
		}

		lastEntry := &ChainEntry{}

		err = cbornode.DecodeInto(endNode.Node.RawData(), lastEntry)

		//err = typecaster.ToType(endMap, lastEntry)
		if err != nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting lastEntry: %v", err)}
		}


		switch tip := blockWithHeaders.PreviousTip; tip{
		case unmarshaledRoot.Node.Cid().String():
			//log.Printf("previous tip of block == rootNode")
			newEntry := &ChainEntry{
				PreviousTip: ct.Dag.Tip.String(),
				BlocksWithHeaders: []*cid.Cid{wrappedBlock.Cid()},
				Previous: endNode.Node.Cid(),
			}

			entryNode := sw.WrapObject(newEntry)

			chainMap["end"] = entryNode.Cid()
			ct.Dag.AddNodes(entryNode)
			//log.Printf("setting end to: %v", entryNode.Cid().String())

			wrappedChainMap := sw.WrapObject(chainMap)

			if sw.Err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping object: %v", err)}
			}

			//log.Printf("chain map: %v", chainMap)

			err = ct.Dag.Swap(chainNode.Node.Cid(), wrappedChainMap)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping object: %v", err)}
			}
			//log.Printf("after swap of chain map")
		case lastEntry.PreviousTip:
			//log.Printf("previous tip of block == ending previousTip")

			lastEntry.BlocksWithHeaders = append(lastEntry.BlocksWithHeaders, wrappedBlock.Cid())

			entryNode := sw.WrapObject(lastEntry)
			if sw.Err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding: %v", sw.Err)}
			}
			err = ct.Dag.Swap(endNode.Node.Cid(), entryNode)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping object: %v", err)}
			}
		default:
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error, tip must be either current tip or same previousTip as last ChainEntry, tip: %v endMap: %v, rootNode: %v", tip, lastEntry.PreviousTip, unmarshaledRoot.Node.Cid())}
		}
	}
	ct.Dag.AddNodes(wrappedBlock)

	ct.Dag.Prune()

	return true, nil
}
