package chaintree

import (
	"testing"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/stretchr/testify/assert"
	"github.com/ipfs/go-ipld-cbor"
)

func TestToType(t *testing.T) {
	trans := &Transaction{
		Type: TransTypeAddData,
		Payload: map[string]interface{}	{
			"path": "child/is/good",
			"value": "good",
		},
	}

	sw := dag.SafeWrap{}
	obj := sw.WrapObject(trans)

	assert.Nil(t, sw.Err)

	jsonish := make(map[string]interface{})
	err := cbornode.DecodeInto(obj.RawData(), &jsonish)
	t.Log(jsonish)

	assert.Nil(t, err)

	newTrans := &Transaction{}

	err = ToType(jsonish, newTrans)

	assert.Nil(t, err)

	assert.Equal(t, trans, newTrans)
}
