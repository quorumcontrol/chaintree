package nodestore

import (
	"testing"

	coremock "github.com/ipsn/go-ipfs/core/mock"
	"github.com/stretchr/testify/require"
)

func TestIpldBased(t *testing.T) {
	node, err := coremock.NewMockNode()
	require.Nil(t, err)
	store := NewIpldStore(node)
	SubtestAll(t, store)
}
