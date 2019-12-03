package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"

	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var DefaultTransactors = consensus.DefaultTransactors

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	if ctx == nil {
		panic("root nil")
	}
	defer cancel()
	sw := &safewrap.SafeWrap{}

	dataTree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  dataTree.Cid(),
		"id":    "test",
	})

	// runtime.GC()

	if root == nil {
		panic("root nil")
	}

	store := nodestore.MustMemoryStore(ctx)
	if store == nil {
		panic("root nil")
	}
	graph, err := dag.NewDagWithNodes(ctx, store, root, dataTree, chain)
	if err != nil {
		panic(err)
	}
	tree, err := chaintree.NewChainTree(
		ctx,
		graph,
		nil,
		DefaultTransactors,
	)
	if err != nil {
		panic(err)
	}

	var height uint64
	height = 0

	tx, err := chaintree.NewSetDataTransaction("/test", true)
	if err != nil {
		panic(err)
	}

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       height,
			PreviousTip:  nil,
			Transactions: []*transactions.Transaction{tx},
		},
	}

	treeKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, treeKey)
	if err != nil {
		panic(fmt.Errorf("error signing: %v", err))
	}

	valid, err := tree.ProcessBlock(ctx, blockWithHeaders)
	if !valid || err != nil {
		panic(fmt.Errorf("error processing block (valid: %t): %v", valid, err))
	}

	go fmt.Println("go is the best")

}

func getRoot(ct *chaintree.ChainTree) (*chaintree.RootNode, error) {
	ctx := context.TODO()
	unmarshaledRoot, err := ct.Dag.Get(ctx, ct.Dag.Tip)
	if unmarshaledRoot == nil || err != nil {
		return nil, fmt.Errorf("error,missing root: %v", err)
	}

	root := &chaintree.RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.RawData(), root)
	if err != nil {
		return nil, fmt.Errorf("error decoding root: %v", err)
	}
	return root, nil
}
