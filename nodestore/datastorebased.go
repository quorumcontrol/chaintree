package nodestore

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-merkledag"
	"github.com/quorumcontrol/chaintree/cachedblockstore"

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

func FromDatastoreOfflineCached(ctx context.Context, ds datastore.Batching, cachesize int) (DagStore, error) {
	bs := blockstoreFromDatastore(ds, cachesize)
	return dagstoreFromBlockstore(bs), nil
}

func FromDatastoreOffline(ctx context.Context, ds datastore.Batching) (DagStore, error) {
	bs := blockstoreFromDatastore(ds, -1)
	return dagstoreFromBlockstore(bs), nil
}

func dagstoreFromBlockstore(bs blockstore.Blockstore) DagStore {
	// The reason this is writethrough is that the blockstore *also* does a check to see
	// if the blocks exist, this is an expensive operation on any non-local storage (like s3).
	// this `NewWriteThrough` is a convenient way to skip one of the checks
	bserv := blockservice.NewWriteThrough(bs, &nullExchange{})
	return merkledag.NewDAGService(bserv)
}

func blockstoreFromDatastore(ds datastore.Batching, cachesize int) blockstore.Blockstore {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)
	if cachesize > 0 {
		wrapped, err := cachedblockstore.WrapInCache(bs, cachesize)
		if err != nil {
			panic(err) // this only fails if for some reason the lru didn't initalize, which doesn't happen
		}
		return wrapped
	}
	return bs
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
