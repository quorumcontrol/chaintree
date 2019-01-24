package messages

import (
	"testing"

	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAny(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	start := map[string]string{"hi": "hi"}
	cbn := sw.WrapObject(start)

	obj := &Start{
		Tip:   cbn.Cid(),
		Nodes: [][]byte{[]byte{byte(0)}},
	}
	toAny, err := ToAny(obj)
	require.Nil(t, err)
	assert.Equal(t, "start", toAny.Type)
}
