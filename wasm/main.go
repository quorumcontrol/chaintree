package main

import (
	"context"

	"github.com/quorumcontrol/chaintree/chaintree"
	_ "github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	_ "github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"

	_ "github.com/quorumcontrol/chaintree/safewrap"
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

	tree := sw.WrapObject(map[string]string{
		"hithere": "hothere",
	})

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
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
	graph, err := dag.NewDagWithNodes(ctx, store, root, tree, chain)
	if err != nil {
		panic(err)
	}
	_, err = chaintree.NewChainTree(
		ctx,
		graph,
		nil,
		DefaultTransactors,
	)
	if err != nil {
		panic(err)
	}

}
