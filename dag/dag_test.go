package dag

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/ipfs/go-ipld-cbor"
	"github.com/ipfs/go-cid"
)

func init() {
	cbornode.RegisterCborType(objWithNilPointers{})
}

func TestCreating(t *testing.T) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child)
	assert.NotNil(t, tree)
}

func TestBidirectionalTree_Prune(t *testing.T) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	orphan := sw.WrapObject(map[string]interface{} {
		"name": "orphan",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
		"key": "value",
	})


	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child, orphan)

	assert.Len(t, tree.nodesByStaticId, 3)

	tree.Prune()

	assert.Len(t, tree.nodesByStaticId, 2)

}

func TestBidirectionalTree_Resolve(t *testing.T) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
		"key": "value",
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child)

	val,remaining,err := tree.Resolve([]string{"child", "name"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "child", val)

	val,remaining,err = tree.Resolve([]string{"key"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "value", val)
}

func TestBidirectionalTree_Swap(t *testing.T) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	newRoot := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
		"isNew": true,
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child)

	newChild := sw.WrapObject(map[string]interface{} {
		"name": "newChild",
	})

	err := tree.Swap(child.Cid(), newChild)
	assert.Nil(t,err)

	val,remaining,err := tree.Resolve([]string{"child", "name"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "newChild", val)

	err = tree.Swap(newChild.Cid(), child)
	assert.Nil(t,err)

	val,remaining,err = tree.Resolve([]string{"child", "name"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "child", val)

	err = tree.Swap(tree.Tip, newRoot)
	assert.Nil(t,err)

	val,remaining,err = tree.Resolve([]string{"isNew"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, true, val)
}

func TestBidirectionalTree_Set(t *testing.T) {
	sw := &SafeWrap{}

	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	unlinked := sw.WrapObject(map[string]interface{}{
		"unlinked": true,
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child)

	err := tree.Set([]string{"test"}, "bob")
	assert.Nil(t, err)

	val,_,err := tree.Resolve([]string{"test"})

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)

	// test works with a CID
	tree.AddNodes(unlinked)

	err = tree.Set([]string{"test"}, unlinked.Cid())
	assert.Nil(t, err)

	val,_,err = tree.Resolve([]string{"test", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)

	// test works in non-existant path

	path := []string{"child", "non-existant-nested", "objects", "test"}
	err = tree.Set(path, "bob")
	assert.Nil(t, err)

	val,_,err = tree.Resolve(path)

	assert.Nil(t, err)
	assert.Equal(t, "bob", val)
}

func TestBidirectionalTree_SetAsLink(t *testing.T) {
	sw := &SafeWrap{}

	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	unlinked := map[string]interface{}{
		"unlinked": true,
	}

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree(root.Cid(), root, child)

	err := tree.SetAsLink([]string{"child", "key"}, unlinked)
	assert.Nil(t, err)

	val,_,err := tree.Resolve([]string{"child", "key", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)
}

func TestBidirectionalTree_Copy(t *testing.T) {
	sw := &SafeWrap{}

	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)
	tree := NewBidirectionalTree(root.Cid(), root, child)

	newTree := tree.Copy()

	assert.Equal(t, len(tree.nodesByCid), len(newTree.nodesByCid))
	assert.Equal(t, tree.Tip, newTree.Tip)

	for _,node := range tree.nodesByCid {
		assert.Equal(t, node.Node.Cid(), newTree.nodesByCid[node.Node.Cid().KeyString()].Node.Cid())
	}
}

func TestBidirectionalTree_Get(t *testing.T) {
	sw := &SafeWrap{}

	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)
	tree := NewBidirectionalTree(root.Cid(), root, child)

	assert.Equal(t, child, tree.Get(child.Cid()).Node)
}

func TestBidirectionalNode_AsMap(t *testing.T) {
	sw := &SafeWrap{}

	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)
	tree := NewBidirectionalTree(root.Cid(), root, child)

	rootAsMap,err := tree.Get(root.Cid()).AsMap()
	assert.Nil(t, err)

	assert.Equal(t, *child.Cid(), rootAsMap["child"].(cid.Cid))
}

type objWithNilPointers struct {
	NilPointer *cid.Cid
	Other string
	Cids []*cid.Cid
}

func TestSafeWrap_WrapObject(t *testing.T) {
	sw := &SafeWrap{}
	for _,test := range []struct{
		description string
		obj *objWithNilPointers
	} {
		{
			description: "an object with an empty cid",
			obj: &objWithNilPointers{Other: "something"},
		},
		{
			description: "an object with an array of CIDs",
			obj: &objWithNilPointers{
				Cids: []*cid.Cid{sw.WrapObject(map[string]string{"test":"test"}).Cid()},
			},
		},
	} {
		node := sw.WrapObject(test.obj)
		_,err := node.MarshalJSON()
		assert.Nil(t, err, test.description)

		//t.Log(string(j))
		assert.Nil(t, sw.Err, test.description)
	}
}

func BenchmarkBidirectionalTree_Swap(b *testing.B) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	tree := NewBidirectionalTree(root.Cid(), root, child)

	newChild := sw.WrapObject(map[string]interface{} {
		"name": "newChild",
	})

	swapper := []*cbornode.Node{child, newChild}

	var err error

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		idx := n % 2
		err = tree.Swap(swapper[idx].Cid(), swapper[(idx + 1) %2])
	}

	assert.Nil(b, err)
}

func BenchmarkBidirectionalTree_Set(b *testing.B) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	tree := NewBidirectionalTree(root.Cid(), root, child)


	swapper := []string{"key", "key2"}
	assert.Nil(b, sw.Err)
	var err error

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		idx := n % 2
		err = tree.Set([]string{"child", "key"}, swapper[(idx + 1) %2])
	}

	assert.Nil(b, err)
}

func BenchmarkBidirectionalTree_Copy(b *testing.B) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	tree := NewBidirectionalTree(root.Cid(), root, child)

	assert.Nil(b, sw.Err)
	// run the Fib function b.N times

	var newTree *BidirectionalTree

	for n := 0; n < b.N; n++ {
		newTree = tree.Copy()
	}

	assert.Equal(b, tree, newTree)
}

func BenchmarkBidirectionalNode_AsMap(b *testing.B) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	tree := NewBidirectionalTree(root.Cid(), root, child)

	assert.Nil(b, sw.Err)
	// run the Fib function b.N times

	var rootMap map[string]interface{}

	for n := 0; n < b.N; n++ {
		rootMap,_ = tree.Get(root.Cid()).AsMap()
	}

	assert.Equal(b, *child.Cid(), rootMap["child"])
}