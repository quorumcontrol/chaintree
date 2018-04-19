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
	"errors"
	"encoding/json"
	"bytes"
	"github.com/davecgh/go-spew/spew"
)

type nodeId int

type BidirectionalTree struct {
	Tip             *cid.Cid
	counter         int
	nodesByStaticId map[nodeId]*bidirectionalNode
	nodesByCid      map[string]*bidirectionalNode
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
	ErrMissingRoot = 0
	ErrMissingPath = 1
	ErrInvalidInput = 2
	ErrEncodingError = 3
	ErrBadInput = 4
	ErrUnknown = 99
)


type bidirectionalNode struct {
	parents []nodeId
	id nodeId
	node    *cbornode.Node
}

func NewBidirectionalTree() *BidirectionalTree {
	return &BidirectionalTree{
		counter: 0,
		nodesByStaticId: make(map[nodeId]*bidirectionalNode),
		nodesByCid: make(map[string]*bidirectionalNode),
	}
}

func (bn *bidirectionalNode) asJsonish() (map[string]interface{}, error) {
	newParentJsonish := make(map[string]interface{})
	err := cbornode.DecodeInto(bn.node.RawData(), &newParentJsonish)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return newParentJsonish, nil
}

func (bn *bidirectionalNode) Resolve(tree *BidirectionalTree, path []string) (interface{}, []string, error) {
	val, remaining, err := bn.node.Resolve(path)
	if err != nil {
		//fmt.Printf("error resolving: %v", err)
		return nil, nil, &ErrorCode{Code: ErrMissingPath, Memo: fmt.Sprintf("error resolving: %v", err)}
	}
	//spew.Dump("resolved on node", val)

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

func (bt *BidirectionalTree) AddNodes(nodes ...*cbornode.Node) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	for i,node := range nodes {
		bidiNode := &bidirectionalNode{
			node: node,
			id: nodeId(bt.counter + i),
			parents: make([]nodeId,0),
		}
		bt.nodesByStaticId[bidiNode.id] = bidiNode
		bt.nodesByCid[node.Cid().KeyString()] = bidiNode
	}
	bt.counter += len(nodes)

	for _,bidiNode := range bt.nodesByStaticId {
		links := bidiNode.node.Links()
		for _,link := range links {
			existing,ok := bt.nodesByCid[link.Cid.KeyString()]
			if ok {
				existing.parents = append(existing.parents, bidiNode.id)
			}
		}
	}
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
		rootMap, err := root.asJsonish()
		if err != nil {
			return nil, nil, &ErrorCode{Code: ErrEncodingError, Memo: fmt.Sprintf("error encoding: %v", err)}
		}
		return rootMap, []string{}, nil
	}
	//fmt.Printf("resolving to root\n")
	return root.Resolve(bt, path)
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
	//fmt.Printf("swapping: %s \n", oldCid.String())

	existing,ok := bt.nodesByCid[oldCid.KeyString()]
	if !ok {
		//fmt.Printf("existing not found")
		return &ErrorCode{Code:ErrMissingPath, Memo: fmt.Sprintf("cannot find %s", oldCid.String())}
	}
	//fmt.Println("existing:")
	//existing.dump()

	existingCid := existing.node.Cid()
	existing.node = newNode
	delete(bt.nodesByCid, existingCid.KeyString())

	bt.nodesByCid[newNode.Cid().KeyString()] = existing

	//fmt.Printf("existing tip: %v\n", bt.Tip.String())

	if bt.Tip.KeyString() == oldCid.KeyString() {
		bt.Tip = newNode.Cid()
	} else {
		for _,parentId := range existing.parents {
			parent := bt.nodesByStaticId[parentId]
			//fmt.Println("parent")
			//parent.dump()
			newParentJsonish := make(map[string]interface{})
			err := cbornode.DecodeInto(parent.node.RawData(), &newParentJsonish)
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
				return fmt.Errorf("error getting node: %v", err)
			}
			//fmt.Println("new parent node")
			obj := make(map[string]interface{})
			cbornode.DecodeInto(newParentNode.RawData(), &obj)
			//spew.Dump(obj)

			if parent.node.Cid() == bt.Tip {
				bt.Tip = newParentNode.Cid()
			}

			bt.Swap(parent.node.Cid(), newParentNode)
		}
	}


	//fmt.Println("after tree")
	//bt.dump()

	return nil
}

func (bn *bidirectionalNode) dump() {
	spew.Dump(bn)
	obj := make(map[string]interface{})
	cbornode.DecodeInto(bn.node.RawData(), &obj)
	spew.Dump(obj)
}

func (bt *BidirectionalTree) dump() {
	spew.Dump(bt)
	for _,n := range bt.nodesByStaticId {
		fmt.Printf("node: %d", n.id)
		n.dump()
	}
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

func fromJsonish(obj interface{}) (*cbornode.Node, error) {
	jBytes,err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshaling: %v", err)
	}
	node,err := cbornode.FromJson(bytes.NewReader(jBytes), multihash.SHA2_256, -1)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return node, nil
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
				//fmt.Printf("k: %s\n", ks)

				if ks == "/" {
					vs, ok := v.(string)
					if ok {
						if vs == oldCid.String() {
							//fmt.Printf("updating link from %s to %s\n", oldCid.String(), newCid.String())
							obj[ks] = newCid.String()
						}
					} else {
						return errors.New("error, link was not a string")
					}
				} else {
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


