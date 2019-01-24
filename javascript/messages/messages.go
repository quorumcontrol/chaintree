package messages

import (
	"fmt"
	"reflect"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
)

var registry = make(map[string]reflect.Type)

func register(obj interface{}) {
	typ := reflect.TypeOf(obj)
	registry[typ.String()] = typ
}

func init() {
	register(Any{})
	register(GetNode{})
	register(GetNodeResponse{})
	register(Start{})
	cbornode.RegisterCborType(Start{})
	register(Finished{})
}

type Any struct {
	Type    string
	Payload []byte
}

func ToAny(other interface{}) (*Any, error) {
	typeStr := reflect.TypeOf(other).String()
	sw := &safewrap.SafeWrap{}
	payload := sw.WrapObject(other)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}

	return &Any{
		Type:    typeStr,
		Payload: payload.RawData(),
	}, nil
}

type Start struct {
	Tip   cid.Cid
	Nodes [][]byte
}

type Finished struct {
	Result []byte
}

type GetNode struct {
	Cid cid.Cid
}

type GetNodeResponse struct {
	Cid  cid.Cid
	Node []byte
}

// type GetTip struct {
// 	Did string
// }

// type GetTipResponse struct {
// 	Did string
// 	Cid cid.Cid
// }
