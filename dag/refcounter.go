package dag

import (
	"context"

	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/nodestore"
)

type cidTracker map[cid.Cid]struct{}

type storeWrapper struct {
	nodestore.DagStore
	touched cidTracker
}

func (sw *storeWrapper) Get(ctx context.Context, id cid.Cid) (format.Node, error) {
	sw.touched[id] = struct{}{}
	return sw.DagStore.Get(ctx, id)
}

func wrapStoreForRefCounting(store nodestore.DagStore) *storeWrapper {
	return &storeWrapper{
		DagStore: store,
		touched:  make(cidTracker),
	}
}

// type RefTrackingDag struct {
// 	*Dag
// 	Touched       cidTracker
// 	originalStore nodestore.DagStore
// }

// func RefCountDag(graph *Dag) *RefTrackingDag {
// 	rtd := &RefTrackingDag{
// 		Touched:       make(cidTracker),
// 		originalStore: graph.store,
// 	}
// 	newDag := &Dag{
// 		Tip:   graph.Tip,
// 		store: &storeWrapper{DagStore: graph.store, refTracker: rtd},
// 	}
// 	rtd.Dag = newDag
// 	return rtd
// }

// func (rtd *RefTrackingDag) Unwrap() *Dag {
// 	return &Dag{
// 		Tip:   rtd.Dag.Tip,
// 		store: rtd.originalStore,
// 	}
// }
