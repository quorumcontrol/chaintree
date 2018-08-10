package nodestore

import (
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipld-cbor"
)

// CidString is the KeyString() of a CID
type CidString string

// Cid returns the CID from the CidString (which is the KeyString format)
func (cs CidString) Cid() *cid.Cid {
	cID, _ := cid.Cast([]byte(string(cs)))
	return cID
}

// ToCidString takes a CID and returns its map key (CidString)
func ToCidString(id *cid.Cid) CidString {
	return CidString(id.KeyString())
}

// UpdateMap is a map of the old CID (in CidString form) to new CID in CID form
type UpdateMap map[CidString]*cid.Cid

// Contains returns true if the UpdateMap contains a CID for an existing node
func (um UpdateMap) Contains(cid *cid.Cid) bool {
	_, ok := um[CidString(cid.KeyString())]
	return ok
}

// MergeUpdateMap merges two UpdateMaps and returns a new UpdateMap
func MergeUpdateMap(um UpdateMap, other UpdateMap) (newMap UpdateMap) {
	newMap = make(UpdateMap)
	for k, v := range um {
		newMap[k] = v
	}
	for k, v := range other {
		newMap[k] = v
	}
	return newMap
}

// NodeStore is an interface for getting and setting nodes
// it allows you to keep track of referenced nodes so you can, for instance, update a whole tree
// without having to manually update links
type NodeStore interface {
	// GetNode takes a CID and returns a cbornode
	GetNode(nodeCid *cid.Cid) (*cbornode.Node, error)
	// CreateNode takes any object and converts it to a cbornode and then returns the saved CID
	CreateNode(obj interface{}) (*cbornode.Node, error)
	// CreateNodeFromBytes creates a new node, but using cbor bytes instead of a native GO object
	CreateNodeFromBytes(nodeBytes []byte) (*cbornode.Node, error)
	// StoreNode just takes a cbornode and sets references, etc in the storage
	StoreNode(node *cbornode.Node) error
	// GetReferences returns a slice of CIDs that contain a link to the CID in the to argument
	GetReferences(to *cid.Cid) ([]*cid.Cid, error)
	// UpdateNode adds the new obj to the NodeStore, then walks the references to the old
	// CID and updates their links to reflect the new object. It then returns the new, updated cbor node
	// for obj and the "tips" of the reference tree: that is the last objects with no parents
	UpdateNode(existing *cid.Cid, obj interface{}) (updatedNode *cbornode.Node, updates UpdateMap, err error)
	// Swap takes an existing CID, and just swaps it out for the new node.
	// It is up to the caller to make sure that any child nodes are already part of the store.
	Swap(existing *cid.Cid, node *cbornode.Node) (updates UpdateMap, err error)
	// DeleteNode deletes a node from the store, it will no-op if the node is referenced by other nodes
	DeleteIfUnreferenced(nodeCid *cid.Cid) error
	// DeleteTree removes everything in a tree starting from a tip as long as none of the nodes have
	// references
	DeleteTree(tip *cid.Cid) error

	// Resolve takes a tip, and walks through the NodeStore until finding a value
	Resolve(tip *cid.Cid, path []string) (val interface{}, remaining []string, err error)
}
