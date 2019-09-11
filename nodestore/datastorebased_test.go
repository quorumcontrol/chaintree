package nodestore

import (
	"context"
	"testing"

	datastore "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/stretchr/testify/require"
)

func TestFromDatastoreOffline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := dsync.MutexWrap(datastore.NewMapDatastore())
	ds, err := FromDatastoreOffline(ctx, store)
	require.Nil(t, err)
	require.NotNil(t, ds)
}

func TestFromDatastoreOfflineCached(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := dsync.MutexWrap(datastore.NewMapDatastore())
	ds, err := FromDatastoreOfflineCached(ctx, store, 100)
	require.Nil(t, err)
	require.NotNil(t, ds)
}

func TestMemoryStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem, err := MemoryStore(ctx)
	require.Nil(t, err)
	require.NotNil(t, mem)

	sw := safewrap.SafeWrap{}
	obj := map[string]string{"test": "test"}
	n := sw.WrapObject(obj)
	require.Nil(t, sw.Err)

	err = mem.Add(ctx, n)
	require.Nil(t, err)

	returnedNode, err := mem.Get(ctx, n.Cid())
	require.Nil(t, err)
	require.NotNil(t, returnedNode)

	// works with a missing node

	obj = map[string]string{"test": "diff"}
	n = sw.WrapObject(obj)
	require.Nil(t, sw.Err)

	_, err = mem.Get(ctx, n.Cid())
	require.Equal(t, format.ErrNotFound, err)
}
