package dag

import (
	"github.com/ipfs/go-cid"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/ipfs/go-ipld-cbor"
)

type TestStruct struct {
	Name string
	Bar *cid.Cid
}

type TestLinks struct {
	Linkies []*cid.Cid
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

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()
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

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

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

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

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

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

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

	unlinked := sw.WrapObject(map[string]interface{}{
		"unlinked": true,
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	assert.Nil(t, sw.Err)

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

	err := tree.SetAsLink([]string{"child", "key"}, unlinked)
	assert.Nil(t, err)

	val,_,err := tree.Resolve([]string{"child", "key", "unlinked"})

	assert.Nil(t, err)
	assert.Equal(t, true, val)
}

func BenchmarkBidirectionalTree_Swap(b *testing.B) {
	sw := &SafeWrap{}
	child := sw.WrapObject(map[string]interface{} {
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
	})

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

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

	tree := NewBidirectionalTree()
	tree.AddNodes(root,child)
	tree.Tip = root.Cid()

	swapper := []*cbornode.Node{sw.WrapObject("key"), sw.WrapObject("key2")}
	assert.Nil(b, sw.Err)
	var err error

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		idx := n % 2
		err = tree.Set([]string{"child", "key"}, swapper[(idx + 1) %2])
	}

	assert.Nil(b, err)
}