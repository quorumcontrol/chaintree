package cachedblockstore

import (
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"golang.org/x/xerrors"
)

type CachedBlockstore struct {
	blockstore.Blockstore
	cache *lru.Cache
}

func WrapInCache(bs blockstore.Blockstore) (*CachedBlockstore, error) {
	cache, err := lru.New(10000)
	if err != nil {
		return nil, xerrors.Errorf("error creating cache: %w", err)
	}
	return &CachedBlockstore{
		Blockstore: bs,
		cache:      cache,
	}, nil
}

func (cbs *CachedBlockstore) DeleteBlock(id cid.Cid) error {
	cbs.cache.Remove(id)
	return cbs.Blockstore.DeleteBlock(id)
}

func (cbs *CachedBlockstore) Get(id cid.Cid) (blocks.Block, error) {
	blckInter, ok := cbs.cache.Get(id)
	if ok {
		return blckInter.(blocks.Block), nil
	}
	return cbs.Blockstore.Get(id)
}

func (cbs *CachedBlockstore) Has(id cid.Cid) (bool, error) {
	// This is a choice here to return false if the cache hasn't seen it
	// and skip doing lookups on the underlying blockstore
	// right now Has is really only used for "should I put this?" and so
	// we opt for more puts, but less lookups
	return cbs.cache.Contains(id), nil
}

func (cbs *CachedBlockstore) Put(block blocks.Block) error {
	cbs.cache.Add(block.Cid(), block)
	return cbs.Blockstore.Put(block)
}

func (cbs *CachedBlockstore) PutMany(blocks []blocks.Block) error {
	for _, block := range blocks {
		cbs.cache.Add(block.Cid(), block)
	}
	return cbs.Blockstore.PutMany(blocks)
}
