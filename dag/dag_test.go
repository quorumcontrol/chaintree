package dag

import (
	"testing"

	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
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

func TestDagNodesForPath(t *testing.T) {
	dag := newDeepDag(t)
	nodes, err := dag.NodesForPath([]string{"child", "deepChild"})
	require.Nil(t, err)
	assert.Len(t, nodes, 3)
	allNodes, _ := dag.Nodes()
	for i, node := range allNodes {
		assert.Equal(t, node.RawData(), nodes[i].RawData())
	}
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

	// test top level sibling
	dag, err = dag.Set([]string{"test2"}, "alice")
	assert.Nil(t, err)

	val, _, err = dag.Resolve([]string{"test"})
	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	val2, _, err := dag.Resolve([]string{"test2"})
	assert.Nil(t, err)
	assert.Equal(t, "alice", val2)

	// test works with a CID
	err = dag.AddNodes(unlinked)
	require.Nil(t, err)

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
	require.Nil(t, err)
	assert.Equal(t, "bob", val)

	siblingVal, _, err := dag.Resolve(siblingPath)

	assert.Nil(t, err)
	assert.Equal(t, "sue", siblingVal)

	// Test sibling of partially existing path
	partiallyExistingPath := []string{"child", "non-existant-nested", "other-objects", "nestedtest"}
	dag, err = dag.Set(partiallyExistingPath, "carol")
	assert.Nil(t, err)

	// original sibling is still available
	val, _, err = dag.Resolve(path)
	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// second sibling is still available
	siblingVal, _, err = dag.Resolve(siblingPath)
	assert.Nil(t, err)
	assert.Equal(t, "sue", siblingVal)

	// check partially existing path set
	partiallyExistingVal, _, err := dag.Resolve(partiallyExistingPath)
	assert.Nil(t, err)
	assert.Equal(t, "carol", partiallyExistingVal)
}

// Verify that when setting a nested value, clobbering of an ancestor is not allowed.
func TestDagSetNestedNoClobber(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	assert.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"outer"}, "flat")
	assert.Nil(t, err)

	_, err = dag.Set([]string{"outer", "inner"}, "nested")
	assert.NotNil(t, err)
}

// Verify that when setting a simple value, clobbering of a complex value is not allowed.
func TestDagSetNoClobberComplex(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.SetAsLink([]string{"complex"}, map[string]interface{}{
		"complex": true,
	})
	require.Nil(t, err)

	_, err = dag.Set([]string{"complex"}, "simple")
	require.NotNil(t, err)
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

// Verify that when setting a nested value as a link, clobbering of an ancestor is not allowed.
func TestDagSetAsLinkNestedNoClobber(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"outer"}, "flat")
	require.Nil(t, err)

	unlinked := map[string]interface{}{
		"unlinked": true,
	}

	_, err = dag.SetAsLink([]string{"outer", "inner"}, unlinked)
	require.NotNil(t, err)
}

// Verify that when setting a complex value, clobbering of a simple value is not allowed.
func TestDagSetAsLinkNoClobberSimple(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"simple"}, "simple")
	require.Nil(t, err)

	_, err = dag.SetAsLink([]string{"simple"}, map[string]interface{}{
		"complex": true,
	})
	require.NotNil(t, err)
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

	_, err = dag.Set([]string{"test"}, map[string]interface{}{
		"child1": "1",
		"child2": "2",
	})
	require.NotNil(t, err)
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

	intermediary := sw.WrapObject(map[string]interface{}{
		"name":   "intermediary",
		"child2": child.Cid(),
	})

	root := sw.WrapObject(map[string]interface{}{
		"name":   "root",
		"child1": intermediary.Cid(),
	})

	require.Nil(t, sw.Err)
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, intermediary, child)
	require.Nil(t, err)

	dag, err = dag.Update([]string{"child1", "child2"}, map[string]interface{}{"name": "changed"})
	require.Nil(t, err)

	val, remain, err := dag.Resolve([]string{"child1", "child2", "name"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, "changed", val)
}
