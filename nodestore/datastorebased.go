package nodestore

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-merkledag"

	"github.com/ipfs/go-blockservice"
	datastore "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
)

// Return a new DagStore which is only in memory
func MemoryStore(ctx context.Context) (DagStore, error) {
	store := dsync.MutexWrap(datastore.NewMapDatastore())
	return FromDatastoreOffline(ctx, store)
}

// Return a new DagStore which is only in memory
func MustMemoryStore(ctx context.Context) DagStore {
	ds, err := MemoryStore(ctx)
	if err != nil {
		panic(err)
	}
	return ds
}

func FromDatastoreOffline(ctx context.Context, ds datastore.Batching) (DagStore, error) {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)
	// The reason this is writethrough is that the blockstore *also* does a check to see
	// if the blocks exist, this is an expensive operation on any non-local storage (like s3).
	// this `NewWriteThrough` is a convenient way to skip one of the checks
	bserv := blockservice.NewWriteThrough(bs, &nullExchange{})
	dags := merkledag.NewDAGService(bserv)
	return dags, nil
}

type nullExchange struct {
	exchange.Interface
}

func (ne *nullExchange) HasBlock(_ blocks.Block) error {
	return nil
}

func (ne *nullExchange) IsOnline() bool {
	return false
}

func (ne *nullExchange) GetBlock(context.Context, cid.Cid) (blocks.Block, error) {
	return nil, blockstore.ErrNotFound
}
