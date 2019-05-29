package nodestore

import(
	"context"
	datastore "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestFromDatastoreOffline(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()
	store := dsync.MutexWrap(datastore.NewMapDatastore())
	ds,err := FromDatastoreOffline(ctx, store)
	require.Nil(t,err)
	require.NotNil(t, ds)
}

func TestMemoryStore(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	mem,err := MemoryStore(ctx)
	require.Nil(t,err)
	require.NotNil(t, mem)
}