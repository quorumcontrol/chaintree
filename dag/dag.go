package dag

/*
	The dag package holds convenience methods for working with a content-addressable DAG.
	The BidirectionalTree holds nodes
 */

import (
	"github.com/ipfs/go-ipld-cbor"
	"github.com/multiformats/go-multihash"
	"github.com/ipfs/go-cid"
	"sync"
	"fmt"
	"github.com/ipfs/go-ipld-format"
	"github.com/davecgh/go-spew/spew"
)

type nodeId int

type BidirectionalTree struct {
	Tip             *cid.Cid
	counter         int
	nodesByStaticId map[nodeId]*BidirectionalNode
	nodesByCid      map[string]*BidirectionalNode
	mutex           sync.Mutex
}

type ErrorCode struct {
	Code  int
	Memo string
}

func (e *ErrorCode) GetCode() int {
	return e.Code
}

func (e *ErrorCode) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Memo)
}

const (
	Success = 0
	ErrMissingRoot = 1
	ErrMissingPath = 2
	ErrInvalidInput = 3
	ErrEncodingError = 4
	ErrBadInput = 5
	ErrUnknown = 99
)


type BidirectionalNode struct {
	Parents map[nodeId]bool
	id      nodeId
	Node    *cbornode.Node
}

func NewBidirectionalTree(root *cid.Cid, nodes ...*cbornode.Node) *BidirectionalTree {
	tree := &BidirectionalTree{
		counter: 0,
		nodesByStaticId: make(map[nodeId]*BidirectionalNode),
		nodesByCid: make(map[string]*BidirectionalNode),
	}

	if len(nodes) > 0 {
		tree.AddNodes(nodes...)
	}
	if root != nil {
		tree.Tip = root
	}

	return tree
}

func (bn *BidirectionalNode) AsMap() (map[string]interface{}, error) {
	newParentJsonish := make(map[string]interface{})
	err := cbornode.DecodeInto(bn.Node.RawData(), &newParentJsonish)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}

	return newParentJsonish, nil
}

func (bn *BidirectionalNode) Resolve(tree *BidirectionalTree, path []string) (interface{}, []string, error) {
	val, remaining, err := bn.Node.Resolve(path)
	if err != nil {
		//fmt.Printf("error resolving: %v", err)
		return nil, nil, &ErrorCode{Code: ErrMissingPath, Memo: fmt.Sprintf("error resolving: %v", err)}
	}
	//spew.Dump("resolved on Node", val)

	switch val.(type) {
	case *format.Link:
		n,ok := tree.nodesByCid[val.(*format.Link).Cid.KeyString()]
		if ok {
			return n.Resolve(tree, remaining)
		} else {
			return nil, nil, &ErrorCode{Code: ErrMissingPath}
		}
	default:
		return val, remaining, err
	}
}

func (bt *BidirectionalTree) Get(id *cid.Cid) (*BidirectionalNode) {
	node,ok := bt.nodesByCid[id.KeyString()]
	if ok {
		return node
	}
	return nil
}

func (bt *BidirectionalTree) Copy() *BidirectionalTree {
	newNodes := make([]*cbornode.Node, len(bt.nodesByStaticId))

	i := 0
	for _,oldNode := range bt.nodesByStaticId {
		newNode, err := cbornode.Decode(oldNode.Node.RawData(), multihash.SHA2_256, -1)
		if err != nil {
			panic(fmt.Sprintf("this encoded, it should never fail to decode: %v", err))
		}
		newNodes[i] = newNode
		i++
	}

	newCid,err := cid.Cast(bt.Tip.Bytes())
	if err != nil {
		panic(fmt.Sprintf("this encoded, it should never fail to decode: %v", err))
	}

	//log.Printf("copying tree")

	return NewBidirectionalTree(newCid, newNodes...)
}

