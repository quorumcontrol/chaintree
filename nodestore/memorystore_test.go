package nodestore

import (
	"testing"

	"github.com/ipfs/go-ipld-cbor"

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

func TestMemoryNodeStoreDeleteIfUnreferenced(t *testing.T) {
	type testStruct struct {
		description  string
		setup        func(t *testing.T) (*cid.Cid, NodeStore)
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
			setup: func(t *testing.T) (*cid.Cid, NodeStore) {
				mns := &MemoryNodeStore{}
				mns.Initialize()
				node, err := mns.CreateNode(defaultMap)
				require.Nil(t, err)
				return node.Cid(), mns
			},
		},
		{
			description:  "a referenced node",
			shouldErr:    false,
			shouldDelete: false,
			setup: func(t *testing.T) (*cid.Cid, NodeStore) {
				mns := &MemoryNodeStore{}
				mns.Initialize()
				node, err := mns.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]*cid.Cid{
					"ref": node.Cid(),
				}
				_, err = mns.CreateNode(root)
				require.Nil(t, err)

				return node.Cid(), mns
			},
		},
		{
			description:  "a node with link",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) (*cid.Cid, NodeStore) {
				mns := &MemoryNodeStore{}
				mns.Initialize()
				node, err := mns.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]*cid.Cid{
					"ref": node.Cid(),
				}
				rootNode, err := mns.CreateNode(root)
				require.Nil(t, err)

				return rootNode.Cid(), mns
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

func TestMemoryNodeStoreDeleteTree(t *testing.T) {
	type testCase struct {
		description string
		setup       func() (nodesToCreate []*cbornode.Node, tipToDelete *cid.Cid)
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
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete *cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				return []*cbornode.Node{node}, node.Cid()
			},
			tests: func(mns NodeStore, nodes []*cbornode.Node) {
				saved, err := mns.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
		{
			description: "a tree",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete *cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				root := map[string]*cid.Cid{"child": node.Cid()}
				rootNode := sw.WrapObject(root)

				return []*cbornode.Node{node, rootNode}, rootNode.Cid()
			},
			tests: func(mns NodeStore, nodes []*cbornode.Node) {
				saved, err := mns.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
				saved, err = mns.GetNode(nodes[1].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
		{
			description: "a tree with another reference",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete *cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				root := map[string]*cid.Cid{"child": node.Cid()}
				rootNode := sw.WrapObject(root)
				otherRefHolder := map[string]*cid.Cid{"diferentNode": node.Cid()}
				otherRefHolderNode := sw.WrapObject(otherRefHolder)
				require.Nil(t, sw.Err)
				return []*cbornode.Node{node, rootNode, otherRefHolderNode}, rootNode.Cid()
			},
			tests: func(mns NodeStore, nodes []*cbornode.Node) {
				saved, err := mns.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.NotNil(t, saved)

				saved, err = mns.GetNode(nodes[1].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)

				saved, err = mns.GetNode(nodes[2].Cid())
				assert.Nil(t, err)
				assert.NotNil(t, saved)
			},
		},
	} {
		mns := &MemoryNodeStore{}
		mns.Initialize()
		nodes, tipToDelete := tc.setup()
		for _, node := range nodes {
			_, err := mns.CreateNodeFromBytes(node.RawData())
			require.Nil(t, err)
		}
		err := mns.DeleteTree(tipToDelete)
		if tc.shouldErr {
			assert.NotNil(t, err, tc.description)
		} else {
			assert.Nil(t, err, tc.description)
		}
		tc.tests(mns, nodes)
	}
}
