package dag

import (
	"fmt"

	"github.com/ipfs/go-ipld-cbor"

	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

// Dag is a convenience wrapper around a node store for setting and pruning
type Dag struct {
	Tip     *cid.Cid
	oldTips []*cid.Cid
	store   nodestore.NodeStore
}

// NewDag takes a tip and a store and returns an initialized Dag
func NewDag(tip *cid.Cid, store nodestore.NodeStore) *Dag {
	return &Dag{
		Tip:   tip,
		store: store,
	}
}

// NewDagWithNodes creates a new Dag, and imports the passed in nodes, the first node is set as the tip
func NewDagWithNodes(store nodestore.NodeStore, nodes ...*cbornode.Node) (*Dag, error) {
	dag := &Dag{
		Tip:   nodes[0].Cid(),
		store: store,
	}
	err := dag.AddNodes(nodes...)
	if err != nil {
		return nil, fmt.Errorf("error adding nodes: %v", err)
	}
	return dag, nil
}

// AddNodes takes cbornodes and adds them to the underlying storage
func (d *Dag) AddNodes(nodes ...*cbornode.Node) error {
	for _, node := range nodes {
		err := d.store.StoreNode(node)
		if err != nil {
			return fmt.Errorf("error storing node (%s): %v", node.Cid().String(), err)
		}
	}
	return nil
}

// Resolve takes a path (as a string slice) and returns the value, remaining path and any error
// it delegates to the underlying store's resolve
func (d *Dag) Resolve(path []string) (interface{}, []string, error) {
	return d.store.Resolve(d.Tip, path)
}

// Nodes returns all the nodes in an entire tree from the Tip out
func (d *Dag) Nodes() ([]*cbornode.Node, error) {
	root, err := d.store.GetNode(d.Tip)
	if err != nil {
		return nil, fmt.Errorf("error getting root: %v", err)
	}
	return d.nodeAndDecendants(root)
}

func (d *Dag) nodeAndDecendants(node *cbornode.Node) ([]*cbornode.Node, error) {
	links := node.Links()
	nodes := []*cbornode.Node{node}
	for _, link := range links {
		linkNode, err := d.store.GetNode(link.Cid)
		if err != nil {
			return nil, fmt.Errorf("error getting link: %v", err)
		}
		childNodes, err := d.nodeAndDecendants(linkNode)
		if err != nil {
			return nil, fmt.Errorf("error getting child nodes: %v", err)
		}
		nodes = append(nodes, childNodes...)
	}
	return nodes, nil
}

// Set sets a value at a path and returns a new dag with a new tip that reflects
// the new state (and adds the old tip to oldTips)
func (d *Dag) Set(pathAndKey []string, val interface{}) (*Dag, error) {
	return d.set(pathAndKey, val, false)
}

// SetAsLink sets a value at a path and returns a new dag with a new tip that reflects
// the new state (and adds the old tip to oldTips)
func (d *Dag) SetAsLink(pathAndKey []string, val interface{}) (*Dag, error) {
	return d.set(pathAndKey, val, true)
}

func (d *Dag) set(pathAndKey []string, val interface{}, asLink bool) (*Dag, error) {
	var path []string
	var key string

	switch len(pathAndKey) {
	case 0:
		return nil, fmt.Errorf("must pass in a key")
	case 1:
		path = []string{}
		key = pathAndKey[0]
	default:
		path = pathAndKey[0 : len(pathAndKey)-1]
		key = pathAndKey[len(pathAndKey)-1]
	}

	existing, _, err := d.Resolve(path)
	if err != nil {
		return nil, fmt.Errorf("error resolving")
	}
	if existing == nil {
		newObj := map[string]interface{}{key: val}
		sw := &safewrap.SafeWrap{}
		wrapped := sw.WrapObject(newObj)
		if sw.Err != nil {
			return nil, err
		}
		err = d.store.StoreNode(wrapped)
		if err != nil {
			return nil, fmt.Errorf("error storing node: %v", err)
		}
		return d.createDeepObject(path, wrapped)
	}

	sw := &safewrap.SafeWrap{}
	existingCbor := sw.WrapObject(existing)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping (%v): %v", existing, err)
	}

	if asLink {
		newNode, err := d.store.CreateNode(val)
		if err != nil {
			return nil, fmt.Errorf("error creating node: %v", err)
		}
		existing.(map[string]interface{})[key] = newNode.Cid()
	} else {
		existing.(map[string]interface{})[key] = val
	}

	_, updates, err := d.store.UpdateNode(existingCbor.Cid(), existing)
	if err != nil {
		return nil, fmt.Errorf("error updagint node: %v", err)
	}
	newTip := updates[nodestore.ToCidString(d.Tip)]

	return &Dag{
		store:   d.store,
		Tip:     newTip,
		oldTips: append(d.oldTips, d.Tip),
	}, nil
}

func (d *Dag) createDeepObject(path []string, node *cbornode.Node) (*Dag, error) {

	var indexOfLastExistingNode int

	for i := len(path); i >= 0; i-- {
		val, _, err := d.Resolve(path[0:i])
		if err != nil {
			return nil, fmt.Errorf("error resolving: %v", err)
		}
		if val != nil {
			indexOfLastExistingNode = i
			break
		}
	}

	var last = node
	var err error

	for i := len(path) - 1; i > indexOfLastExistingNode; i-- {
		last, err = d.store.CreateNode(map[string]*cid.Cid{path[i]: last.Cid()})
		if err != nil {
			return nil, fmt.Errorf("error creating node: %v", err)
		}
	}

	setPath := path[0 : indexOfLastExistingNode+1]
	return d.set(setPath, last.Cid(), false)
}
