package dag

import (
	"context"
	"math"
	"testing"

	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

type dagTestStruct struct {
	Height uint64 `refmt:"height" json:"height" cbor:"height"`
	Child  *cid.Cid
}

func init() {
	cbornode.RegisterCborType(dagTestStruct{})
}

func newDeepDag(t *testing.T, ctx context.Context) *Dag {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child": true})
	root := sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag, err := NewDagWithNodes(ctx, store, root, deepChild, child)
	require.Nil(t, err)
	return dag
}

func newDeepAndWideDag(t *testing.T, ctx context.Context) *Dag {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child1 := sw.WrapObject(map[string]interface{}{"deepChild1": deepChild.Cid(), "child1": true})
	child2 := sw.WrapObject(map[string]interface{}{"deepChild2": deepChild.Cid(), "child2": true})
	root := sw.WrapObject(map[string]interface{}{"child1": child1.Cid(), "child2": child2.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag, err := NewDagWithNodes(ctx, store, root, deepChild, child1, child2)
	require.Nil(t, err)
	return dag
}

func TestDagNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dag := newDeepDag(t, ctx)
	nodes, err := dag.Nodes(ctx)
	assert.Nil(t, err)
	assert.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t, ctx)
	nodes, err = dag.Nodes(ctx)
	assert.Nil(t, err)
	// Removes uniques
	assert.Len(t, nodes, 4)
}

// This test is asserting a weirdness just to show you what's going on with uint64
// because we resolve into an empty interface{}, refmt now defaults to a int intead of an int64
func TestUint64(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := safewrap.SafeWrap{}
	// making a struct with the maximum uint64 size
	child := sw.WrapObject(&dagTestStruct{Height: math.MaxUint64})
	root := sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)

	// resolving to an empty interface overflows the int and makes it a -1
	val, remain, err := dag.Resolve(ctx, []string{"child", "height"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, -1, val)

	// however, you can resolve to a known type and it will resolve correctly
	typed := &dagTestStruct{}
	err = dag.ResolveInto(ctx, []string{"child"}, typed)
	require.Nil(t, err)
	assert.Equal(t, uint64(math.MaxUint64), typed.Height)
}

func TestResolveInto(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := safewrap.SafeWrap{}
	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	// it works when the node is the root node
	root := sw.WrapObject(&dagTestStruct{Height: 1234})
	require.Nil(t, sw.Err)

	dag, err := NewDagWithNodes(ctx, store, root)
	require.Nil(t, err)
	newRoot := &dagTestStruct{}
	err = dag.ResolveInto(ctx, nil, newRoot)
	require.Nil(t, err)
	assert.Equal(t, uint64(1234), newRoot.Height)

	// it works one level down (and with a uint64)
	child := sw.WrapObject(&dagTestStruct{Height: math.MaxUint64})
	root = sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	dag, err = NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)
	newChild := &dagTestStruct{}
	err = dag.ResolveInto(ctx, []string{"child"}, newChild)
	require.Nil(t, err)
	assert.Equal(t, uint64(math.MaxUint64), newChild.Height)

	// it gives a not found when trying to use a link with a remain
	err = dag.ResolveInto(ctx, []string{"child", "bob"}, newChild)
	assert.Equal(t, format.ErrNotFound, err)

	// it gives an error when trying to use a non-object
	err = dag.ResolveInto(ctx, []string{"child", "child"}, newChild)
	assert.Equal(t, "error the path did not resolve to a link", err.Error())
}

func TestDagResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dag := newDeepDag(t, ctx)
	val, remain, err := dag.Resolve(ctx, []string{"child", "deepChild", "deepChild"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, true, val)
}

// Test that the ResolveAt method can operate with a tip that need not be current.
func TestDagResolveAt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dag := newDeepDag(t, ctx)
	oldTip := dag.Tip
	dag, err := dag.Set(ctx, []string{"child", "value"}, true)
	require.Nil(t, err)

	val, remain, err := dag.ResolveAt(ctx, oldTip, []string{"child", "deepChild", "deepChild"})
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, true, val)

	missingVal, remain, err := dag.ResolveAt(ctx, oldTip, []string{"child", "value"})
	require.Nil(t, err)
	require.Len(t, remain, 1)
	require.Equal(t, remain, []string{"value"})
	require.Nil(t, missingVal)
}

func TestOrderedNodesForPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child": true})
	root := sw.WrapObject(map[string]interface{}{"child": child.Cid(), "root": true})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, deepChild, child)
	require.Nil(t, err)

	nodes, err := dag.orderedNodesForPath(ctx, []string{"child", "deepChild"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	require.Equal(t, nodes[0].RawData(), root.RawData())
	require.Equal(t, nodes[1].RawData(), child.RawData())
	require.Equal(t, nodes[2].RawData(), deepChild.RawData())
}

func TestDagNodesForPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dag := newDeepDag(t, ctx)
	nodes, err := dag.NodesForPath(ctx, []string{"child", "deepChild"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)
	allNodes, err := dag.Nodes(ctx)
	require.Nil(t, err)
	require.Len(t, allNodes, 3)

	nodeBytes := make([][]byte, len(allNodes))
	for i, n := range allNodes {
		nodeBytes[i] = n.RawData()
	}

	for _, n := range nodes {
		require.Contains(t, nodeBytes, n.RawData())
	}

	dag = newDeepAndWideDag(t, ctx)
	nodes, err = dag.NodesForPath(ctx, []string{"child2", "deepChild2"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t, ctx)
	nodes, err = dag.NodesForPath(ctx, []string{"child2"})
	require.Nil(t, err)
	require.Len(t, nodes, 2)
}

func TestDagNodesForPathWithDecendants(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dag := newDeepAndWideDag(t, ctx)
	nodes, err := dag.NodesForPathWithDecendants(ctx, []string{"child2", "deepChild2"})
	require.Nil(t, err)
	require.Len(t, nodes, 3)

	dag = newDeepAndWideDag(t, ctx)
	nodes2, err := dag.NodesForPathWithDecendants(ctx, []string{"child2"})
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)

	dag, err = dag.Set(ctx, []string{"test"}, "bob")
	assert.Nil(t, err)

	val, _, err := dag.Resolve(ctx, []string{"test"})

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// test top level sibling
	dag, err = dag.Set(ctx, []string{"test2"}, "alice")
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"test"})
	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	val2, _, err := dag.Resolve(ctx, []string{"test2"})
	assert.Nil(t, err)
	assert.Equal(t, "alice", val2)

	// test works with a CID
	err = dag.AddNodes(ctx, unlinked)
	require.Nil(t, err)

	dag, err = dag.Set(ctx, []string{"test"}, unlinked.Cid())
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"test", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)

	// test works in non-existant path

	path := []string{"child", "non-existant-nested", "objects", "test"}
	dag, err = dag.Set(ctx, path, "bob")
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, path)

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// Test sibling of existing path
	siblingPath := []string{"child", "non-existant-nested", "objects", "siblingtest"}
	dag, err = dag.Set(ctx, siblingPath, "sue")
	assert.Nil(t, err)

	// original sibling is still available
	val, _, err = dag.Resolve(ctx, path)
	require.Nil(t, err)
	assert.Equal(t, "bob", val)

	siblingVal, _, err := dag.Resolve(ctx, siblingPath)

	assert.Nil(t, err)
	assert.Equal(t, "sue", siblingVal)

	// Test sibling of partially existing path
	partiallyExistingPath := []string{"child", "non-existant-nested", "other-objects", "nestedtest"}
	dag, err = dag.Set(ctx, partiallyExistingPath, "carol")
	assert.Nil(t, err)

	// original sibling is still available
	val, _, err = dag.Resolve(ctx, path)
	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// second sibling is still available
	siblingVal, _, err = dag.Resolve(ctx, siblingPath)
	assert.Nil(t, err)
	assert.Equal(t, "sue", siblingVal)

	// check partially existing path set
	partiallyExistingVal, _, err := dag.Resolve(ctx, partiallyExistingPath)
	assert.Nil(t, err)
	assert.Equal(t, "carol", partiallyExistingVal)
}

func TestDagSetAsLink(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)

	dag, err = dag.SetAsLink(ctx, []string{"child", "grandchild", "key"}, unlinked)
	assert.Nil(t, err)
	val, _, err := dag.Resolve(ctx, []string{"child", "grandchild", "key", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)

	unlinked2 := map[string]interface{}{
		"unlinked2": false,
	}

	dag, err = dag.SetAsLink(ctx, []string{"child", "grandchild", "key", "unlinkedsibling"}, unlinked2)
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"child", "grandchild", "key", "unlinkedsibling", "unlinked2"})
	assert.Nil(t, err)
	assert.Equal(t, false, val)
}

func TestDagSetNestedAfterSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	tip := sw.WrapObject(map[string]interface{}{})
	dag, err := NewDagWithNodes(ctx, store, tip)
	require.Nil(t, err)

	// random other key to ensure other data remains intact
	dag, err = dag.Set(ctx, []string{"other"}, "hello")
	assert.Nil(t, err)

	// with string value
	dag, err = dag.Set(ctx, []string{"test"}, "test-str")
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err := dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	dag, err = dag.Set(ctx, []string{"test", "test-key"}, "test-str-2")
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	val, _, err = dag.Resolve(ctx, []string{"test", "test-key"})
	assert.Nil(t, err)
	assert.Equal(t, "test-str-2", val)

	// with int value
	dag, err = dag.Set(ctx, []string{"test"}, 42)
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	dag, err = dag.Set(ctx, []string{"test", "test-key"}, 43)
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	val, _, err = dag.Resolve(ctx, []string{"test", "test-key"})
	assert.Nil(t, err)
	assert.Equal(t, 43, val)

	// with multiple levels of non-existent path
	dag, err = dag.Set(ctx, []string{"test"}, "test-str")
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	dag, err = dag.Set(ctx, []string{"test", "down", "in", "the", "thing"}, "test-str-2")
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	val, _, err = dag.Resolve(ctx, []string{"test", "down", "in", "the", "thing"})
	assert.Nil(t, err)
	assert.Equal(t, "test-str-2", val)
}

func TestDagSetAsLinkAfterSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sw := &safewrap.SafeWrap{}

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	tip := sw.WrapObject(map[string]interface{}{})
	dag, err := NewDagWithNodes(ctx, store, tip)
	require.Nil(t, err)

	// random other key to ensure other data remains intact
	dag, err = dag.Set(ctx, []string{"other"}, "hello")
	assert.Nil(t, err)

	// with string value
	dag, err = dag.Set(ctx, []string{"test"}, "test-str")
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err := dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	dag, err = dag.SetAsLink(ctx, []string{"test"}, map[string]string{"test-key": "test-str-2"})
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"test", "test-key"})
	assert.Nil(t, err)
	assert.Equal(t, "test-str-2", val)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	// with int value
	dag, err = dag.Set(ctx, []string{"test"}, 42)
	assert.Nil(t, err)

	dag, err = dag.SetAsLink(ctx, []string{"test"}, map[string]int{"test-key": 43})
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"test", "test-key"})
	assert.Nil(t, err)
	assert.Equal(t, 43, val)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	// with multiple levels of non-existent path
	dag, err = dag.SetAsLink(ctx, []string{"test"}, map[string]string{
		"foo": "bar",
	})
	assert.Nil(t, err)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)

	dag, err = dag.Set(ctx, []string{"test", "down", "in", "the", "thing"}, "test-str-2")
	assert.Nil(t, err)

	val, _, err = dag.Resolve(ctx, []string{"test", "down", "in", "the", "thing"})
	assert.Nil(t, err)
	assert.Equal(t, "test-str-2", val)

	// make sure other key & value are still there
	val, remaining, err = dag.Resolve(ctx, []string{"other"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "hello", val)
}

func TestDagInvalidSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)

	_, err = dag.Set(ctx, []string{"test"}, map[string]interface{}{
		"child1": "1",
		"child2": "2",
	})
	require.NotNil(t, err)
}

func TestDagGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	notStored := sw.WrapObject(map[string]string{"test": "notinthedb"})

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)
	n, err := dag.Get(ctx, child.Cid())
	require.Nil(t, err)
	assert.Equal(t, child.Cid().String(), n.Cid().String())

	// Getting an not-found node doesn't error
	n, err = dag.Get(ctx, notStored.Cid())
	require.Nil(t, err)
	assert.Nil(t, n)
}

func TestDagDump(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Not really a test here, but do call it just to make sure no panics
	dag := newDeepDag(t, ctx)
	t.Log(dag.Dump(ctx))
}

func TestDagWithNewTip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, child)
	require.Nil(t, err)

	newDag := dag.WithNewTip(child.Cid())
	assert.Equal(t, newDag.Tip.String(), child.Cid().String())
	nodes, err := newDag.Nodes(ctx)
	require.Nil(t, err)
	assert.Len(t, nodes, 1)
}

func TestDagUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	dag, err := NewDagWithNodes(ctx, store, root, intermediary, child)
	require.Nil(t, err)

	dag, err = dag.Update(ctx, []string{"child1", "child2"}, map[string]interface{}{"name": "changed"})
	require.Nil(t, err)

	val, remain, err := dag.Resolve(ctx, []string{"child1", "child2", "name"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, "changed", val)
}

func TestDagDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag, err := NewDagWithNodes(ctx, store, root, intermediary, child)
	require.Nil(t, err)

	dag, err = dag.Delete(ctx, []string{"child1", "child2"})
	require.Nil(t, err)

	val, remain, err := dag.Resolve(ctx, []string{"child1"})
	require.Nil(t, err)
	assert.Len(t, remain, 0)

	valCast := make(map[string]string, len(val.(map[string]interface{})))
	for k, v := range val.(map[string]interface{}) {
		valCast[k] = v.(string)
	}
	assert.Equal(t, valCast, map[string]string{"name": "intermediary"})
}
