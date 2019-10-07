package dag

import (
	"context"
	"testing"

	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefCounter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sw := &safewrap.SafeWrap{}

	store, err := nodestore.MemoryStore(ctx)
	require.Nil(t, err)

	tip := sw.WrapObject(map[string]interface{}{"hi": "hi"})
	graph, err := NewDagWithNodes(ctx, store, tip)
	require.Nil(t, err)
	counter := RefCountDag(graph)

	_, _, err = counter.Resolve(ctx, []string{"/"})

	assert.Len(t, counter.Touched, 1)

	unwrapped := counter.Unwrap()
	assert.Equal(t, counter.Dag.Tip.String(), unwrapped.Tip.String())
}
