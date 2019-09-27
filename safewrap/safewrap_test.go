package safewrap

import (
	"testing"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	multihash "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	cbornode.RegisterCborType(objWithNilPointers{})
}

type objWithNilPointers struct {
	NilPointer *cid.Cid
	Other      string
	Cids       []cid.Cid
}

func TestSafeWrap_WrapObject(t *testing.T) {
	sw := &SafeWrap{}

	for _, test := range []struct {
		description string
		obj         *objWithNilPointers
	}{
		{
			description: "an object with an empty cid",
			obj:         &objWithNilPointers{Other: "something"},
		},
		{
			description: "an object with an array of CIDs",
			obj: &objWithNilPointers{
				Cids: []cid.Cid{sw.WrapObject(map[string]string{"test": "test"}).Cid()},
			},
		},
	} {
		node := sw.WrapObject(test.obj)
		require.Nil(t, sw.Err)
		_, err := node.MarshalJSON()
		assert.Nil(t, err, test.description)

		assert.Nil(t, sw.Err, test.description)
	}

	cbor, err := cbornode.WrapObject("foo", multihash.SHA2_256, -1)
	require.Nil(t, err)

	wrappedCbor := sw.WrapObject(cbor)
	require.Nil(t, sw.Err)
	assert.Equal(t, cbor, wrappedCbor)
}

func TestSafeWrap_Decode(t *testing.T) {
	sw := &SafeWrap{}

	for _, test := range []struct {
		description string
		obj         interface{}
	}{
		{
			description: "an object with an empty cid",
			obj:         &objWithNilPointers{Other: "something"},
		},
		{
			description: "a large uint64",
			obj:         uint64(12348347582345823458),
		},
	} {
		node := sw.WrapObject(test.obj)
		require.Nil(t, sw.Err)

		new := sw.Decode(node.RawData())
		assert.Nil(t, sw.Err, test.description)
		assert.True(t, node.Cid().Equals(new.Cid()))
	}

}
