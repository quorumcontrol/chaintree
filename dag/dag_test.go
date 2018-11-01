package dag

import (
	"testing"

	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeepDag(t *testing.T) *Dag {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child": true})
	root := sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, deepChild, child)
	require.Nil(t, err)
	return dag
}

func TestDagNodes(t *testing.T) {
	dag := newDeepDag(t)
	nodes, err := dag.Nodes()
	assert.Nil(t, err)
	assert.Len(t, nodes, 3)
}

func TestDagResolve(t *testing.T) {
	dag := newDeepDag(t)
	val, remain, err := dag.Resolve([]string{"child", "deepChild", "deepChild"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, true, val)
}

func TestDagSet(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	unlinked := sw.WrapObject(map[string]interface{}{
		"unlinked": true,
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"test"}, "bob")
	assert.Nil(t, err)

	val, _, err := dag.Resolve([]string{"test"})

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// test works with a CID
	dag.AddNodes(unlinked)

	dag, err = dag.Set([]string{"test"}, unlinked.Cid())
	assert.Nil(t, err)

	val, _, err = dag.Resolve([]string{"test", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)

	// test works in non-existant path

	path := []string{"child", "non-existant-nested", "objects", "test"}
	dag, err = dag.Set(path, "bob")
	assert.Nil(t, err)

	val, _, err = dag.Resolve(path)

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// Test sibling of existing path
	siblingPath := []string{"child", "non-existant-nested", "objects", "siblingtest"}
	dag, err = dag.Set(siblingPath, "sue")
	assert.Nil(t, err)

	// original sibling is still available
	val, _, err = dag.Resolve(path)
	assert.Equal(t, "bob", val)

	siblingVal, _, err := dag.Resolve(siblingPath)

	assert.Nil(t, err)
	assert.Equal(t, "sue", siblingVal)
}

func TestDagSetAsLink(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	unlinked := map[string]interface{}{
		"unlinked": true,
	}

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	dag, err = dag.SetAsLink([]string{"child", "grandchild", "key"}, unlinked)
	assert.Nil(t, err)
	val, _, err := dag.Resolve([]string{"child", "grandchild", "key", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)

	unlinked2 := map[string]interface{}{
		"unlinked2": false,
	}

	dag, err = dag.SetAsLink([]string{"child", "grandchild", "key", "unlinkedsibling"}, unlinked2)
	assert.Nil(t, err)

	val, _, err = dag.Resolve([]string{"child", "grandchild", "key", "unlinkedsibling", "unlinked2"})
	assert.Nil(t, err)
	assert.Equal(t, false, val)
}

func TestDagInvalidSet(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"test"}, map[string]interface{}{
		"child1": "1",
		"child2": "2",
	})

	assert.NotNil(t, err)
}

func TestDagGet(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)
	n, err := dag.Get(child.Cid())
	require.Nil(t, err)
	assert.Equal(t, child.Cid().String(), n.Cid().String())
}

func TestDagDump(t *testing.T) {
	// Not really a test here, but do call it just to make sure no panics
	dag := newDeepDag(t)
	dag.Dump()
}

func TestDagWithNewTip(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	newDag := dag.WithNewTip(child.Cid())
	assert.Equal(t, newDag.Tip.String(), child.Cid().String())
	nodes, err := newDag.Nodes()
	require.Nil(t, err)
	assert.Len(t, nodes, 1)
}

func TestDagUpdate(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	require.Nil(t, sw.Err)
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	dag, err = dag.Update(child.Cid(), map[string]interface{}{"name": "changed"})

	val, remain, err := dag.Resolve([]string{"child", "name"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, "changed", val)
}

func TestDagSwap(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	require.Nil(t, sw.Err)
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, child)
	require.Nil(t, err)

	newChildNode := sw.WrapObject(map[string]interface{}{
		"name": "changed",
	})

	dag, err = dag.Swap(child.Cid(), newChildNode)

	val, remain, err := dag.Resolve([]string{"child", "name"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, "changed", val)
}