func (bt *BidirectionalTree) AddNodes(nodes ...*cbornode.Node) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()
	//log.Printf("Adding nodes: %d", len(nodes))

	for i,node := range nodes {
		_,ok := bt.nodesByCid[node.Cid().KeyString()]
		if !ok {
			bidiNode := &BidirectionalNode{
				Node:    node,
				id:      nodeId(bt.counter + i),
				Parents: make(map[nodeId]bool),
			}
			//log.Printf("adding bidiNode: %v, %s", bidiNode.id, node.Cid().String())
			bt.nodesByStaticId[bidiNode.id] = bidiNode
			bt.nodesByCid[node.Cid().KeyString()] = bidiNode
		}
	}
	bt.counter += len(nodes)
	bt.updateParents()
}

func (bt *BidirectionalTree) updateParents() {
	for _, bidiNode := range bt.nodesByStaticId {
		links := bidiNode.Node.Links()
		for _, link := range links {
			existing, ok := bt.nodesByCid[link.Cid.KeyString()]
			if ok {
				existing.Parents[bidiNode.id] = true
			}
		}
	}
}

func (bt *BidirectionalTree) Nodes() []*BidirectionalNode {
	nodes := make([]*BidirectionalNode, len(bt.nodesByCid))
	i := 0
	for _,node := range bt.nodesByCid {
		nodes[i] = node
		i++
	}
	return nodes
}

func (bt *BidirectionalTree) Resolve(path []string) (interface{}, []string, error) {
	//fmt.Printf("resolving: %v\n", path)
	root,ok := bt.nodesByCid[bt.Tip.KeyString()]
	if !ok {
		//fmt.Printf("error resolving\n")
		return nil, nil, &ErrorCode{Code: ErrMissingRoot}
	}
	if len(path) == 0 || len(path) == 1 && path[0] == "/" {
		//fmt.Printf("path length == 1\n")
		rootMap, err := root.AsMap()
		if err != nil {
			return nil, nil, &ErrorCode{Code: ErrEncodingError, Memo: fmt.Sprintf("error encoding: %v", err)}
		}
		return rootMap, []string{}, nil
	}
	//fmt.Printf("resolving to root\n")
	return root.Resolve(bt, path)
}

func (bt *BidirectionalTree) keepWalker(node *BidirectionalNode) map[string]*BidirectionalNode {
	toKeep := map[string]*BidirectionalNode{node.Node.Cid().KeyString(): node}

	links := node.Node.Links()
	for _,l := range links {
		linkNode,ok := bt.nodesByCid[l.Cid.KeyString()]
		if ok {
			toKeep[node.Node.Cid().KeyString()] = linkNode
			decedents := bt.keepWalker(linkNode)
			for k,v := range decedents {
				toKeep[k] = v
			}
		}
	}
	return toKeep
}

func (bt *BidirectionalTree) Prune() {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	toKeep := bt.keepWalker(bt.Get(bt.Tip))
	for cid,node := range bt.nodesByCid {
		_,ok := toKeep[cid]
		if !ok {
			delete(bt.nodesByCid, cid)
			delete(bt.nodesByStaticId, node.id)
		}
	}
}

func (bt *BidirectionalTree) createLinks(path []string, node *cbornode.Node) error {

	var idx int

	for i := len(path);i >= 0; i-- {
		_,_,err := bt.Resolve(path[0:i])
		if err == nil {
			 idx = i
			 break
		} else {
			if err.(*ErrorCode).Code != ErrMissingPath {
				return &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("unkown error: %v", err)}
			}
		}
	}

	var last *cbornode.Node
	last = node

	nodes := []*cbornode.Node{node}

	sw := &SafeWrap{}
	for i := len(path)-1;i > idx; i-- {
		obj := make(map[string]*cid.Cid)
		obj[path[i]] = last.Cid()
		newNode := sw.WrapObject(obj)
		nodes = append(nodes, newNode)
		last = newNode
	}
	if sw.Err != nil {
		return &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error wrapping: %v", sw.Err)}
	}

	bt.AddNodes(nodes...)

	//fmt.Printf("calling set with: %v\n", path[0:idx+1])
	return bt.Set(path[0:idx+1], last.Cid())
}

func (bt *BidirectionalTree) Set(pathAndKey []string, val interface{}) error {
	return bt.set(pathAndKey, val, false)
}

