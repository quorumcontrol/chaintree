package javascript

import (
	"io/ioutil"
	"testing"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	script, err := ioutil.ReadFile("./js/test.js")
	require.Nil(t, err)
	sw := &safewrap.SafeWrap{}

	tree := sw.WrapObject(map[string]string{
		"validIf": string(script),
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
		"id":    "test",
	})

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	dag, err := dag.NewDagWithNodes(store, root, tree, chain)
	require.Nil(t, err)
	chainTree, err := chaintree.NewChainTree(
		dag,
		nil,
		nil,
	)
	require.Nil(t, err)
	res, err := Validate(chainTree)
	require.Nil(t, err)

	assert.Equal(t, res, []byte("invalid"))

	// now set the node that the validator looks at
	dag, err = dag.Set([]string{"tree", "ok"}, true)
	require.Nil(t, err)
	chainTree.Dag = dag
	res, err = Validate(chainTree)
	require.Nil(t, err)

	assert.Equal(t, res, []byte("ok"))
}
