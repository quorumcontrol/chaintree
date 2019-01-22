package nodestore

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/namedlocker"
	"github.com/quorumcontrol/storage"
)

var trueByte = []byte{byte(1)}

// StorageBasedStore is a NodeStore that can take an arbitrary storage back end
type StorageBasedStore struct {
	store  storage.Storage
	locker *namedlocker.NamedLocker
}

var _ NodeStore = (*StorageBasedStore)(nil)

// NewStorageBasedStore creates a new NodeStore using the store argument for the backend
func NewStorageBasedStore(store storage.Storage) *StorageBasedStore {
	return &StorageBasedStore{
		store:  store,
		locker: namedlocker.NewNamedLocker(),
	}
}

// CreateNode takes any object and converts it to a cbornode and then returns the saved CID
func (sbs *StorageBasedStore) CreateNode(obj interface{}) (node *cbornode.Node, err error) {
	node, err = objToCbor(obj)
	if err != nil {
		return nil, fmt.Errorf("error converting obj: %v", err)
	}
	return node, sbs.StoreNode(node)
}

// CreateNodeFromBytes implements the NodeStore interface
func (sbs *StorageBasedStore) CreateNodeFromBytes(data []byte) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.Decode(data)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return node, sbs.StoreNode(node)
}

// GetNode returns a cbornode for a CID
func (sbs *StorageBasedStore) GetNode(cid cid.Cid) (node *cbornode.Node, err error) {
	nodeBytes, err := sbs.store.Get([]byte(cid.KeyString()))
	if err != nil {
		return nil, fmt.Errorf("error getting cid: %v", err)
	}
	if len(nodeBytes) == 0 {
		return nil, nil
	}
	sw := &safewrap.SafeWrap{}
	node = sw.Decode(nodeBytes)
	return node, sw.Err
}

// DeleteNode implements the NodeStore DeleteNode interface.
func (sbs *StorageBasedStore) DeleteNode(nodeCid cid.Cid) error {
	return sbs.store.Delete([]byte(nodeCid.KeyString()))
}

// DeleteTree implements the NodeStore DeleteTree interface
func (sbs *StorageBasedStore) DeleteTree(tip cid.Cid) error {
	tipNode, err := sbs.GetNode(tip)
	if err != nil {
		return fmt.Errorf("error getting tip: %v", err)
	}

	links := tipNode.Links()

	for _, link := range links {
		err := sbs.DeleteTree(link.Cid)
		if err != nil {
			return fmt.Errorf("error deleting: %v", err)
		}
	}
	return sbs.DeleteNode(tip)
}

// Resolve implements the NodeStore interface
func (sbs *StorageBasedStore) Resolve(tip cid.Cid, path []string) (val interface{}, remaining []string, err error) {
	node, err := sbs.GetNode(tip)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting node (%s): %v", tip.String(), err)
	}
	val, remaining, err = node.Resolve(path)
	if err != nil {
		// If the link is just missing, then just return the whole path as remaining, with a nil value
		// instead of an error
		if err == cbornode.ErrNoSuchLink {
			return nil, path, nil
		}
		return nil, nil, err
	}

	switch val.(type) {
	case *format.Link:
		linkNode, err := sbs.GetNode(val.(*format.Link).Cid)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting linked node (%s): %v", linkNode.Cid().String(), err)
		}
		if linkNode != nil {
			return sbs.Resolve(linkNode.Cid(), remaining)
		}
		return nil, remaining, nil
	default:
		return val, remaining, err
	}
}

// StoreNode implements the NodeStore interface
func (sbs *StorageBasedStore) StoreNode(node *cbornode.Node) error {
	nodeCid := node.Cid()
	sbs.locker.Lock(nodeCid.KeyString())
	defer sbs.locker.UnlockAndDelete(nodeCid.KeyString())

	err := sbs.store.Set([]byte(node.Cid().KeyString()), node.RawData())
	if err != nil {
		return fmt.Errorf("error saving storage: %v", err)
	}
	return nil
}

// CborNodeToObj takes a cbornode and returns a map[string]interface{} representation
// of the data. Useful for setting values, etc
func CborNodeToObj(node *cbornode.Node) (obj interface{}, err error) {
	err = cbornode.DecodeInto(node.RawData(), &obj)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return
}

func objToCbor(obj interface{}) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return
}
