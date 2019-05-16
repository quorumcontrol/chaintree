package chaintree

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/signatures"
	"github.com/quorumcontrol/messages/transactions"
)

func NewSetOwnershipTransaction(keyAddrs []string) (*transactions.Transaction, error) {
	payload := &transactions.SetOwnershipPayload{
		Authentication: keyAddrs,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling SetOwnership payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETOWNERSHIP,
		Payload: payloadWrapper,
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

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling SetData payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETDATA,
		Payload: payloadWrapper,
	}, nil
}

func NewEstablishTokenTransaction(name string, max uint64) (*transactions.Transaction, error) {
	policy := &transactions.TokenMonetaryPolicy{Maximum: max}

	payload := &transactions.EstablishTokenPayload{
		Name:           name,
		MonetaryPolicy: policy,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling EstablishToken payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_ESTABLISHTOKEN,
		Payload: payloadWrapper,
	}, nil
}

func NewMintTokenTransaction(name string, amount uint64) (*transactions.Transaction, error) {
	payload := &transactions.MintTokenPayload{
		Name:   name,
		Amount: amount,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling MintToken payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_MINTTOKEN,
		Payload: payloadWrapper,
	}, nil
}

func NewSendTokenTransaction(id, name string, amount uint64, destination string) (*transactions.Transaction, error) {
	payload := &transactions.SendTokenPayload{
		Id:          id,
		Name:        name,
		Amount:      amount,
		Destination: destination,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling SendToken payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SENDTOKEN,
		Payload: payloadWrapper,
	}, nil
}

func NewReceiveTokenTransaction(sendTid string, tip []byte, sig *signatures.Signature, leaves [][]byte) (*transactions.Transaction, error) {
	payload := &transactions.ReceiveTokenPayload{
		SendTokenTransactionId: sendTid,
		Tip:       tip,
		Signature: sig,
		Leaves:    leaves,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling ReceiveToken payload: %v", err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_RECEIVETOKEN,
		Payload: payloadWrapper,
	}, nil
}
