package graftabledag

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

type TestDagGetter struct {
	dagsYo map[string]*dag.Dag
}

var _ DagGetter = (*TestDagGetter)(nil)

func (tdg *TestDagGetter) GetTip(_ context.Context, did string) (*cid.Cid, error) {
	if d, ok := tdg.dagsYo[did]; ok {
		return &d.Tip, nil
	}

	return nil, chaintree.ErrTipNotFound
}

func (tdg *TestDagGetter) GetLatest(ctx context.Context, did string) (*chaintree.ChainTree, error) {
	if d, ok := tdg.dagsYo[did]; ok {
		return dagToChaintree(ctx, d)
	}

	return nil, fmt.Errorf("no chaintree found for %s", did)
}

func dagToChaintree(ctx context.Context, d *dag.Dag) (*chaintree.ChainTree, error) {
	blockValidators := make([]chaintree.BlockValidatorFunc, 0)
	transactorFuncs := make(map[transactions.Transaction_Type]chaintree.TransactorFunc)
	return chaintree.NewChainTree(ctx, d, blockValidators, transactorFuncs)
}

func newDagGetter(t *testing.T, ctx context.Context, dagsYo ...*dag.Dag) *TestDagGetter {
	dagGetter := &TestDagGetter{
		dagsYo: make(map[string]*dag.Dag),
	}

	for _, d := range dagsYo {
		uncastDid, _, err := d.Resolve(ctx, []string{"id"})
		require.Nil(t, err)
		did, ok := uncastDid.(string)
		require.True(t, ok)
		dagGetter.dagsYo[did] = d
	}

	return dagGetter
}

