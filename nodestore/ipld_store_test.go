package nodestore

import (
	"context"
	"testing"

	"github.com/ipsn/go-ipfs/core"
	coremock "github.com/ipsn/go-ipfs/core/mock"
	mocknet "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
)

func TestIpldBased(t *testing.T) {
	ctx := context.Background()
	node, err := core.NewNode(ctx, &core.BuildCfg{
		Online: false,
		Host:   coremock.MockHostOption(mocknet.New(ctx)),
	})
	require.Nil(t, err)
	store := NewIpldStore(node)
	SubtestAll(t, store)
}
