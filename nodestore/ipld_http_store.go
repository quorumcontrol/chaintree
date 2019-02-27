package nodestore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cid "github.com/ipfs/go-cid"
	ipfsHttpClient "github.com/ipfs/go-ipfs-http-client"
	cbornode "github.com/ipfs/go-ipld-cbor"
	ipldFormat "github.com/ipfs/go-ipld-format"
	ipfsCoreApiIface "github.com/ipfs/interface-go-ipfs-core"
	ipfsCoreApiOpt "github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipsn/go-ipfs/core"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/quorumcontrol/chaintree/safewrap"
)

type IpldHttpStoreConfig struct {
	Address ma.Multiaddr
}

// IpldHttpStore is a NodeStore that uses IPLD HTTP api
type IpldHttpStore struct {
	node *core.IpfsNode
	api  ipfsCoreApiIface.CoreAPI
}

var ErrHTTPNoSuchLink = errors.New("no such link found")

// var _ NodeStore = (*IpldHttpStore)(nil)

// NewIpldHttpStore creates a new NodeStore using the store argument for the backend
func NewIpldHttpStore(cfg *IpldHttpStoreConfig) *IpldHttpStore {
	api, _ := ipfsHttpClient.NewApi(cfg.Address)
	return &IpldHttpStore{
		api: api,
	}
}

func (ipld *IpldHttpStore) dag() ipfsCoreApiIface.APIDagService {
	return ipld.api.Dag()
}

func (ipld *IpldHttpStore) pin() ipfsCoreApiIface.PinAPI {
	return ipld.api.Pin()
}

func (ipld *IpldHttpStore) block() ipfsCoreApiIface.BlockAPI {
	return ipld.api.Block()
}

// CreateNode takes any object and converts it to a cbornode and then returns the saved CID
func (ipld *IpldHttpStore) CreateNode(obj interface{}) (node *cbornode.Node, err error) {
	node, err = objToCbor(obj)
	if err != nil {
		return nil, fmt.Errorf("error converting obj: %v", err)
	}
	return node, ipld.StoreNode(node)
}

// CreateNodeFromBytes implements the NodeStore interface
func (ipld *IpldHttpStore) CreateNodeFromBytes(data []byte) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.Decode(data)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return node, ipld.StoreNode(node)
}

// GetNode returns a cbornode for a CID
func (ipld *IpldHttpStore) GetNode(nodeCid cid.Cid) (node *cbornode.Node, err error) {
	ctx := context.Background()
	// castCid, _ := ipsnCid.Parse(nodeCid.String())

	// IPLDFIXME
	pins, err := ipld.pin().Ls(ctx, ipfsCoreApiOpt.Pin.Type.Direct())
	if err != nil {
		return nil, fmt.Errorf("error fetching pins: %v", err)
	}

	foundNode := false
	for _, p := range pins {
		if p.Path().Cid().Equals(nodeCid) {
			foundNode = true
			break
		}
	}

	if !foundNode {
		return nil, nil
	}

	// IPLDFIXMEA
	dagNode, err := ipld.dag().Get(ctx, nodeCid)

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
func (ipld *IpldHttpStore) DeleteNode(nodeCid cid.Cid) error {
	ctx := context.Background()
	path := ipfsCoreApiIface.IpldPath(nodeCid)

	// IPLDFIXME
	err := ipld.pin().Rm(ctx, path, ipfsCoreApiOpt.Pin.RmRecursive(false))

	if err != nil {
		return fmt.Errorf("error unpinning cid %s: %v", nodeCid.String(), err)
	}

	// IPLDFIXME
	err = ipld.block().Rm(ctx, ipfsCoreApiIface.IpldPath(nodeCid))
	if err != nil {
		return fmt.Errorf("error removing block cid %s: %v", nodeCid.String(), err)
	}

	return nil
}

// DeleteTree implements the NodeStore DeleteTree interface
func (ipld *IpldHttpStore) DeleteTree(tip cid.Cid) error {
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

func (ipld *IpldHttpStore) resolveNode(tip cid.Cid, path []string) (ipldFormat.Node, []string, error) {
	ctx := context.Background()
	// castCid, _ := ipsnCid.Parse(tip.String())
	// IPLDFIXME
	resolvedPath, err := ipld.api.ResolvePath(ctx, ipfsCoreApiIface.Join(ipfsCoreApiIface.IpldPath(tip), path...))

	if err != nil && err.Error() == ErrHTTPNoSuchLink.Error() && len(path) > 0 {
		parentPath := path[:len(path)-1]
		// IPLDFIXME
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
func (ipld *IpldHttpStore) Resolve(tip cid.Cid, path []string) (interface{}, []string, error) {
	dagNode, dagRemaining, err := ipld.resolveNode(tip, path)

	if err != nil {
		return nil, dagRemaining, nil
	}
	nodeValue, remaining, err := dagNode.Resolve(dagRemaining)

	if err != nil && err.Error() == ErrHTTPNoSuchLink.Error() {
		return nil, dagRemaining, nil
	}

	if err != nil {
		return nodeValue, remaining, fmt.Errorf("Could not resolve path %s for cid %s, err: %v", tip.String(), path, err)
	}

	return nodeValue, remaining, nil
}

// StoreNode implements the NodeStore interface
func (ipld *IpldHttpStore) StoreNode(node *cbornode.Node) error {
	nodeCid := node.Cid()
	path := ipfsCoreApiIface.IpldPath(nodeCid)
	ctx := context.Background()

	// IPLDFIXME
	err := ipld.dag().Add(ctx, node)
	if err != nil {
		return fmt.Errorf("error putting key %v err: %v", nodeCid.String(), err)
	}

	// IPLDFIXME
	err = ipld.pin().Add(ctx, path, ipfsCoreApiOpt.Pin.Recursive(false))
	if err != nil {
		return fmt.Errorf("error pinning key %v err: %v", nodeCid.String(), err)
	}

	return nil
}
