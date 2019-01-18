package dag

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreapi"
	coreapiiface "github.com/ipsn/go-ipfs/core/coreapi/interface"
	coreapiopt "github.com/ipsn/go-ipfs/core/coreapi/interface/options"
	cid "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-cid"
	ds "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-datastore"
	dsq "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-datastore/query"
)

// IpfsDatastore uses a standard Go map for internal storage.
type IpfsDatastore struct {
	Node *core.IpfsNode
	api  coreapiiface.CoreAPI
	ctx  context.Context
}

// NewIpfsDatastore constructs a IpfsDatastore
func NewIpfsDatastore(ipfsConfig *core.BuildCfg) (d *IpfsDatastore) {
	ctx := context.Background()

	node, err := core.NewNode(ctx, ipfsConfig)
	if err != nil {
		return nil
	}

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil
	}

	return &IpfsDatastore{
		Node: node,
		api:  api,
		ctx:  ctx,
	}
}

func parseKeyToCid(key ds.Key) (*cid.Cid, bool) {
	rawKey := key.List()
	cid, err := cid.Parse(rawKey[len(rawKey)-1])
	return &cid, err == nil
}

// Put implements Datastore.Put
func (d *IpfsDatastore) Put(key ds.Key, value []byte) (err error) {
	path, err := d.api.Dag().Put(d.ctx, bytes.NewReader(value), coreapiopt.Dag.InputEnc("cbor"))
	if err != nil {
		return fmt.Errorf("error putting key %v err: %v", key.String(), err)
	}

	d.api.Pin().Add(d.ctx, path, coreapiopt.Pin.Recursive(false))

	return nil
}

// Get implements Datastore.Get
func (d *IpfsDatastore) Get(key ds.Key) (value []byte, err error) {
	cid, ok := parseKeyToCid(key)
	if !ok {
		return nil, fmt.Errorf("Key is not a valid CID: %v", key.String())
	}

	has, err := d.Has(key)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch CID %v err: %v", cid, err)
	}
	if !has {
		return nil, ds.ErrNotFound
	}

	node, err := d.api.Dag().Get(d.ctx, coreapiiface.IpldPath(*cid))

	if err != nil {
		return nil, fmt.Errorf("Could not fetch CID %v err: %v", cid, err)
	}

	return node.RawData(), nil
}

// Has implements Datastore.Has
func (d *IpfsDatastore) Has(key ds.Key) (exists bool, err error) {
	cid, ok := parseKeyToCid(key)
	if !ok {
		return false, fmt.Errorf("Key is not a valid CID: %v", key.String())
	}

	pins, err := d.api.Pin().Ls(d.ctx, coreapiopt.Pin.Type.Direct())

	if err != nil {
		return false, fmt.Errorf("Could not fetch keys: %v", err)
	}

	found := false

	for _, pin := range pins {
		if pin.Path().Cid().Equals(*cid) {
			found = true
			break
		}
	}

	return found, nil
}

// GetSize implements Datastore.GetSize
func (d *IpfsDatastore) GetSize(key ds.Key) (size int, err error) {
	nodeBytes, err := d.Get(key)

	if err != nil {
		return -1, err
	}

	return len(nodeBytes), nil
}

// Delete implements Datastore.Delete
func (d *IpfsDatastore) Delete(key ds.Key) (err error) {
	cid, ok := parseKeyToCid(key)
	if !ok {
		return fmt.Errorf("Key is not a valid CID: %v", key.String())
	}

	return d.Node.Pinning.Unpin(d.ctx, *cid, false)
}

// Query implements Datastore.Query
func (d *IpfsDatastore) Query(q dsq.Query) (dsq.Results, error) {
	pins, err := d.api.Pin().Ls(d.ctx, coreapiopt.Pin.Type.Direct())

	if err != nil {
		return nil, fmt.Errorf("Could not fetch keys: %v", err)
	}

	re := make([]dsq.Entry, len(pins))

	for i, pin := range pins {
		node, err := d.api.Dag().Get(d.ctx, pin.Path())
		if err != nil {
			return nil, fmt.Errorf("Could not fetch key %v: %v", pin.Path().Cid(), err)
		}

		re[i] = dsq.Entry{
			Key:   pin.Path().Cid().String(),
			Value: node.RawData(),
		}
	}

	r := dsq.ResultsWithEntries(q, re)
	r = dsq.NaiveQueryApply(q, r)
	return r, nil
}

func (d *IpfsDatastore) Batch() (ds.Batch, error) {
	return ds.NewBasicBatch(d), nil
}

func (d *IpfsDatastore) Close() error {
	return nil
}
