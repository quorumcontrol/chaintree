package dag

import (
	"context"
	"fmt"

	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"

	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

// Dag is a convenience wrapper around a node store for setting and pruning
type Dag struct {
	Tip     cid.Cid
	oldTips []cid.Cid
	store   nodestore.DagStore
}

type NodeMap map[cid.Cid]format.Node

// NewDag takes a tip and a store and returns an initialized Dag
func NewDag(_ context.Context, tip cid.Cid, store nodestore.DagStore) *Dag {
	return &Dag{
		Tip:   tip,
		store: store,
	}
}

// NewDagWithNodes creates a new Dag, and imports the passed in nodes, the first node is set as the tip
func NewDagWithNodes(ctx context.Context, store nodestore.DagStore, nodes ...format.Node) (*Dag, error) {
	dag := &Dag{
		Tip:   nodes[0].Cid(),
		store: store,
	}
	err := dag.AddNodes(ctx, nodes...)
	if err != nil {
		return nil, fmt.Errorf("error adding nodes: %v", err)
	}
	return dag, nil
}

// AddNodes takes cbornodes and adds them to the underlying storage
func (d *Dag) AddNodes(ctx context.Context, nodes ...format.Node) error {
	for _, node := range nodes {
		err := d.store.Add(ctx, node)
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
func (d *Dag) Get(ctx context.Context, id cid.Cid) (format.Node, error) {
	n, err := d.store.Get(ctx, id)
	if err == format.ErrNotFound {
		return nil, nil
	}
	return n, err
}

// CreateNode adds an object to the Dags underlying storage (doesn't change the tip)
// and returns the cbornode
func (d *Dag) CreateNode(ctx context.Context, obj interface{}) (format.Node, error) {
	sw := &safewrap.SafeWrap{}
	n := sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping object: %v", sw.Err)
	}
	return n, d.store.Add(ctx, n)
}

func (d *Dag) ResolveInto(ctx context.Context, path []string, obj interface{}) error {
	var initialPath []string
	var lastKey string
	switch {
	case len(path) == 0:
		// do nothing, the nils above are ok
	case len(path) == 1:
		lastKey = path[0]
	default:
		initialPath = path[0 : len(path)-1]
		lastKey = path[len(path)-1]
	}
	if lastKey == "" {
		n, err := d.store.Get(ctx, d.Tip)
		if err != nil {
			return fmt.Errorf("error getting tip: %v", err)
		}
		return cbornode.DecodeInto(n.RawData(), obj)
	}
	// otherwise we get a map and use the last key
	val, remain, err := d.Resolve(ctx, initialPath)
	if err != nil {
		return fmt.Errorf("error getting initialPath: %v", err)
	}
	if len(remain) > 0 {
		return format.ErrNotFound
	}
	mapped, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("error the path you specify must resolve to an object with links")
	}
	cidInter, ok := mapped[lastKey]
	if !ok {
		return format.ErrNotFound
	}
	id, ok := cidInter.(cid.Cid)
	if !ok {
		return fmt.Errorf("error the path did not resolve to a link")
	}
	n, err := d.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting cid: %v", err)
	}
	return cbornode.DecodeInto(n.RawData(), obj)
}

// Resolve takes a path (as a string slice) and returns the value, remaining path and any error.
func (d *Dag) Resolve(ctx context.Context, path []string) (interface{}, []string, error) {
	return d.ResolveAt(ctx, d.Tip, path)
}

