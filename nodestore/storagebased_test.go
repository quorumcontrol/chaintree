package nodestore

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipld-cbor"
	ds "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-datastore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageBasedStoreCreateNode(t *testing.T) {
	sbs := NewStorageBasedStore(ds.NewMapDatastore())
	obj := map[string]string{"hi": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	node, err := sbs.CreateNode(obj)
	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func TestStorageBasedStoreGetNode(t *testing.T) {
	sbs := NewStorageBasedStore(ds.NewMapDatastore())
	obj := map[string]string{"hi": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	sbs.CreateNode(obj)

	node, err := sbs.GetNode(testNode.Cid())

	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func TestStorageBasedStoreGetReferences(t *testing.T) {
	sbs := NewStorageBasedStore(ds.NewMapDatastore())
	sw := &safewrap.SafeWrap{}

	child := map[string]string{"hi": "value"}
	childNode := sw.WrapObject(child)
	root := map[string]cid.Cid{"child": childNode.Cid()}
	rootNode := sw.WrapObject(root)

	_, err := sbs.CreateNode(child)
	require.Nil(t, err)
	_, err = sbs.CreateNode(root)
	require.Nil(t, err)

	refs, err := sbs.GetReferences(childNode.Cid())
	require.Nil(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t, refs[rootNode.Cid().KeyString()].String(), rootNode.Cid().String())
}

func TestStorageBasedStoreDeleteIfUnreferenced(t *testing.T) {
	type testStruct struct {
		description  string
		setup        func(t *testing.T) (cid.Cid, NodeStore)
		shouldErr    bool
		shouldDelete bool
	}
	defaultMap := map[string]string{
		"hi": "hi",
	}
	for _, test := range []testStruct{
		{
			description:  "an unreferenced node",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) (cid.Cid, NodeStore) {
				sbs := NewStorageBasedStore(ds.NewMapDatastore())
				node, err := sbs.CreateNode(defaultMap)
				require.Nil(t, err)
				return node.Cid(), sbs
			},
		},
		{
			description:  "a referenced node",
			shouldErr:    false,
			shouldDelete: false,
			setup: func(t *testing.T) (cid.Cid, NodeStore) {
				sbs := NewStorageBasedStore(ds.NewMapDatastore())
				node, err := sbs.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]cid.Cid{
					"ref": node.Cid(),
				}
				_, err = sbs.CreateNode(root)
				require.Nil(t, err)

				return node.Cid(), sbs
			},
		},
		{
			description:  "a node with link",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) (cid.Cid, NodeStore) {
				sbs := NewStorageBasedStore(ds.NewMapDatastore())
				node, err := sbs.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]cid.Cid{
					"ref": node.Cid(),
				}
				rootNode, err := sbs.CreateNode(root)
				require.Nil(t, err)

				return rootNode.Cid(), sbs
			},
		},
	} {
		toDelete, store := test.setup(t)
		err := store.DeleteIfUnreferenced(toDelete)
		existing, err := store.GetNode(toDelete)
		require.Nil(t, err, test.description)

		if test.shouldDelete {
			require.Nil(t, existing, test.description)
		} else {
			require.NotNil(t, existing, test.description)
		}
		if test.shouldErr {
			require.NotNil(t, err, test.description)
		} else {
			require.Nil(t, err, test.description)
		}

	}
}

func TestStorageBasedStoreDeleteTree(t *testing.T) {
	type testCase struct {
		description string
		setup       func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid)
		tests       func(store NodeStore, nodes []*cbornode.Node)
		shouldErr   bool
	}

	defaultMap := map[string]string{
		"hi": "hi",
	}

	for _, tc := range []testCase{
		{
			description: "a single node",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				return []*cbornode.Node{node}, node.Cid()
			},
			tests: func(sbs NodeStore, nodes []*cbornode.Node) {
				saved, err := sbs.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
		{
			description: "a tree",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				root := map[string]cid.Cid{"child": node.Cid()}
				rootNode := sw.WrapObject(root)

				return []*cbornode.Node{node, rootNode}, rootNode.Cid()
			},
			tests: func(sbs NodeStore, nodes []*cbornode.Node) {
				saved, err := sbs.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
				saved, err = sbs.GetNode(nodes[1].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
		{
			description: "a tree with another reference",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				root := map[string]cid.Cid{"child": node.Cid()}
				rootNode := sw.WrapObject(root)
				otherRefHolder := map[string]cid.Cid{"diferentNode": node.Cid()}
				otherRefHolderNode := sw.WrapObject(otherRefHolder)
				require.Nil(t, sw.Err)
				return []*cbornode.Node{node, rootNode, otherRefHolderNode}, rootNode.Cid()
			},
			tests: func(sbs NodeStore, nodes []*cbornode.Node) {
				saved, err := sbs.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.NotNil(t, saved)

				saved, err = sbs.GetNode(nodes[1].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)

				saved, err = sbs.GetNode(nodes[2].Cid())
				assert.Nil(t, err)
				assert.NotNil(t, saved)
			},
		},
	} {
		sbs := NewStorageBasedStore(ds.NewMapDatastore())
		nodes, tipToDelete := tc.setup()
		for _, node := range nodes {
			_, err := sbs.CreateNodeFromBytes(node.RawData())
			require.Nil(t, err)
		}
		err := sbs.DeleteTree(tipToDelete)
		if tc.shouldErr {
			assert.NotNil(t, err, tc.description)
		} else {
			assert.Nil(t, err, tc.description)
		}
		tc.tests(sbs, nodes)
	}
}

func TestStorageBasedStoreResolve(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
		"key":   "value",
	})

	assert.Nil(t, sw.Err)
	sbs := NewStorageBasedStore(ds.NewMapDatastore())
	sbs.CreateNodeFromBytes(child.RawData())
	sbs.CreateNodeFromBytes(root.RawData())

	// Resolves through the tree
	val, remaining, err := sbs.Resolve(root.Cid(), []string{"child", "name"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "child", val)

	// Resolves on the object itself
	val, remaining, err = sbs.Resolve(root.Cid(), []string{"key"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "value", val)

	// Does not error on missing paths, but returns a nil value, with the part of the path missing
	val, remaining, err = sbs.Resolve(root.Cid(), []string{"child", "missing", "path"})
	assert.Nil(t, err)
	assert.Nil(t, val)
	assert.Equal(t, []string{"missing", "path"}, remaining)
}