func (bt *BidirectionalTree) SetAsLink(pathAndKey []string, val interface{}) error {
	tree,ok := val.(*BidirectionalTree)
	if ok {
		newNodes := make([]*cbornode.Node, len(tree.nodesByStaticId))
		i := 0
		for _,node := range tree.nodesByStaticId {
			newNodes[i] = node.Node
			i++
		}
		bt.AddNodes(newNodes...)
		rootMap,err := tree.Get(tree.Tip).AsMap()
		if err != nil {
			return &ErrorCode{Code: ErrBadInput, Memo: "bad map"}
		}
		val = rootMap
	}

	return bt.set(pathAndKey, val, true)
}

func (bt *BidirectionalTree) set(pathAndKey []string, val interface{}, asLink bool) error {

	var path []string
	var key string

	switch len(pathAndKey) {
	case 0:
		return &ErrorCode{Code: ErrBadInput, Memo: "must pass in a key"}
	case 1:
		path = []string{}
		key = pathAndKey[0]
	default:
		path = pathAndKey[0:len(pathAndKey)-1]
		key = pathAndKey[len(pathAndKey)-1]
	}
	//fmt.Printf("setting %v, key: %v\n", path, key)

	existing, remaining, err := bt.Resolve(path)
	if err != nil {
		if err.(*ErrorCode).Code == ErrMissingPath {
			newObj := map[string]interface{}{key: val}
			sw := &SafeWrap{}
			wrapped := sw.WrapObject(newObj)
			if sw.Err != nil {
				return err
			}
			return bt.createLinks(path, wrapped)
		} else {
			fmt.Printf("error resolving: %v\n", path)
			return err
		}
	}

	//spew.Dump("existing: ", existing)

	if len(remaining) != 0 {
		return &ErrorCode{Code: ErrInvalidInput, Memo: "The selected path didn't resolve."}
	}

	//fmt.Printf("tip: %v\n", bt.Tip.String())

	existingCbor,err := fromJsonish(existing)
	if err != nil {
		return &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting object: %v", err)}
	}

	existingCid := existingCbor.Cid()

	if asLink {
		newNode,err := fromJsonish(val)
		if err != nil {
			return &ErrorCode{Code: ErrBadInput, Memo: fmt.Sprintf("error converting val: %v", err)}
		}
		bt.AddNodes(newNode)
		existing.(map[string]interface{})[key] = newNode.Cid()
	} else {
		existing.(map[string]interface{})[key] = val
	}

	wrappedModified,err := fromJsonish(existing)
	if err != nil {
		return &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting object: %v", err)}
	}

	//fmt.Printf("swapping: %v for %v\n", existingCid.String(), wrappedModified.Cid().String())
	err = bt.Swap(existingCid, wrappedModified)
	if err != nil {
		return &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error swapping: %v", err)}
	}
	//spew.Dump("after swap:", bt.nodesByCid[wrappedModified.Cid().KeyString()])
	return nil
}

func (bt *BidirectionalTree) Swap(oldCid *cid.Cid, newNode *cbornode.Node) error {
	existing,ok := bt.nodesByCid[oldCid.KeyString()]
	if !ok {
		//log.Printf("existing not found")
		return &ErrorCode{Code:ErrMissingPath, Memo: fmt.Sprintf("cannot find %s", oldCid.String())}
	}

	//newMapped,_ := (&BidirectionalNode{Node: newNode}).AsMap()
	//oldMapped,_ := existing.AsMap()
	//log.Printf("swapping: %s from: \n %v \nto:\n %v with CID: %v", oldCid.String(), oldMapped, newMapped, newNode.Cid())

	//fmt.Println("existing:")
	//existing.Dump()

	existingCid := existing.Node.Cid()
	existing.Node = newNode
	delete(bt.nodesByCid, existingCid.KeyString())

	bt.nodesByCid[newNode.Cid().KeyString()] = existing

	//fmt.Printf("existing tip: %v\n", bt.Tip.String())

	if bt.Tip.KeyString() == oldCid.KeyString() {
		bt.Tip = newNode.Cid()
	} else {
		for parentId := range existing.Parents {
			parent := bt.nodesByStaticId[parentId]
			//fmt.Println("parent")
			//parent.Dump()
			newParentJsonish := make(map[string]interface{})
			err := cbornode.DecodeInto(parent.Node.RawData(), &newParentJsonish)
			if err != nil {
				return fmt.Errorf("error decoding: %v", err)
			}

			//fmt.Println("before:")
			//spew.Dump(newParentJsonish)

			err = updateLinks(newParentJsonish, existingCid, newNode.Cid())
			if err != nil {
				return fmt.Errorf("error updating links: %v", err)
			}

			//fmt.Println("after:")
			//spew.Dump(newParentJsonish)

			newParentNode,err := fromJsonish(newParentJsonish)
			if err != nil {
				return fmt.Errorf("error getting Node: %v", err)
			}
			//fmt.Println("new parent Node")
			obj := make(map[string]interface{})
			cbornode.DecodeInto(newParentNode.RawData(), &obj)
			//spew.Dump(obj)

			bt.Swap(parent.Node.Cid(), newParentNode)
		}
	}

	bt.updateParents()

	//fmt.Println("after tree")
	//bt.Dump()

	return nil
}