// ResolveAt takes a tip and a path (as a string slice) and returns the value, remaining path
// and any error.
func (d *Dag) ResolveAt(ctx context.Context, tip cid.Cid, path []string) (val interface{}, remaining []string, err error) {
	node, err := d.store.Get(ctx, tip)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting node (%s): %v", tip.String(), err)
	}
	val, remaining, err = node.Resolve(path)
	if err != nil {
		switch err {
		case cbornode.ErrNoSuchLink:
			// If the link is just missing, then just return the whole path as remaining, with a nil value
			// instead of an error
			return nil, path, nil
		case cbornode.ErrNoLinks:
			// this means there was a simple value somewhere along the path
			// try resolving less of the path to find the existing boundary
			var err error
			for i := 1; i < len(path); i++ {
				val, _, err = node.Resolve(path[:len(path)-i])
				if err != nil {
					continue
				} else {
					// return the simple value and the rest of the path as remaining
					return val, path[len(path)-i:], nil
				}
			}
			if err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, err
	}

	switch val := val.(type) {
	case *format.Link:
		linkNode, err := d.store.Get(ctx, val.Cid)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting linked node (%s): %v", linkNode.Cid().String(), err)
		}
		if linkNode != nil {
			return d.ResolveAt(ctx, linkNode.Cid(), remaining)
		}
		return nil, remaining, nil
	default:
		return val, remaining, err
	}
}

func (d *Dag) NodesForPathWithDecendants(ctx context.Context, path []string) ([]format.Node, error) {
	nodes, err := d.orderedNodesForPath(ctx, path)
	if err != nil {
		return nil, err
	}
	lastNode := nodes[len(nodes)-1]

	collector := NodeMap{}
	for _, n := range nodes {
		collector[n.Cid()] = n
	}

	err = d.nodeAndDescendants(ctx, lastNode, collector)
	if err != nil {
		return nil, err
	}

	nodes = make([]format.Node, len(collector))
	i := 0
	for _, v := range collector {
		nodes[i] = v
		i++
	}
	return nodes, nil
}

func (d *Dag) NodesForPath(ctx context.Context, path []string) ([]format.Node, error) {
	return d.orderedNodesForPath(ctx, path)
}

func (d *Dag) orderedNodesForPath(ctx context.Context, path []string) ([]format.Node, error) {
	nodes := make([]format.Node, len(path)+1) // + 1 for tip node

	tipNode, err := d.Get(ctx, d.Tip)
	if err != nil {
		return nil, err
	}

	nodes[0] = tipNode
	cur := tipNode

	for i, val := range path {
		nextNode, remaining, err := cur.ResolveLink([]string{val})
		if err != nil {
			return nil, err
		}

		if len(remaining) > 0 {
			return nil, fmt.Errorf("error: unexpected remaining path elements: %v", remaining)
		}

		cur, err = d.Get(ctx, nextNode.Cid)
		if err != nil {
			return nil, err
		}

		nodes[i+1] = cur
	}

	return nodes, nil
}

// Nodes returns all the nodes in an entire tree from the Tip out
func (d *Dag) Nodes(ctx context.Context) ([]format.Node, error) {
	root, err := d.store.Get(ctx, d.Tip)
	if err != nil {
		return nil, fmt.Errorf("error getting root: %v", err)
	}
	collector := NodeMap{}

	err = d.nodeAndDescendants(ctx, root, collector)
	if err != nil {
		return nil, fmt.Errorf("error getting dec: %v", err)
	}

	nodes := make([]format.Node, len(collector))
	i := 0
	for _, v := range collector {
		nodes[i] = v
		i++
	}
	return nodes, nil
}

func (d *Dag) nodeAndDescendants(ctx context.Context, node format.Node, collector NodeMap) error {
	collector[node.Cid()] = node

	links := node.Links()
	for _, link := range links {
		_, ok := collector[link.Cid]
		if ok {
			continue
		}
		linkNode, err := d.store.Get(ctx, link.Cid)
		if err != nil && err != format.ErrNotFound {
			return fmt.Errorf("error getting link: %v", err)
		}
		if linkNode == nil {
			// it's OK to omit certain parts of the tree in e.g. send_token payloads
			continue
		}
		err = d.nodeAndDescendants(ctx, linkNode, collector)
		if err != nil {
			return fmt.Errorf("error getting child nodes: %v", err)
		}
	}

	return nil
}

