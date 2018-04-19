package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
)

const (
	ErrUnknownTransactionType = 1
	ErrRetryableError = 2
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

type ChainTree struct {
	Dag *dag.BidirectionalTree
	Transactors map[string]TransactorFunc
	BlockValidators []BlockValidatorFunc
}

func (ct *ChainTree) ProcessBlock(blockWithHeaders *BlockWithHeaders) (valid bool, err error) {
	// first validate the block
	for _,validator := range ct.BlockValidators {
		valid,err := validator(ct.Dag, blockWithHeaders)
		if err != nil || !valid {
			return valid,err
		}
	}

	newDag := ct.Dag.Copy()

	for _,transaction := range blockWithHeaders.Transactions {
		transactor,ok := ct.Transactors[transaction.Type]
		if !ok {
			return false, &ErrorCode{Code: ErrUnknownTransactionType, Memo: fmt.Sprintf("unknown transaction type: %v", transaction.Type)}
		}
		valid,err := transactor(newDag, transaction)
		if err != nil || !valid {
			return valid,err
		}
	}

	ct.Dag = newDag
	return true, nil
}

type Transaction struct {
	Type string `refmt:"type" json:"type" cbor:"type"`
	Payload interface{} `refmt:"payload" json:"payload" cbor:"payload"`
}

type Block struct {
	Parents []*cid.Cid `refmt:"parents" json:"parents" cbor:"parents"`
	Transactions []*Transaction `refmt:"transactions" json:"transactions" cbor:"transactions"`
}

type BlockWithHeaders struct {
	Block
	Headers map[string]interface{} `refmt:"headers" json:"headers" cbor:"headers"`
}
