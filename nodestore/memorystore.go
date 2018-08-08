package nodestore

import (
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"

	"github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
)

type memoryNodeHolderMap map[string]*cbornode.Node
type referenceHolderMap map[string]map[string]bool

// MemoryNodeStore implements a NodeStore in memory
// only.
type MemoryNodeStore struct {
	nodes memoryNodeHolderMap
	refs  referenceHolderMap
	lock  *sync.RWMutex
}

// just confirm that MemoryNodeStore conforms to interface
var _ NodeStore = (*MemoryNodeStore)(nil)

// Initialize sets up the initial state and
// creates maps,etc
func (mns *MemoryNodeStore) Initialize() {
	mns.nodes = make(memoryNodeHolderMap)
	mns.refs = make(referenceHolderMap)
	mns.lock = &sync.RWMutex{}
}

// CreateNode takes any object and converts it to a cbornode and then returns the saved CID
func (mns *MemoryNodeStore) CreateNode(obj interface{}) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}

	return mns.createNodeFromCborNode(node)
}

// CreateNodeFromBytes implements the NodeStore interface
func (mns *MemoryNodeStore) CreateNodeFromBytes(data []byte) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.Decode(data)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return mns.createNodeFromCborNode(node)
}

// GetNode returns a cbornode for a CID
func (mns *MemoryNodeStore) GetNode(cid *cid.Cid) (node *cbornode.Node, err error) {
	node, ok := mns.nodes[cid.KeyString()]
	if !ok {
		return nil, nil
	}
	return
}

// GetReferences implements NodeStore GetReferences
func (mns *MemoryNodeStore) GetReferences(to *cid.Cid) (refs []*cid.Cid, err error) {
	mns.lock.RLock()
	defer mns.lock.RUnlock()
	toRefs, ok := mns.refs[to.KeyString()]
	if !ok {
		return nil, nil
	}

	refs = make([]*cid.Cid, len(toRefs))

	i := 0
	for ref := range toRefs {
		cid, err := cid.Cast([]byte(ref))
		if err != nil {
			return nil, fmt.Errorf("error casting CID: %v", err)
		}
		refs[i] = cid
		i++
	}
	return
}

// UpdateNode implements NodeStore UpdateNode
func (mns *MemoryNodeStore) UpdateNode(existing *cid.Cid, obj interface{}) (updated *cbornode.Node, tips []*cid.Cid, err error) {
	updated, err = mns.CreateNode(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating node: %v", err)
	}
	refs, err := mns.GetReferences(existing)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting references: %v", err)
	}

	if len(refs) == 0 {
		return updated, []*cid.Cid{updated.Cid()}, nil
	}

	for _, ref := range refs {
		reffedNode, err := mns.GetNode(ref)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting node (%s): %v", ref.String(), err)
		}
		reffedObj, err := CborNodeToObj(reffedNode)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting node to obj (%s): %v", ref.String(), err)
		}
		err = updateLinks(reffedObj, existing, updated.Cid())
		if err != nil {
			return nil, nil, fmt.Errorf("error updating links (%s): %v", ref.String(), err)
		}
		refUpd, refTip, err := mns.UpdateNode(ref, reffedObj)
		if err != nil {
			return nil, nil, fmt.Errorf("error updating reference (%s): %v", ref.String(), err)
		}
		if len(refTip) == 1 && refTip[0].Equals(refUpd.Cid()) {
			tips = append(tips, refTip[0])
		}
	}
	return updated, tips, nil
}

// DeleteIfUnreferenced implements the NodeStore DeleteIfUnreferenced interface.
func (mns *MemoryNodeStore) DeleteIfUnreferenced(nodeCid *cid.Cid) error {
	refs, err := mns.GetReferences(nodeCid)
	if err != nil {
		return fmt.Errorf("error getting refs: %v", err)
	}
	if len(refs) > 0 {
		return nil
	}

	existing, err := mns.GetNode(nodeCid)
	if err != nil {
		return fmt.Errorf("error getting existing: %v", err)
	}
	for _, link := range existing.Links() {
		mns.deleteReferences(link.Cid, nodeCid)
	}

	delete(mns.nodes, nodeCid.KeyString())
	return nil
}

// DeleteTree implements the NodeStore DeleteTree interface
func (mns *MemoryNodeStore) DeleteTree(tip *cid.Cid) error {
	tipNode, err := mns.GetNode(tip)
	if err != nil {
		return fmt.Errorf("error getting tip: %v", err)
	}

	links := tipNode.Links()

	for _, link := range links {
		mns.deleteReferences(link.Cid, tip)
		err := mns.DeleteTree(link.Cid)
		if err != nil {
			return fmt.Errorf("error deleting: %v", err)
		}
	}
	return mns.DeleteIfUnreferenced(tip)
}

func (mns *MemoryNodeStore) createNodeFromCborNode(node *cbornode.Node) (*cbornode.Node, error) {
	nodeCid := node.Cid()

	mns.lock.Lock()
	defer mns.lock.Unlock()

	mns.nodes[nodeCid.KeyString()] = node

	links := node.Links()
	for _, link := range links {
		err := mns.saveReferences(link.Cid, nodeCid)
		if err != nil {
			return nil, fmt.Errorf("error saving reference: %v", err)
		}
	}
	return node, nil
}

func (mns *MemoryNodeStore) saveReferences(to *cid.Cid, from ...*cid.Cid) error {
	toRefs, ok := mns.refs[to.KeyString()]
	if !ok {
		toRefs = make(map[string]bool)
	}
	for _, fromID := range from {
		toRefs[fromID.KeyString()] = true
	}
	mns.refs[to.KeyString()] = toRefs
	return nil
}

func (mns *MemoryNodeStore) deleteReferences(to *cid.Cid, from ...*cid.Cid) error {
	toRefs, ok := mns.refs[to.KeyString()]
	if !ok {
		toRefs = make(map[string]bool)
	}
	for _, fromID := range from {
		delete(toRefs, fromID.KeyString())
	}
	if len(toRefs) > 0 {
		mns.refs[to.KeyString()] = toRefs
	} else {
		delete(mns.refs, to.KeyString())
	}
	return nil
}

// CborNodeToObj takes a cbornode and returns a map[string]interface{} representation
// of the data. Useful for setting values, etc
func CborNodeToObj(node *cbornode.Node) (obj map[string]interface{}, err error) {
	obj = make(map[string]interface{})
	err = cbornode.DecodeInto(node.RawData(), &obj)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return
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
				if v.(*cid.Cid).Equals(oldCid) {
					obj[ks] = newCid
				}
			case cid.Cid:
				ptr := v.(cid.Cid)
				if (&ptr).Equals(oldCid) {
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
