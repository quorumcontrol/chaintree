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

func newDeepAndWideDag(t *testing.T) *Dag {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child1 := sw.WrapObject(map[string]interface{}{"deepChild1": deepChild.Cid(), "child1": true})
	child2 := sw.WrapObject(map[string]interface{}{"deepChild2": deepChild.Cid(), "child2": true})
	root := sw.WrapObject(map[string]interface{}{"child1": child1.Cid(), "child2": child2.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, deepChild, child1, child2)
	require.Nil(t, err)
	return dag
}

func TestDagNodes(t *testing.T) {
	dag := newDeepDag(t)
	nodes, err := dag.Nodes()
	assert.Nil(t, err)
	assert.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t)
	nodes, err = dag.Nodes()
	assert.Nil(t, err)
	// Removes uniques
	assert.Len(t, nodes, 4)
}

func TestDagResolve(t *testing.T) {
	dag := newDeepDag(t)
	val, remain, err := dag.Resolve([]string{"child", "deepChild", "deepChild"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, true, val)
}

// Test that the ResolveAt method can operate with a tip that need not be current.
func TestDagResolveAt(t *testing.T) {
	dag := newDeepDag(t)
	oldTip := dag.Tip
	dag, err := dag.Set([]string{"child", "value"}, true)
	require.Nil(t, err)

	val, remain, err := dag.ResolveAt(oldTip, []string{"child", "deepChild", "deepChild"})
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, true, val)

	missingVal, remain, err := dag.ResolveAt(oldTip, []string{"child", "value"})
	require.Nil(t, err)
	require.Len(t, remain, 1)
	require.Equal(t, remain, []string{"value"})
	require.Nil(t, missingVal)
}

func TestOrderedNodesForPath(t *testing.T) {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child": true})
	root := sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root, deepChild, child)
	require.Nil(t, err)

	nodes, err := dag.orderedNodesForPath([]string{"child", "deepChild"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	require.Equal(t, nodes[0].RawData(), root.RawData())
	require.Equal(t, nodes[1].RawData(), child.RawData())
	require.Equal(t, nodes[2].RawData(), deepChild.RawData())
}

func TestDagNodesForPath(t *testing.T) {
	dag := newDeepDag(t)
	nodes, err := dag.NodesForPath([]string{"child", "deepChild"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)
	allNodes, err := dag.Nodes()
	require.Nil(t, err)
	require.Len(t, allNodes, 3)

	nodeBytes := make([][]byte, len(allNodes))
	for i, n := range allNodes {
		nodeBytes[i] = n.RawData()
	}

	for _, n := range nodes {
		require.Contains(t, nodeBytes, n.RawData())
	}

	dag = newDeepAndWideDag(t)
	nodes, err = dag.NodesForPath([]string{"child2", "deepChild2"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t)
	nodes, err = dag.NodesForPath([]string{"child2"})
	require.Nil(t, err)
	require.Len(t, nodes, 2)
}

func TestDagNodesForPathWithDecendants(t *testing.T) {
	dag := newDeepAndWideDag(t)
	nodes, err := dag.NodesForPathWithDecendants([]string{"child2", "deepChild2"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t)
	nodes2, err := dag.NodesForPathWithDecendants([]string{"child2"})
	require.Nil(t, err)
	require.Len(t, nodes2, 3)

	nodeBytes := make([][]byte, len(nodes))
	for i, n := range nodes {
		nodeBytes[i] = n.RawData()
	}

	for _, node := range nodes2 {
		require.Contains(t, nodeBytes, node.RawData())
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
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"outer"}, "flat")
	require.Nil(t, err)

	_, err = dag.Set([]string{"outer", "inner"}, "nested")

	require.Equal(t, "attempt to overwrite non-link at outer", err.Error())
}

// Verify that when setting a non-link, clobbering of a link is not allowed.
func TestDagSetNoClobberLink(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.SetAsLink([]string{"link"}, map[string]interface{}{
		"link": true,
	})
	require.Nil(t, err)

	_, err = dag.Set([]string{"link"}, "simple")

	require.Equal(t, "attempt to overwrite link at link with non-link", err.Error())
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

	require.Equal(t, "attempt to overwrite non-link at outer", err.Error())
}

// Verify that when setting a link, clobbering of a non-link is not allowed.
func TestDagSetAsLinkNoClobberNonLink(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.Set([]string{"non-link"}, "simple")
	require.Nil(t, err)

	_, err = dag.SetAsLink([]string{"non-link"}, map[string]interface{}{
		"link": true,
	})

	require.Equal(t, "attempt to overwrite non-link at non-link with a link", err.Error())
}

// Verify that SetAsLink allows overwriting
func TestDagSetAsLinkOverwrite(t *testing.T) {
	path := []string{"path"}
	newVal := []string{"test"}

	sw := &safewrap.SafeWrap{}
	root := sw.WrapObject(map[string]interface{}{})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := NewDagWithNodes(store, root)
	require.Nil(t, err)

	dag, err = dag.SetAsLink(path, "test")
	require.Nil(t, err)

	dag, err = dag.SetAsLink(path, newVal)
	require.Nil(t, err)

	got, remainingPath, err := dag.Resolve(path)
	gotSlice, ok := got.([]interface{})
	require.True(t, ok)
	require.Nil(t, err)
	require.Equal(t, 0, len(remainingPath))
	require.Equal(t, 1, len(gotSlice))
	require.Equal(t, "test", gotSlice[0])
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
