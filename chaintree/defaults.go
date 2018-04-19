package chaintree

import (
	"strings"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/polydawn/refmt/obj"
	"github.com/polydawn/refmt/obj/atlas"
)


const (
	TransTypeAddData = "ADD_DATA"
	//UPDATE_OWNERSHIP Transaction_TransactionType = 1
	//MINT_COIN        Transaction_TransactionType = 2
	//SEND_COIN        Transaction_TransactionType = 3
	//RECEIVE_COIN     Transaction_TransactionType = 4
	//BALANCE          Transaction_TransactionType = 5
)

func IsSigned(tree *ChainTree, signedBlock *SignedBlock) (bool, CodedError) {
	owners,remain,err := tree.Dag.Resolve(strings.Split("_qc/authentication", "/"))
	if err != nil {
		if err.(*dag.ErrorCode).Code == dag.ErrMissingPath {
			return false, &ErrorCode{Code: dag.ErrMissingPath, Memo: "error getting path"}
		}
	}

	if len(remain) != 0 {
		return false, &ErrorCode{Code: dag.ErrMissingPath, Memo: "error getting path"}
	}

	obj.NewMarshaller(atlas.MustBuild())

	if len(owners.([]interface{})) == 0 {
		return false, &ErrorCode{Code: dag.ErrBadInput, Memo: "missing owners"}
	}


	return true, nil
}

