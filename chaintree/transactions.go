package chaintree

import (
	"fmt"

	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
)

func NewSetOwnershipTransaction(keyAddrs []string) (*transactions.Transaction, error) {
	payload := &transactions.SetOwnershipPayload{
		Authentication: keyAddrs,
	}

	return &transactions.Transaction{
		Type:                transactions.Transaction_SETOWNERSHIP,
		SetOwnershipPayload: payload,
	}, nil
}

func NewSetDataTransaction(path string, value interface{}) (*transactions.Transaction, error) {
	sw := safewrap.SafeWrap{}
	wrappedVal := sw.WrapObject(value)
	if sw.Err != nil {
		return nil, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping data value: %v", sw.Err)}
	}

	valBytes := wrappedVal.RawData()

	return NewSetDataBytesTransaction(path, valBytes)
}

func NewSetDataBytesTransaction(path string, data []byte) (*transactions.Transaction, error) {
	payload := &transactions.SetDataPayload{
		Path:  path,
		Value: data,
	}

	return &transactions.Transaction{
		Type:           transactions.Transaction_SETDATA,
		SetDataPayload: payload,
	}, nil
}

func NewEstablishTokenTransaction(name string, max uint64) (*transactions.Transaction, error) {
	policy := &transactions.TokenMonetaryPolicy{Maximum: max}

	payload := &transactions.EstablishTokenPayload{
		Name:           name,
		MonetaryPolicy: policy,
	}

	return &transactions.Transaction{
		Type:                  transactions.Transaction_ESTABLISHTOKEN,
		EstablishTokenPayload: payload,
	}, nil
}

func NewMintTokenTransaction(name string, amount uint64) (*transactions.Transaction, error) {
	payload := &transactions.MintTokenPayload{
		Name:   name,
		Amount: amount,
	}

	return &transactions.Transaction{
		Type:             transactions.Transaction_MINTTOKEN,
		MintTokenPayload: payload,
	}, nil
}

func NewSendTokenTransaction(id, name string, amount uint64, destination string) (*transactions.Transaction, error) {
	payload := &transactions.SendTokenPayload{
		Id:          id,
		Name:        name,
		Amount:      amount,
		Destination: destination,
	}

	return &transactions.Transaction{
		Type:             transactions.Transaction_SENDTOKEN,
		SendTokenPayload: payload,
	}, nil
}

func NewReceiveTokenTransaction(sendTid string, tip []byte, treeState *signatures.TreeState, leaves [][]byte) (*transactions.Transaction, error) {
	payload := &transactions.ReceiveTokenPayload{
		SendTokenTransactionId: sendTid,
		Tip:                    tip,
		TreeState:              treeState,
		Leaves:                 leaves,
	}

	return &transactions.Transaction{
		Type:                transactions.Transaction_RECEIVETOKEN,
		ReceiveTokenPayload: payload,
	}, nil
}
