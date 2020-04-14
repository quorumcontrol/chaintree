package graftabledag

import (
	"context"
	"fmt"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
)

type GraftableDag interface {
	GlobalResolve(ctx context.Context, path chaintree.Path) (value interface{}, remaining chaintree.Path, err error)
	OriginDag() *dag.Dag
	DagGetter() DagGetter
}

type DagGetter interface {
	GetTip(ctx context.Context, did string) (*cid.Cid, error)
	GetLatest(ctx context.Context, did string) (*chaintree.ChainTree, error)
}

type GraftedDag struct {
	dagCache  *lru.Cache // Is this premature optimization?
	dagGetter DagGetter
	origin    *dag.Dag
}

var _ GraftableDag = (*GraftedDag)(nil)

func New(origin *dag.Dag, dagGetter DagGetter) (*GraftedDag, error) {
	cache, err := lru.New(16)
	if err != nil {
		return nil, fmt.Errorf("could not create cache for GraftableDag: %w", err)
	}

	return &GraftedDag{
		dagCache:  cache,
		dagGetter: dagGetter,
		origin:    origin,
	}, nil
}

func (gd *GraftedDag) getChaintreeDag(ctx context.Context, did string) (*dag.Dag, error) {
	tip, err := gd.dagGetter.GetTip(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("could not get tip for %s: %w", did, err)
	}

	if uncastDag, ok := gd.dagCache.Get(tip); ok {
		if ctDag, ok := uncastDag.(*dag.Dag); ok {
			return ctDag, nil
		}
	}

	chainTree, err := gd.dagGetter.GetLatest(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("could not get latest for %s: %w", did, err)
	}

	gd.dagCache.Add(chainTree.Dag.Tip, chainTree.Dag)

	return chainTree.Dag, nil
}

// TODO: Does this need loop detection? It's OK to go through the same chaintree more than
//  once, just not the same chaintree+path.
func (gd *GraftedDag) resolveRecursively(ctx context.Context, path chaintree.Path, d *dag.Dag) (value interface{}, remaining chaintree.Path, err error) {
	value, remaining, err = d.Resolve(ctx, path)
	if err != nil {
		return value, remaining, err
	}

	didPaths := make([]chaintree.Path, 0)
	values := make([]interface{}, 0)

	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "did:tupelo:") {
			didPath := strings.Split(v, "/")
			didPaths = append(didPaths, didPath)
		} else {
			values = append(values, v)
		}
	case []interface{}:
		for _, val := range v {
			if sv, ok := val.(string); ok {
				if strings.HasPrefix(sv, "did:tupelo:") {
					didPath := strings.Split(sv, "/")
					didPaths = append(didPaths, didPath)
				} else {
					values = append(values, sv)
				}
			} else {
				values = append(values, val)
			}
		}
	default:
		values = append(values, v)
	}

	for _, didPath := range didPaths {
		did := didPath[0]

		var nextDag *dag.Dag
		nextDag, err = gd.getChaintreeDag(ctx, did)
		if err != nil {
			return value, remaining, err
		}

		nextPath := append(didPath[1:], remaining...)

		if len(nextPath) > 0 {
			value, remaining, err = gd.resolveRecursively(ctx, nextPath, nextDag)
			if err != nil {
				return value, remaining, err
			}
		} else {
			value = nextDag
		}
		values = append(values, value)
	}

	switch len(values) {
	case 1:
		value = values[0]
	case 0:
		value = nil
	default:
		value = values
	}

	return value, remaining, nil
}

// GlobalResolve works like dag.Resolve but will resolve across multiple chaintrees
// when it encounters string values that start with `did:tupelo:` (i.e. chaintree DIDs).
func (gd *GraftedDag) GlobalResolve(ctx context.Context, path chaintree.Path) (value interface{}, remaining chaintree.Path, err error) {
	return gd.resolveRecursively(ctx, path, gd.origin)
}

func (gd *GraftedDag) OriginDag() *dag.Dag {
	return gd.origin
}

func (gd *GraftedDag) DagGetter() DagGetter {
	return gd.dagGetter
}