func newGraftedDag(t *testing.T, ctx context.Context) (gd *GraftedDag, graftedPath chaintree.Path) {
	sw := safewrap.SafeWrap{}

	// DAG 1
	deepChild := sw.WrapObject(map[string]interface{}{"deepChildBoolVal": true, "deepChildStrVal": "deepStr", "deepChildIntVal": 1805092908})
	child1 := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child1": true, "otherVal": "stringTest"})
	child2 := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child2": true, "otherVal": 42})
	chain := sw.WrapObject(map[string]interface{}{})
	data := sw.WrapObject(map[string]interface{}{
		"child1": child1.Cid(),
		"child2": child2.Cid(),
		"children": []interface{}{
			child1.Cid(),
			child2.Cid(),
		},
	})
	tree := sw.WrapObject(map[string]interface{}{
		"data": data.Cid(),
	})
	did1 := "did:tupelo:imachaintree"
	root := sw.WrapObject(map[string]interface{}{
		"id":    did1,
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag1, err := dag.NewDagWithNodes(ctx, store, root, chain, tree, data, child1, child2, deepChild)
	require.Nil(t, err)

	// DAG 2
	randomValue := sw.WrapObject(map[string]interface{}{"random": "thingy"})
	graftPoint := sw.WrapObject(map[string]interface{}{"graft": did1 + "/tree/data/child1/deepChild"})
	data2 := sw.WrapObject(map[string]interface{}{
		"child1":          randomValue.Cid(),
		"child2":          graftPoint.Cid(),
		"graftedChildren": did1 + "/tree/data/children",
		"graftedSlice": []string{
			did1 + "/tree/data/child2/otherVal",
			did1 + "/tree/data/child1/otherVal",
			did1 + "/tree/data/child2/otherVal",
		},
		"mixedSlice": []interface{}{
			"mixedString",
			did1 + "/tree/data/child1/otherVal",
			true,
			did1 + "/tree/data/child2/otherVal",
		},
		"stringSlice": []string{
			"test1",
			"test2",
			"test1",
		},
		"stringSliceLen1": []string{
			"testSingle",
		},
		"boolSlice": []bool{
			true, true, false,
		},
	})
	tree2 := sw.WrapObject(map[string]interface{}{
		"data": data2.Cid(),
	})
	root2 := sw.WrapObject(map[string]interface{}{
		"id":    "did:tupelo:imachaintreetoo",
		"chain": chain.Cid(),
		"tree":  tree2.Cid(),
	})
	require.Nil(t, sw.Err)

	dag2, err := dag.NewDagWithNodes(ctx, store, root2, chain, tree2, data2, graftPoint, randomValue)
	require.Nil(t, err)

	dg := newDagGetter(t, ctx, dag1, dag2)

	gd, err = New(dag2, dg)
	require.Nil(t, err)

	graftedPath = chaintree.Path{"tree", "data", "child2", "graft", "deepChildBoolVal"}

	return gd, graftedPath
}

func newGraftedDagWithLoop(t *testing.T, ctx context.Context) (gd *GraftedDag, graftedPath chaintree.Path) {
	sw := safewrap.SafeWrap{}

	// DAG 1
	deepChild := sw.WrapObject(map[string]interface{}{"deepChildBoolVal": true, "deepChildStrVal": "deepStr", "deepChildIntVal": 1805092908})
	child1 := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child1": true})
	child2 := sw.WrapObject(map[string]interface{}{"deepChild": deepChild.Cid(), "child2": true})
	chain := sw.WrapObject(map[string]interface{}{})
	data := sw.WrapObject(map[string]interface{}{
		"child1": child1.Cid(),
		"child2": child2.Cid(),
		"loop":   "did:tupelo:imachaintreethree/tree/data/loop",
	})
	tree := sw.WrapObject(map[string]interface{}{
		"data": data.Cid(),
	})
	did1 := "did:tupelo:imachaintree"
	root := sw.WrapObject(map[string]interface{}{
		"id":    did1,
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag1, err := dag.NewDagWithNodes(ctx, store, root, chain, tree, data, child1, child2, deepChild)
	require.Nil(t, err)

	// DAG 2
	randomValue := sw.WrapObject(map[string]interface{}{"random": "thingy"})
	graftPoint := sw.WrapObject(map[string]interface{}{"graft": did1 + "/tree/data/child1/deepChild"})
	data2 := sw.WrapObject(map[string]interface{}{
		"child1": randomValue.Cid(),
		"child2": graftPoint.Cid(),
		"loop":   "did:tupelo:imachaintree/tree/data/loop",
	})
	tree2 := sw.WrapObject(map[string]interface{}{
		"data": data2.Cid(),
	})
	root2 := sw.WrapObject(map[string]interface{}{
		"id":    "did:tupelo:imachaintreetoo",
		"chain": chain.Cid(),
		"tree":  tree2.Cid(),
	})
	require.Nil(t, sw.Err)

	dag2, err := dag.NewDagWithNodes(ctx, store, root2, tree2, data2, graftPoint, randomValue)
	require.Nil(t, err)

	data3 := sw.WrapObject(map[string]interface{}{
		"loop": "did:tupelo:imachaintreetoo/tree/data/loop/down/in/the/thing",
	})
	tree3 := sw.WrapObject(map[string]interface{}{
		"data": data3.Cid(),
	})
	root3 := sw.WrapObject(map[string]interface{}{
		"id":    "did:tupelo:imachaintreethree",
		"chain": chain.Cid(),
		"tree":  tree3.Cid(),
	})
	require.Nil(t, sw.Err)

	dag3, err := dag.NewDagWithNodes(ctx, store, root3, tree3, data3)
	require.Nil(t, err)

	dg := newDagGetter(t, ctx, dag1, dag2, dag3)

	gd, err = New(dag3, dg)
	require.Nil(t, err)

	graftedPath = chaintree.Path{"tree", "data", "loop"}

	return gd, graftedPath
}

func TestGraftedDag_GlobalResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gd, path := newGraftedDag(t, ctx)

	val, remaining, err := gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, val, true)
}

func TestGraftedDag_GlobalResolveTypes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gd, _ := newGraftedDag(t, ctx)

	var (
		val       interface{}
		remaining chaintree.Path
		err       error
		path      chaintree.Path
	)

	// direct value string slice
	path = chaintree.Path{"tree", "data", "stringSlice"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, []interface{}{"test1", "test2", "test1"}, val)

	// direct value string slice, len 1
	path = chaintree.Path{"tree", "data", "stringSliceLen1"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, []interface{}{"testSingle"}, val)

	// direct value bool slice
	path = chaintree.Path{"tree", "data", "boolSlice"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, []interface{}{true, true, false}, val)

	// grafted single bool val
	path = chaintree.Path{"tree", "data", "child2", "graft", "deepChildBoolVal"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, true, val)

	// grafted map of different types
	path = chaintree.Path{"tree", "data", "child2", "graft"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, map[string]interface{}{
		"deepChildBoolVal": true,
		"deepChildIntVal":  1805092908,
		"deepChildStrVal":  "deepStr",
	}, val)

	// grafted index access
	path = chaintree.Path{"tree", "data", "graftedChildren", "0", "otherVal"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, "stringTest", val)

	// grafted index access
	path = chaintree.Path{"tree", "data", "graftedChildren", "1", "otherVal"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, 42, val)

	// grafted array access, should maintain order
	path = chaintree.Path{"tree", "data", "graftedSlice"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, []interface{}{42, "stringTest", 42}, val)

	// mixed grafted / non grafted values
	path = chaintree.Path{"tree", "data", "mixedSlice"}
	val, remaining, err = gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)
	assert.Equal(t, []interface{}{"mixedString", "stringTest", true, 42}, val)
}

func TestGraftedDag_GlobalResolve_LoopDetection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gd, path := newGraftedDagWithLoop(t, ctx)

	_, _, err := gd.GlobalResolve(ctx, path)

	require.NotNil(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "loop detected"))
}

func TestGraftedDag_GlobalResolve_BeforeChaintreeExists(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := safewrap.SafeWrap{}

	chain := sw.WrapObject(map[string]interface{}{})
	data := sw.WrapObject(map[string]interface{}{
		"precomputed": "did:tupelo:doesnotexistyet/thingy",
	})
	tree := sw.WrapObject(map[string]interface{}{
		"data": data.Cid(),
	})
	did1 := "did:tupelo:imachaintree"
	root := sw.WrapObject(map[string]interface{}{
		"id":    did1,
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	d, err := dag.NewDagWithNodes(ctx, store, root, chain, tree, data)
	require.Nil(t, err)

	dg := newDagGetter(t, ctx, d)

	gd, err := New(d, dg)
	require.Nil(t, err)

	val, remaining, err := gd.GlobalResolve(ctx, chaintree.Path{"tree", "data", "precomputed"})
	assert.Nil(t, err)
	assert.Equal(t, "did:tupelo:doesnotexistyet/thingy", val)
	assert.Empty(t, remaining)
}