// Delete removes a key from the dag
func (d *Dag) Delete(ctx context.Context, path []string) (*Dag, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("Can not execute Delete on root of dag, please supply non-empty path")
	}

	parentPath := path[:len(path)-1]
	parentObj, remaining, err := d.Resolve(ctx, parentPath)

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

	keyToDelete := path[len(path)-1]

	if _, ok := parentMap[keyToDelete]; !ok {
		return nil, fmt.Errorf("key %v does not exist at path %v", keyToDelete, parentPath)
	}

	delete(parentMap, keyToDelete)

	return d.Update(ctx, parentPath, parentObj)
}

// Update returns a new Dag with the old node at path swapped out for the new object
func (d *Dag) Update(ctx context.Context, path []string, newObj interface{}) (*Dag, error) {
	updatedNode, err := d.CreateNode(ctx, newObj)
	if err != nil {
		return nil, fmt.Errorf("error creating node: %v", err)
	}
	if len(path) == 0 {
		// We've updated all ancestors and have a new tip to set
		return d.WithNewTip(updatedNode.Cid()), nil
	} else {
		// We've got more path to recursively update; store this node & update its ref in its parent
		parentPath := path[:len(path)-1]
		parentObj, remaining, err := d.Resolve(ctx, parentPath)
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
		return d.Update(ctx, parentPath, parentMap)
	}
}

func isComplexObj(val interface{}) bool {
	switch val.(type) {
	// These are the built in type of go (excluding map) plus cid.Cid
	// Use SetAsLink if attempting to set map
	case bool, byte, complex64, complex128, error, float32, float64, int, int8, int16, int32, int64, string, uint, uint16, uint32, uint64, uintptr, *cid.Cid, *bool, *byte, *complex64, *complex128, *error, *float32, *float64, *int, *int8, *int16, *int32, *int64, *string, *uint, *uint16, *uint32, *uint64, *uintptr, cid.Cid, []bool, []byte, []complex64, []complex128, []error, []float32, []float64, []int, []int8, []int16, []int32, []int64, []string, []uint, []uint16, []uint32, []uint64, []uintptr, []*cid.Cid, []*bool, []*byte, []*complex64, []*complex128, []*error, []*float32, []*float64, []*int, []*int8, []*int16, []*int32, []*int64, []*string, []*uint, []*uint16, []*uint32, []*uint64, []*uintptr, []cid.Cid:
		return false
	default:
		return true
	}
}

// Set sets a value at a path and returns a new dag with a new tip that reflects
// the new state (and adds the old tip to oldTips)
func (d *Dag) Set(ctx context.Context, pathAndKey []string, val interface{}) (*Dag, error) {
	if isComplexObj(val) {
		return nil, fmt.Errorf("can not set complex objects, use asLink=true: %v", val)
	}
	return d.set(ctx, pathAndKey, val, false)
}

// SetAsLink sets a value at a path and returns a new dag with a new tip that reflects
// the new state (and adds the old tip to oldTips)
func (d *Dag) SetAsLink(ctx context.Context, pathAndKey []string, val interface{}) (*Dag, error) {
	return d.set(ctx, pathAndKey, val, true)
}

func (d *Dag) getExisting(ctx context.Context, path []string) (val map[string]interface{}, remainingPath []string, err error) {
	existing, remaining, err := d.Resolve(ctx, path)
	if err != nil {
		return nil, nil, err
	}

	if len(remaining) == len(path) {
		// special case so we don't clobber other keys set at the root level
		existing, _, _ = d.Resolve(ctx, []string{})
	}

	switch existing := existing.(type) {
	case map[string]interface{}:
		return existing, remaining, nil
	case nil:
		// nil can be returned when an object exists at a part of the path, but the next
		// segment of the path (a key in the object) does not exist.
		// In those cases, fetch the next object up (parent of remaining) and use that
		// as the existing object. Still needs to return remaining from the original call though
		// since thats where the caller needs to manipulate up to
		existingAncestor, _, err := d.getExisting(ctx, path[:len(path)-len(remaining)])
		return existingAncestor, remaining, err
	default:
		return make(map[string]interface{}), remaining, nil
	}
}

