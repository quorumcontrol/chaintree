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
func (mns *MemoryNodeStore) CreateNode(obj interface{}) (*cbornode.Node, error) {
	sw := safewrap.SafeWrap{}
	node := sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	mns.nodes[node.Cid().KeyString()] = node
	return node, nil
}

func (mns *MemoryNodeStore) GetNode(cid *cid.Cid) (*cbornode.Node, error) {
	node, ok := mns.nodes[cid.KeyString()]
	if !ok {
		return nil, nil
	}
	return node, nil
}

func (mns *MemoryNodeStore) SaveReference(from, to *cid.Cid) error {
	mns.lock.Lock()
	defer mns.lock.Unlock()
	toRefs, ok := mns.refs[to.KeyString()]
	if !ok {
		toRefs = make(map[string]bool)
	}
	toRefs[from.KeyString()] = true
	mns.refs[to.KeyString()] = toRefs
	return nil
}

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
	return refs, nil
}

func (mns *MemoryNodeStore) UpdateNode(existing *cid.Cid, obj interface{}) (*cbornode.Node, error) {
	return nil, fmt.Errorf("not yet implemented")
}
