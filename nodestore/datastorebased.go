package nodestore

import(
	"context"
	"github.com/ipfs/go-merkledag"
	blocks "github.com/ipfs/go-block-format"

	datastore "github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-blockservice"
	dsync "github.com/ipfs/go-datastore/sync"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
)

// Return a new DagStore which is only in memory
func MemoryStore(ctx context.Context) (DagStore,error) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()
	store := dsync.MutexWrap(datastore.NewMapDatastore())
	return FromDatastoreOffline(ctx, store)
}

func FromDatastoreOffline(ctx context.Context, ds datastore.Batching) (DagStore,error) {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)
	cachedbs, err := blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return nil, err
	}

	bserv := blockservice.New(cachedbs, &nullExchange{}) //only do offline for now.

	dags := merkledag.NewDAGService(bserv)
	return dags,nil
}

type nullExchange struct {
	exchange.Interface
}

func (ne *nullExchange) HasBlock(_ blocks.Block) error {
	return nil
}