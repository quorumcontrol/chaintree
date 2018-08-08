package nodestore

import (
	"testing"

	"github.com/ipfs/go-cid"

	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryNodeStoreCreateNode(t *testing.T) {
	mns := &MemoryNodeStore{}
	mns.Initialize()
	obj := map[string]string{"hi": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	node, err := mns.CreateNode(obj)
	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func TestMemoryNodeStoreGetNode(t *testing.T) {
	mns := &MemoryNodeStore{}
	mns.Initialize()
	obj := map[string]string{"hi": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	mns.CreateNode(obj)

	node, err := mns.GetNode(testNode.Cid())

	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func TestMemoryNodeStoreGetReferences(t *testing.T) {
	mns := &MemoryNodeStore{}
	mns.Initialize()
	sw := &safewrap.SafeWrap{}

	child := map[string]string{"hi": "value"}
	childNode := sw.WrapObject(child)
	root := map[string]*cid.Cid{"child": childNode.Cid()}
	rootNode := sw.WrapObject(root)

	_, err := mns.CreateNode(child)
	require.Nil(t, err)
	_, err = mns.CreateNode(root)
	require.Nil(t, err)

	refs, err := mns.GetReferences(childNode.Cid())

	require.Len(t, refs, 1)

	assert.Equal(t, refs[0].String(), rootNode.Cid().String())
}

func TestMemoryNodeStoreUpdateNode(t *testing.T) {
	mns := &MemoryNodeStore{}
	mns.Initialize()
	sw := &safewrap.SafeWrap{}

	child := map[string]string{"hi": "value"}
	childNode := sw.WrapObject(child)

	newChild := map[string]string{"hi": "newValue"}
	newChildNode := sw.WrapObject(newChild)

	expectedNewRoot := map[string]*cid.Cid{"child": newChildNode.Cid()}
	expectedNewRootNode := sw.WrapObject(expectedNewRoot)

	require.Nil(t, sw.Err)

	_, err := mns.CreateNode(child)
	require.Nil(t, err)

	root := map[string]*cid.Cid{"child": childNode.Cid()}
	_, err = mns.CreateNode(root)
	require.Nil(t, err)

	updated, tips, err := mns.UpdateNode(childNode.Cid(), newChild)
	require.Nil(t, err)
	require.Len(t, tips, 1)

	assert.Equal(t, tips[0].String(), expectedNewRootNode.Cid().String())
	assert.Equal(t, updated.Cid().String(), newChildNode.Cid().String())
}
