package safewrap

import (
	"testing"

	"github.com/ipfs/go-ipld-cbor"

	cid "github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

func init() {
	cbornode.RegisterCborType(objWithNilPointers{})
}

type objWithNilPointers struct {
	NilPointer *cid.Cid
	Other      string
	Cids       []*cid.Cid
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
				Cids: []*cid.Cid{sw.WrapObject(map[string]string{"test": "test"}).Cid()},
			},
		},
	} {
		node := sw.WrapObject(test.obj)
		_, err := node.MarshalJSON()
		assert.Nil(t, err, test.description)

		//t.Log(string(j))
		assert.Nil(t, sw.Err, test.description)
	}
}
