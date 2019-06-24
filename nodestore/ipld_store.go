package nodestore

import (
	"context"
	"fmt"
	"strings"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	ipldFormat "github.com/ipfs/go-ipld-format"
	coreApiIface "github.com/ipfs/interface-go-ipfs-core"
	ipldpath "github.com/ipfs/interface-go-ipfs-core/path"
	coreApiOpt "github.com/ipfs/interface-go-ipfs-core/options"
	multihash "github.com/multiformats/go-multihash"
	"github.com/quorumcontrol/chaintree/safewrap"
)

// IpldStore is a NodeStore that uses IPLD
type IpldStore struct {
	api coreApiIface.CoreAPI
}

var errNoSuchLink = cbornode.ErrNoSuchLink

var _ NodeStore = (*IpldStore)(nil)

// NewIpldStore creates a new NodeStore using an IPLD api
func NewIpldStore(api coreApiIface.CoreAPI) *IpldStore {
	return &IpldStore{
		api: api,
	}
}

func (ipld *IpldStore) dag() coreApiIface.APIDagService {
	return ipld.api.Dag()
}

func (ipld *IpldStore) pin() coreApiIface.PinAPI {
	return ipld.api.Pin()
}

func (ipld *IpldStore) block() coreApiIface.BlockAPI {
	return ipld.api.Block()
}

// CreateNode takes any object and converts it to a cbornode and then returns the saved CID
func (ipld *IpldStore) CreateNode(obj interface{}) (node *cbornode.Node, err error) {
	node, err = objToCbor(obj)
	if err != nil {
		return nil, fmt.Errorf("error converting obj: %v", err)
	}
	return node, ipld.StoreNode(node)
}

// CreateNodeFromBytes implements the NodeStore interface
func (ipld *IpldStore) CreateNodeFromBytes(data []byte) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.Decode(data)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return node, ipld.StoreNode(node)
}

// GetNode returns a cbornode for a CID
func (ipld *IpldStore) GetNode(nodeCid cid.Cid) (node *cbornode.Node, err error) {
	ctx := context.Background()
	castCid, _ := cid.Parse(nodeCid.String())

	pins, err := ipld.pin().Ls(ctx, coreApiOpt.Pin.Type.Direct())
	if err != nil {
		return nil, fmt.Errorf("error fetching pins: %v", err)
	}

	foundNode := false
	for _, p := range pins {
		if p.Path().Cid().Equals(castCid) {
			foundNode = true
			break
		}
	}

	if !foundNode {
		return nil, nil
	}

	dagNode, err := ipld.dag().Get(ctx, castCid)

	if err == ipldFormat.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error getting cid: %v", err)
	}
	nodeBytes := dagNode.RawData()
	if len(nodeBytes) == 0 {
		return nil, nil
	}
	sw := &safewrap.SafeWrap{}
	node = sw.Decode(nodeBytes)
	return node, sw.Err
}

// DeleteNode implements the NodeStore DeleteNode interface.
func (ipld *IpldStore) DeleteNode(nodeCid cid.Cid) error {
	ctx := context.Background()
	castCid, _ := cid.Parse(nodeCid.String())
	path := ipldpath.IpldPath(castCid)

	err := ipld.pin().Rm(ctx, path, coreApiOpt.Pin.RmRecursive(false))

	if err != nil {
		return fmt.Errorf("error unpinning cid %s: %v", nodeCid.String(), err)
	}

	err = ipld.block().Rm(ctx, ipldpath.IpldPath(castCid))
	if err != nil {
		return fmt.Errorf("error removing block cid %s: %v", nodeCid.String(), err)
	}

	return nil
}

// DeleteTree implements the NodeStore DeleteTree interface
func (ipld *IpldStore) DeleteTree(tip cid.Cid) error {
	tipNode, err := ipld.GetNode(tip)
	if err != nil {
		return fmt.Errorf("error getting tip: %v", err)
	}

	links := tipNode.Links()

	for _, link := range links {
		err := ipld.DeleteTree(link.Cid)
		if err != nil {
			return fmt.Errorf("error deleting: %v", err)
		}
	}
	return ipld.DeleteNode(tip)
}

func (ipld *IpldStore) resolveNode(tip cid.Cid, path []string) (ipldFormat.Node, []string, error) {
	ctx := context.Background()
	castCid, _ := cid.Parse(tip.String())
	resolvedPath, err := ipld.api.ResolvePath(ctx, ipldpath.Join(ipldpath.IpldPath(castCid), path...))

	if err != nil && err.Error() == errNoSuchLink.Error() && len(path) > 0 {
		parentPath := path[:len(path)-1]
		parentNode, parentRemainder, parentErr := ipld.resolveNode(tip, parentPath)
		return parentNode, append(parentRemainder, path[len(parentPath):]...), parentErr
	}

	if err != nil {
		return nil, nil, err
	}

	remaining := []string{}

	if resolvedPath.Remainder() != "" {
		remaining = strings.Split(resolvedPath.Remainder(), "/")
	}

	dagNode, err := ipld.api.ResolveNode(ctx, resolvedPath)
	if err != nil {
		return nil, nil, err
	}

	return dagNode, remaining, nil
}

// Resolve implements the NodeStore interface
func (ipld *IpldStore) Resolve(tip cid.Cid, path []string) (interface{}, []string, error) {
	dagNode, dagRemaining, err := ipld.resolveNode(tip, path)

	if err != nil {
		return nil, dagRemaining, nil
	}
	nodeValue, remaining, err := dagNode.Resolve(dagRemaining)

	if err != nil && err.Error() == errNoSuchLink.Error() {
		return nil, dagRemaining, nil
	}

	if err != nil {
		return nodeValue, remaining, fmt.Errorf("Could not resolve path %s for cid %s, err: %v", tip.String(), path, err)
	}

	return nodeValue, remaining, nil
}

// StoreNode implements the NodeStore interface
func (ipld *IpldStore) StoreNode(node *cbornode.Node) error {
	nodeCid := node.Cid()
	castCid, _ := cid.Parse(nodeCid.String())
	path := ipldpath.IpldPath(castCid)
	ctx := context.Background()

	ipsnNode, err := cbornode.Decode(node.RawData(), multihash.SHA2_256, -1)
	if err != nil {
		return fmt.Errorf("error decoding %v err: %v", nodeCid.String(), err)
	}

	err = ipld.dag().Add(ctx, ipsnNode)
	if err != nil {
		return fmt.Errorf("error putting key %v err: %v", nodeCid.String(), err)
	}

	err = ipld.pin().Add(ctx, path, coreApiOpt.Pin.Recursive(false))
	if err != nil {
		return fmt.Errorf("error pinning key %v err: %v", nodeCid.String(), err)
	}

	return nil
}
