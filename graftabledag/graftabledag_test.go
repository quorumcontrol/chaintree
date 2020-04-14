package graftabledag

import (
	"context"
	"fmt"
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

	return nil, fmt.Errorf("no tip found for %s", did)
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

func newDagGetter(t *testing.T, ctx context.Context, dagsYo... *dag.Dag) *TestDagGetter {
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

func newGraftedDag(t *testing.T, ctx context.Context) (gd *GraftedDag, graftedPath []string) {
	sw := safewrap.SafeWrap{}
	deepChild := sw.WrapObject(map[string]interface{}{"deepChild": true})
	child1 := sw.WrapObject(map[string]interface{}{"deepChild1": deepChild.Cid(), "child1": true})
	child2 := sw.WrapObject(map[string]interface{}{"deepChild2": deepChild.Cid(), "child2": true})
	chain := sw.WrapObject(map[string]interface{}{})
	data := sw.WrapObject(map[string]interface{}{
		"child1": child1.Cid(),
		"child2": child2.Cid(),
	})
	tree := sw.WrapObject(map[string]interface{}{
		"data": data.Cid(),
	})
	root := sw.WrapObject(map[string]interface{}{
		"id":    "did:tupelo:imachaintree",
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
	})
	require.Nil(t, sw.Err)

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)
	dag1, err := dag.NewDagWithNodes(ctx, store, root, chain, data, tree, deepChild, child1, child2)
	require.Nil(t, err)
	ct1, err := dagToChaintree(ctx, dag1)
	require.Nil(t, err)
	did1, err := ct1.Id(ctx)
	require.Nil(t, err)

	randomValue := sw.WrapObject(map[string]interface{}{"random": "thingy"})
	graftPoint := sw.WrapObject(map[string]interface{}{"graft": did1+"/tree/data/child1/deepChild1"})
	data2 := sw.WrapObject(map[string]interface{}{
		"child1": randomValue.Cid(),
		"child2": graftPoint.Cid(),
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

	dg := newDagGetter(t, ctx, dag1, dag2)

	gd, err = New(dag2, dg)
	require.Nil(t, err)

	graftedPath = []string{"tree", "data", "child2", "graft", "deepChild"}

	return gd, graftedPath
}

func TestGraftedDag_GlobalResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gd, path := newGraftedDag(t, ctx)

	val, remaining, err := gd.GlobalResolve(ctx, path)
	require.Nil(t, err)
	require.Empty(t, remaining)

	v, ok := val.(bool)
	assert.True(t, ok)
	assert.True(t, v)
}
