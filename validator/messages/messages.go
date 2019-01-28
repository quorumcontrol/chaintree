package messages

import (
	"fmt"
	"reflect"
	"strings"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
)

var registry = make(map[string]reflect.Type)

func typeStringToRegistry(str string) string {
	str = strings.Split(str, ".")[1]
	str = strings.ToLower(str)
	return str
}
func register(obj interface{}) {
	typ := reflect.TypeOf(obj)
	str := typeStringToRegistry(typ.String())
	registry[str] = typ
	cbornode.RegisterCborType(obj)
}

func init() {
	register(Any{})
	register(GetNode{})
	register(GetNodeResponse{})
	register(Start{})
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
		Type:    typeStringToRegistry(typeStr),
		Payload: payload.RawData(),
	}, nil
}

func FromSerialized(bits []byte) (interface{}, error) {
	any := &Any{}
	err := cbornode.DecodeInto(bits, any)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return FromAny(any)
}

func FromAny(any *Any) (interface{}, error) {
	typ, ok := registry[any.Type]
	if !ok {
		return nil, fmt.Errorf("error, unknown type %s", any.Type)
	}
	ptr := reflect.New(typ).Interface()
	err := cbornode.DecodeInto(any.Payload, ptr)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return ptr, nil
}

type Start struct {
	Tip   cid.Cid
	Nodes map[string][]byte
}

type Finished struct {
	Result string
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
