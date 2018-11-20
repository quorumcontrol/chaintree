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

var nodeBucket = []byte("nodeStoreNodes")
var trueByte = []byte{byte(1)}

// StorageBasedStore is a NodeStore that can take an arbitrary storage back end
type StorageBasedStore struct {
	store  storage.Storage
	locker *namedlocker.NamedLocker
}

var _ NodeStore = (*StorageBasedStore)(nil)

// NewStorageBasedStore creates a new NodeStore using the store argument for the backend
func NewStorageBasedStore(store storage.Storage) *StorageBasedStore {
	store.CreateBucketIfNotExists(nodeBucket)
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
	nodeBytes, err := sbs.store.Get(nodeBucket, []byte(cid.KeyString()))
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

// GetReferences implements NodeStore GetReferences
func (sbs *StorageBasedStore) GetReferences(to cid.Cid) (refs []cid.Cid, err error) {
	bucketName := refBucketName(to)
	sbs.locker.RLock(string(bucketName))
	defer sbs.locker.RUnlockAndDelete(string(bucketName))
	if !sbs.store.BucketExists(bucketName) {
		return nil, nil
	}

	keys, err := sbs.store.GetKeys(bucketName)
	if err != nil {
		return nil, fmt.Errorf("error getting keys from storage: %v", err)
	}

	refs = make([]cid.Cid, len(keys))

	for i, keyBytes := range keys {
		cid, err := cid.Cast(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("error casting CID: %v", err)
		}
		refs[i] = cid
	}
	return
}

// UpdateNode implements NodeStore UpdateNode
func (sbs *StorageBasedStore) UpdateNode(existing cid.Cid, obj interface{}) (updatedNode *cbornode.Node, updates UpdateMap, err error) {
	updatedNode, err = objToCbor(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating node: %v", err)
	}
	updates, err = sbs.Swap(existing, updatedNode)
	if err != nil {
		return nil, nil, fmt.Errorf("error swapping: %v", err)
	}
	return
}

// Swap implements the NodeStore interface.
func (sbs *StorageBasedStore) Swap(existing cid.Cid, updatedNode *cbornode.Node) (updates UpdateMap, err error) {
	err = sbs.StoreNode(updatedNode)
	if err != nil {
		return nil, fmt.Errorf("error storing: %v", err)
	}
	if updates == nil {
		updates = make(UpdateMap)
	}
	refs, err := sbs.GetReferences(existing)
	if err != nil {
		return nil, fmt.Errorf("error getting references: %v", err)
	}

	if len(refs) == 0 {
		return UpdateMap{ToCidString(existing): updatedNode.Cid()}, nil
	}

	updates[ToCidString(existing)] = updatedNode.Cid()

	for _, ref := range refs {
		reffedNode, err := sbs.GetNode(ref)
		if err != nil {
			return nil, fmt.Errorf("error getting node (%s): %v", ref.String(), err)
		}
		reffedObj, err := CborNodeToObj(reffedNode)
		if err != nil {
			return nil, fmt.Errorf("error converting node to obj (%s): %v", ref.String(), err)
		}
		err = updateLinks(reffedObj, existing, updatedNode.Cid())
		if err != nil {
			return nil, fmt.Errorf("error updating links (%s): %v", ref.String(), err)
		}
		_, refUpdates, err := sbs.UpdateNode(ref, reffedObj)
		if err != nil {
			return nil, fmt.Errorf("error updating reference (%s): %v", ref.String(), err)
		}
		updates = MergeUpdateMap(updates, refUpdates)
	}
	return updates, nil
}

// DeleteIfUnreferenced implements the NodeStore DeleteIfUnreferenced interface.
func (sbs *StorageBasedStore) DeleteIfUnreferenced(nodeCid cid.Cid) error {
	refs, err := sbs.GetReferences(nodeCid)
	if err != nil {
		return fmt.Errorf("error getting refs: %v", err)
	}
	if len(refs) > 0 {
		return nil
	}

	existing, err := sbs.GetNode(nodeCid)
	if err != nil {
		return fmt.Errorf("error getting existing: %v", err)
	}
	for _, link := range existing.Links() {
		sbs.deleteReferences(link.Cid, nodeCid)
	}

	return sbs.store.Delete(nodeBucket, []byte(nodeCid.KeyString()))
}

// DeleteTree implements the NodeStore DeleteTree interface
func (sbs *StorageBasedStore) DeleteTree(tip cid.Cid) error {
	tipNode, err := sbs.GetNode(tip)
	if err != nil {
		return fmt.Errorf("error getting tip: %v", err)
	}

	links := tipNode.Links()

	for _, link := range links {
		sbs.deleteReferences(link.Cid, tip)
		err := sbs.DeleteTree(link.Cid)
		if err != nil {
			return fmt.Errorf("error deleting: %v", err)
		}
	}
	return sbs.DeleteIfUnreferenced(tip)
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

	err := sbs.store.Set(nodeBucket, []byte(node.Cid().KeyString()), node.RawData())
	if err != nil {
		return fmt.Errorf("error saving storage: %v", err)
	}
	links := node.Links()
	for _, link := range links {
		err := sbs.saveReferences(link.Cid, nodeCid)
		if err != nil {
			return fmt.Errorf("error saving reference: %v", err)
		}
	}
	return nil
}

func (sbs *StorageBasedStore) saveReferences(to cid.Cid, from ...cid.Cid) error {
	bucketName := refBucketName(to)
	sbs.locker.Lock(string(bucketName))
	defer sbs.locker.UnlockAndDelete(string(bucketName))

	sbs.store.CreateBucketIfNotExists(bucketName)
	for _, fromID := range from {
		err := sbs.store.Set(bucketName, []byte(fromID.KeyString()), trueByte)
		if err != nil {
			return fmt.Errorf("error storing reference: %v", err)
		}
	}
	return nil
}

func refBucketName(nodeID cid.Cid) []byte {
	return []byte(nodeID.KeyString() + "-refs")
}

func (sbs *StorageBasedStore) deleteReferences(to cid.Cid, from ...cid.Cid) error {
	bucketName := refBucketName(to)
	sbs.locker.Lock(string(bucketName))
	defer sbs.locker.UnlockAndDelete(string(bucketName))
	if !sbs.store.BucketExists(bucketName) {
		return nil
	}

	for _, fromID := range from {
		sbs.store.Delete(bucketName, []byte(fromID.KeyString()))
	}
	keys, err := sbs.store.GetKeys(bucketName)
	if err != nil {
		return fmt.Errorf("error getting keys: %v", err)
	}
	if len(keys) == 0 {
		sbs.store.DeleteBucket(bucketName)
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

func updateLinks(obj interface{}, oldCid cid.Cid, newCid cid.Cid) error {
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
			case cid.Cid:
				if v.(cid.Cid).Equals(oldCid) {
					obj[ks] = newCid
				}
			case *cid.Cid:
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
