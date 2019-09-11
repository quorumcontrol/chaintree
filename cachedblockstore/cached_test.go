package cachedblockstore

import (
	"bytes"
	"testing"

	blocks "github.com/ipfs/go-block-format"

	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	u "github.com/ipfs/go-ipfs-util"
)

// These tests are just taken from https://github.com/ipfs/go-ipfs-blockstore/blob/master/blockstore_test.go

func TestGetWhenKeyNotPresent(t *testing.T) {
	underlying := blockstore.NewBlockstore(dsync.MutexWrap(ds.NewMapDatastore()))
	bs, err := WrapInCache(underlying, 100)
	if err != nil {
		t.Error(err)
	}
	c := cid.NewCidV0(u.Hash([]byte("stuff")))
	bl, err := bs.Get(c)

	if bl != nil {
		t.Error("nil block expected")
	}
	if err == nil {
		t.Error("error expected, got nil")
	}
}

func TestGetWhenKeyIsNil(t *testing.T) {
	underlying := blockstore.NewBlockstore(dsync.MutexWrap(ds.NewMapDatastore()))
	bs, err := WrapInCache(underlying, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = bs.Get(cid.Cid{})
	if err != blockstore.ErrNotFound {
		t.Fail()
	}
}

func TestPutThenGetBlock(t *testing.T) {
	underlying := blockstore.NewBlockstore(dsync.MutexWrap(ds.NewMapDatastore()))
	bs, err := WrapInCache(underlying, 100)
	if err != nil {
		t.Error(err)
	}
	block := blocks.NewBlock([]byte("some data"))

	err = bs.Put(block)
	if err != nil {
		t.Fatal(err)
	}

	blockFromBlockstore, err := bs.Get(block.Cid())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(block.RawData(), blockFromBlockstore.RawData()) {
		t.Fail()
	}
}
