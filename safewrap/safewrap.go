package safewrap

import (
	"github.com/ipfs/go-ipld-cbor"
	multihash "github.com/multiformats/go-multihash"
)

// SafeWrap has default options for encoding/decoding objects
// and lifts errors so you can wrap an arbitrary number of objects
// if one errors, then none of the others will do anything
// the error is saved to the struct.
type SafeWrap struct {
	Err error
}

func (sf *SafeWrap) WrapObject(obj interface{}) *cbornode.Node {
	if sf.Err != nil {
		return nil
	}

	var (
		node *cbornode.Node
		err  error
	)

	switch obj.(type) {
	case cbornode.Node:
		cbor := obj.(cbornode.Node)
		node = &cbor
	case *cbornode.Node:
		node = obj.(*cbornode.Node)
	default:
		node, err = cbornode.WrapObject(obj, multihash.SHA2_256, -1)
	}

	sf.Err = err
	return node
}

func (sf *SafeWrap) Decode(data []byte) *cbornode.Node {
	if sf.Err != nil {
		return nil
	}

	node, err := cbornode.Decode(data, multihash.SHA2_256, -1)
	sf.Err = err
	return node
}