func (bt *BidirectionalTree) ByteSize() int64 {
	var length int64
	for _,node := range bt.nodesByCid {
		length += int64(len(node.Node.RawData()))
	}
	return length
}

func (bn *BidirectionalNode) Dump(bt *BidirectionalTree, isLink bool) map[string]interface{} {
	nodeMap,_ := bn.AsMap()
	for k,v := range nodeMap {
		switch v := v.(type) {
		case *cid.Cid:
			node := bt.Get(v)
			if node == nil {
				nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
			} else {
				nodeMap[k] = node.Dump(bt, true)
			}
		case cid.Cid:
			node := bt.Get(&v)
			if node == nil {
				nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
			} else {
				nodeMap[k] = node.Dump(bt, true)
			}
		default:
			continue
		}
	}
	if isLink {
		nodeMap["_isLink"] = true
	}
	return nodeMap
}

func (bt *BidirectionalTree) Dump() string {
	rootNode := bt.Get(bt.Tip)
	nodes := make([]string, len(bt.nodesByCid))
	i := 0
	for _,node := range bt.nodesByCid {
		nodes[i] = node.Node.Cid().String()
		i++
	}
	return fmt.Sprintf(`
Tip: %s,
Tree:	
%s

Nodes: %v

	`,
		bt.Tip.String(),
		spew.Sdump(rootNode.Dump(bt, false)),
		nodes)
}


type SafeWrap struct {
	Err error
}

func (sf *SafeWrap) WrapObject(obj interface{}) *cbornode.Node {
	if sf.Err != nil {
		return nil
	}

	node,err := cbornode.WrapObject(obj, multihash.SHA2_256, -1)
	sf.Err = err
	return node
}

func (sf *SafeWrap) Decode(data []byte) *cbornode.Node {
	if sf.Err != nil {
		return nil
	}

	node,err := cbornode.Decode(data, multihash.SHA2_256, -1)
	sf.Err = err
	return node
}

func fromJsonish(obj interface{}) (*cbornode.Node, error) {
	sw := &SafeWrap{}
	node := sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error marshaling: %v", sw.Err)
	}
	return node,nil
}


func updateLinks(obj interface{}, oldCid *cid.Cid, newCid *cid.Cid) error {
	switch obj := obj.(type) {
		case map[interface{}]interface{}:
			for _, v := range obj {
				if err := updateLinks(v, oldCid, newCid); err != nil {
					return err
				}
			}
			return nil
		case map[string]interface{}:
			for ks, v := range obj {
				switch v.(type) {
				case *cid.Cid:
					if v.(*cid.Cid).String() == oldCid.String() {
						obj[ks] = newCid
					}
				case cid.Cid:
					ptr := v.(cid.Cid)
					if (&ptr).String() == oldCid.String() {
						obj[ks] = newCid
					}
				default:
					if err := updateLinks(v, oldCid, newCid); err != nil {
						return err
					}
				}
			}
			return nil
		case []interface{}:
			for _, v := range obj {
				if err := updateLinks(v, oldCid, newCid); err != nil {
					return err
				}
			}
			return nil
		default:
			return nil
		}
}
