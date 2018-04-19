package chaintree

import (
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/ipfs/go-cid"
	"fmt"
)

const (
	BLSGroupSig = 0
	Ed25119 = 1
	Secp256k1 = 2
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
type TransactorFunc func(tree *ChainTree, transaction *Transaction) (valid bool, err CodedError)

// Validator funcs are run
type BlockValidatorFunc func(tree *ChainTree, signedBlock *SignedBlock) (valid bool, err CodedError)

type ChainTree struct {
	Dag *dag.BidirectionalTree
	Transactors map[string]TransactorFunc
	BlockValidators []BlockValidatorFunc
}

type Transaction struct {
	Type string `refmt:"type" json:"type" cbor:"type"`
	Payload interface{} `refmt:"payload" json:"payload" cbor:"payload"`
}

type Block struct {
	Parents map[string]*cid.Cid
	Transactions []*cid.Cid
}

type SignedBlock struct {
	Block
	Signatures []*Signature
}

type Signature struct {
	Creator string
	Signers []bool
	Signature []byte
	SignatureType int
	Memo []byte
}

type PublicKey struct {
	Type int
	PublicKey []byte
}