func (d *Dag) set(ctx context.Context, pathAndKey []string, val interface{}, asLink bool) (*Dag, error) {
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

	// lookup existing portion of path & leaf node's value
	leafNodeObj, remainingPath, err := d.getExisting(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error resolving path %s: %v", path, err)
	}

	existingPath := path[:len(path)-len(remainingPath)]
	/*
		Alright, there are basically three possible scenarios now:
		1. The path we're setting doesn't exist at all.
		   leafNodeObj will be nil and remainingPath will be == path
		2. The path we're setting partially exists.
		   leafNodeObj will be the last existing node and remainingPath will be
		   the path elements that don't exist yet.
		3. The path we're setting fully exists.
		   leafNodeObj will be the node we want to set key and val in
		   (respecting asLink) and remainingPath will be empty.
	*/

	// create the new leaf node object or use the existing one if the path fully exists
	var newLeafNodeObj map[string]interface{}
	if len(remainingPath) > 0 {
		newLeafNodeObj = make(map[string]interface{})
	} else {
		newLeafNodeObj = leafNodeObj
	}

	// set key to (link to) val in new leaf node
	if asLink {
		// create val as new node and set its CID under key in new leaf node
		newLinkNode, err := d.CreateNode(ctx, val)
		if err != nil {
			return nil, fmt.Errorf("error creating node: %v", err)
		}
		newLeafNodeObj[key] = newLinkNode.Cid()
	} else {
		// set key to val in new leaf node
		newLeafNodeObj[key] = val
	}

	// create any missing path segments, starting with the new leaf node
	// go up (i.e. right to left) the path segments, linking them as we go
	nextNodeObj := newLeafNodeObj
	for i := len(remainingPath) - 1; i >= 0; i-- {
		nextNode, err := d.CreateNode(ctx, nextNodeObj)
		if err != nil {
			return nil, fmt.Errorf("error creating node for path element %s: %v", remainingPath[i], err)
		}

		if i > 0 || leafNodeObj == nil {
			nextNodeObj = make(map[string]interface{})
		} else {
			nextNodeObj = leafNodeObj
		}

		nextNodeObj[remainingPath[i]] = nextNode.Cid()
	}

	// update former leaf node to (link to) new val
	newDag, err := d.Update(ctx, existingPath, nextNodeObj)
	if err != nil {
		return nil, fmt.Errorf("error updating DAG: %v", err)
	}

	newDag.oldTips = append(d.oldTips, newDag.Tip)

	return newDag, nil
}

func (d *Dag) dumpNode(ctx context.Context, node format.Node, isLink bool) interface{} {
	var nodeData interface{}
	if err := cbornode.DecodeInto(node.RawData(), &nodeData); err != nil {
		panic(err)
	}

	switch nodeData.(type) {
	case map[interface{}]interface{}:
		nodeMap := nodeData.(map[interface{}]interface{})
		for k, v := range nodeMap {
			switch v := v.(type) {
			case cid.Cid:
				node, _ := d.store.Get(ctx, v)
				if node == nil {
					nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
				} else {
					nodeMap[k] = d.dumpNode(ctx, node, true)
				}
			case *cid.Cid:
				node, _ := d.store.Get(ctx, *v)
				if node == nil {
					nodeMap[k] = fmt.Sprintf("non existant link: %s", v.String())
				} else {
					nodeMap[k] = d.dumpNode(ctx, node, true)
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
func (d *Dag) Dump(ctx context.Context) string {
	rootNode, _ := d.store.Get(ctx, d.Tip)
	nodes, _ := d.Nodes(ctx)
	nodeStrings := make([]string, len(nodes))
	for i, node := range nodes {
		nodeJSON, err := node.(*cbornode.Node).MarshalJSON()
		if err != nil {
			panic(fmt.Sprintf("error marshalling JSON for node %v: %v", node, err))
		}
		nodeStrings[i] = fmt.Sprintf("%v : %v", node.Cid().String(), string(nodeJSON))
	}
	return fmt.Sprintf(`
Tip: %s,
Tree:
%s

Nodes:
%v

	`,
		d.Tip.String(),
		spew.Sdump(d.dumpNode(ctx, rootNode, false)),
		strings.Join(nodeStrings, "\n\n"))
}
