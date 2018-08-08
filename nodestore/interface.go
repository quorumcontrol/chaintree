package nodestore

import (
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipld-cbor"
)

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
	// GetReferences returns a slice of CIDs that contain a link to the CID in the to argument
	GetReferences(to *cid.Cid) ([]*cid.Cid, error)
	// UpdateNode adds the new obj to the NodeStore, then walks the references to the old
	// CID and updates their links to reflect the new object. It then returns the new, updated cbor node
	// for obj and the "tips" of the reference tree: that is the last objects with no parents
	UpdateNode(existing *cid.Cid, obj interface{}) (updated *cbornode.Node, tips []*cid.Cid, err error)
	// DeleteNode deletes a node from the store, it will no-op if the node is referenced by other nodes
	DeleteIfUnreferenced(nodeCid *cid.Cid) error
	// DeleteTree removes everything in a tree starting from a tip as long as none of the nodes have
	// references
	DeleteTree(tip *cid.Cid) error
}
