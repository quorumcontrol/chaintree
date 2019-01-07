package dag

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

// Dag is a convenience wrapper around a node store for setting and pruning
type Dag struct {
	Tip     cid.Cid
	oldTips []cid.Cid
	store   nodestore.NodeStore
}

// NewDag takes a tip and a store and returns an initialized Dag
func NewDag(tip cid.Cid, store nodestore.NodeStore) *Dag {
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

// WithNewTip returns a new Dag, but with the Tip set to the argument
func (d *Dag) WithNewTip(tip cid.Cid) *Dag {
	return &Dag{
		Tip:     tip,
		store:   d.store,
		oldTips: append(d.oldTips, d.Tip),
	}
}

// Get takes a CID and returns the cbornode
func (d *Dag) Get(id cid.Cid) (*cbornode.Node, error) {
	return d.store.GetNode(id)
}

// CreateNode adds an object to the Dags underlying storage (doesn't change the tip)
// and returns the cbornode
func (d *Dag) CreateNode(obj interface{}) (*cbornode.Node, error) {
	return d.store.CreateNode(obj)
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

// Update returns a new Dag with the old node at path swapped out for the new object
func (d *Dag) Update(path []string, newObj interface{}) (*Dag, error) {
	updatedNode, err := d.CreateNode(newObj)
	if err != nil {
		return nil, fmt.Errorf("error creating node: %v", err)
	}
	if len(path) == 0 {
		// We've updated all ancestors and have a new tip to set
		return d.WithNewTip(updatedNode.Cid()), nil
	} else {
		// We've got more path to recursively update; store this node & update its ref in its parent
		parentPath := path[:len(path)-1]
		parentObj, remaining, err := d.Resolve(parentPath)
		if err != nil {
			return nil, fmt.Errorf("error resolving parent node: %v", err)
		}
		if len(remaining) > 0 {
			return nil, fmt.Errorf("path elements remaining after resolving parent node: %v", remaining)
		}
		parentMap, ok := parentObj.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error asserting type map[string]interface{} of parent node: %v", parentObj)
		}
		parentMap[path[len(path)-1]] = updatedNode.Cid()
		return d.Update(parentPath, parentMap)
	}
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

func (d *Dag) getExisting(path []string) (interface{}, error) {
	existing, _, err := d.Resolve(path)
	if err != nil {
		return nil, err
	}

	switch existing.(type) {
	case map[string]interface{}:
		return existing.(map[string]interface{}), nil
	case nil:
		return nil, nil
	default:
		return make(map[string]interface{}), nil
	}
}

func (d *Dag) set(pathAndKey []string, val interface{}, asLink bool) (*Dag, error) {
	if !asLink {
		switch val.(type) {
		// These are the built in type of go (excluding map) plus cid.Cid
		// Use SetAsLink if attempting to set map
		case bool, byte, complex64, complex128, error, float32, float64, int, int8, int16, int32, int64, string, uint, uint16, uint32, uint64, uintptr, *cid.Cid, *bool, *byte, *complex64, *complex128, *error, *float32, *float64, *int, *int8, *int16, *int32, *int64, *string, *uint, *uint16, *uint32, *uint64, *uintptr, cid.Cid, []bool, []byte, []complex64, []complex128, []error, []float32, []float64, []int, []int8, []int16, []int32, []int64, []string, []uint, []uint16, []uint32, []uint64, []uintptr, []*cid.Cid, []*bool, []*byte, []*complex64, []*complex128, []*error, []*float32, []*float64, []*int, []*int8, []*int16, []*int32, []*int64, []*string, []*uint, []*uint16, []*uint32, []*uint64, []*uintptr, []cid.Cid:
			// Noop here, its a valid type, continue on
		default:
			return nil, fmt.Errorf("can not set complex objects, use asLink=true: %v", val)
		}
	}

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

	existing, err := d.getExisting(path)
	if err != nil {
		return nil, fmt.Errorf("error resolving")
	}
	if existing == nil {
		var newObj interface{}

		if asLink {
			path = append(path, key)
			newObj = val
		} else {
			newObj = map[string]interface{}{key: val}
		}

		sw := &safewrap.SafeWrap{}
		wrapped := sw.WrapObject(newObj)
		if sw.Err != nil {
			return nil, fmt.Errorf("error wrapping (%v): %v", newObj, sw.Err)
		}
		err = d.store.StoreNode(wrapped)
		if err != nil {
			return nil, fmt.Errorf("error storing node: %v", err)
		}
		return d.createDeepObject(path, wrapped)
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

	newDag, err := d.Update(path, existing)
	if err != nil {
		return nil, fmt.Errorf("error updating node: %v", err)
	}

	newDag.oldTips = append(d.oldTips, newDag.Tip)

	return newDag, nil
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
		last, err = d.store.CreateNode(map[string]cid.Cid{path[i]: last.Cid()})
		if err != nil {
			return nil, fmt.Errorf("error creating node: %v", err)
		}
	}

	setPath := path[0 : indexOfLastExistingNode+1]
	return d.set(setPath, last.Cid(), false)
}

func (d *Dag) dumpNode(node *cbornode.Node, isLink bool) interface{} {
	nodeData, _ := nodestore.CborNodeToObj(node)

	switch nodeData.(type) {
	case map[interface{}]interface{}:
		nodeMap := nodeData.(map[interface{}]interface{})
		for k, v := range nodeMap {
			switch v := v.(type) {
			case cid.Cid:
				node, _ := d.store.GetNode(v)
				if node == nil {
					nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
				} else {
					nodeMap[k] = d.dumpNode(node, true)
				}
			case *cid.Cid:
				node, _ := d.store.GetNode(*v)
				if node == nil {
					nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
				} else {
					nodeMap[k] = d.dumpNode(node, true)
				}
			default:
				continue
			}
		}
		if isLink {
			nodeMap["_isLink"] = true
		}
		return nodeMap
	default:
		return nodeData
	}
}

// Dump dumps the current DAG out as a string for debugging
func (d *Dag) Dump() string {
	rootNode, _ := d.store.GetNode(d.Tip)
	nodes, _ := d.Nodes()
	nodeStrings := make([]string, len(nodes))
	for i, node := range nodes {
		nodeStrings[i] = node.Cid().String()
	}
	return fmt.Sprintf(`
Tip: %s,
Tree:	
%s

Nodes: %v

	`,
		d.Tip.String(),
		spew.Sdump(d.dumpNode(rootNode, false)),
		nodes)
}
