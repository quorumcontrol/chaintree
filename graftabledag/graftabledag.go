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

// PathsContainPrefix is used for loop detection (these are DAGs after all).
// If any element of haystack has needle as a prefix, or if any haystack item is a
// prefix of needle, then this returns true, otherwise false.
func PathsContainPrefix(haystack []chaintree.Path, needle chaintree.Path) bool {
	for _, p := range haystack {
		var prefixLen int
		if len(p) < len(needle) {
			prefixLen = len(p)
		} else {
			prefixLen = len(needle)
		}

		for i := 0; i < prefixLen; i++ {
			if needle[i] != p[i] {
				break
			}

			if i == (prefixLen - 1) {
				return true
			}
		}
	}
	return false
}

func (gd *GraftedDag) resolveGraftedVal(ctx context.Context, didPath chaintree.Path, parentRemaining chaintree.Path, seen []chaintree.Path) (value interface{}, remaining chaintree.Path, err error) {
	did := didPath[0]

	var nextDag *dag.Dag
	nextDag, err = gd.getChaintreeDag(ctx, did)
	if err != nil {
		return value, remaining, err
	}

	nextPath := append(didPath[1:], parentRemaining...)

	if len(nextPath) > 0 {
		return gd.resolveRecursively(ctx, nextPath, nextDag, seen)
	}

	return nextDag, remaining, err
}

func (gd *GraftedDag) resolveRecursively(ctx context.Context, path chaintree.Path, d *dag.Dag, seen []chaintree.Path) (value interface{}, remaining chaintree.Path, err error) {
	value, remaining, err = d.Resolve(ctx, path)
	if err != nil {
		return value, remaining, err
	}

	nextSeen := seen

	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "did:tupelo:") {
			didPath := strings.Split(v, "/")
			if PathsContainPrefix(seen, didPath) {
				return nil, nil, fmt.Errorf("loop detected; some or all of %v was already visited in this resolution", strings.Join(didPath, "/"))
			}
			nextSeen = append(nextSeen, didPath)
			value, remaining, err = gd.resolveGraftedVal(ctx, didPath, remaining, nextSeen)
			if err != nil || len(remaining) > 0 {
				return value, remaining, err
			}
		} else {
			value = v
		}
	case []interface{}:
		values := make([]interface{}, len(v))
		for i, val := range v {
			if sv, ok := val.(string); ok {
				if strings.HasPrefix(sv, "did:tupelo:") {
					didPath := strings.Split(sv, "/")
					if PathsContainPrefix(seen, didPath) {
						return nil, nil, fmt.Errorf("loop detected; some or all of %v was already visited in this resolution", strings.Join(didPath, "/"))
					}
					nextSeen = append(nextSeen, didPath)
					graftedVal, remaining, err := gd.resolveGraftedVal(ctx, didPath, remaining, nextSeen)
					if err != nil || len(remaining) > 0 {
						return value, remaining, err
					}
					values[i] = graftedVal
				} else {
					values[i] = sv
				}
			} else {
				values[i] = val
			}
		}
		value = values
	default:
		value = v
	}

	return value, remaining, nil
}

// GlobalResolve works like dag.Resolve but will resolve across multiple chaintrees
// when it encounters string values that start with `did:tupelo:` (i.e. chaintree DIDs).
func (gd *GraftedDag) GlobalResolve(ctx context.Context, path chaintree.Path) (value interface{}, remaining chaintree.Path, err error) {
	seen := make([]chaintree.Path, 0)
	return gd.resolveRecursively(ctx, path, gd.origin, seen)
}

func (gd *GraftedDag) OriginDag() *dag.Dag {
	return gd.origin
}

func (gd *GraftedDag) DagGetter() DagGetter {
	return gd.dagGetter
}
