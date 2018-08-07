package nodestore

import (
	"testing"

	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryNodeStoreCreateNode(t *testing.T) {
	ns := &MemoryNodeStore{}
	ns.Initialize()
	obj := map[string]string{"hi": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	node, err := ns.CreateNode(obj)
	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}
