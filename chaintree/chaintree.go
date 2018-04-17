package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
)


type ErrorCode struct {
	Code  int
	Memo string
}

func (e *ErrorCode) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Memo)
}


// TransactorFunc mutates a  ChainTree and returns whether the transaction is valid
// or if there was an error processing the transactor. Errors should be retried,
// valid means it isn't a valid transaction
type TransactorFunc func(tree *ChainTree) (valid bool, err error)

type ChainTree struct {
	Dag *dag.BidirectionalTree
	Transactors map[string]TransactorFunc
}

type Transaction struct {
	Type string
	Payload interface{}
}

type Block struct {
	Parents map[string]*cid.Cid
	Transactions []*cid.Cid
}

