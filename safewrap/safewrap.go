package safewrap

import (
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
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

	if cbor, ok := obj.(*cbornode.Node); ok {
		node = cbor
	} else {
		node, err = cbornode.WrapObject(obj, multihash.SHA2_256, -1)
	}

	sf.Err = err
	return node
}

func (sf *SafeWrap) Decode(data []byte) *cbornode.Node {
	if sf.Err != nil {
		return nil
	}

	hash, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		sf.Err = err
		return nil
	}
	c := cid.NewCidV1(cid.DagCBOR, hash)

	blk, err := blocks.NewBlockWithCid(data, c)
	if err != nil {
		sf.Err = err
		return nil
	}

	node, err := cbornode.DecodeBlock(blk)
	sf.Err = err
	return node.(*cbornode.Node)
}
