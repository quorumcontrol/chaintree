package nodestore

import (
	"context"
	"os"
	"testing"

	"github.com/ipsn/go-ipfs/core"
	config "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-ipfs-config"
	"github.com/ipsn/go-ipfs/plugin/loader"
	"github.com/ipsn/go-ipfs/repo/fsrepo"
	"github.com/stretchr/testify/require"
)

func TestIpldBased(t *testing.T) {
	repoPath := ".tmp"
	os.RemoveAll(repoPath)
	os.MkdirAll(repoPath, 0755)
	defer os.RemoveAll(repoPath)

	ncfg := &core.BuildCfg{
		Online:    false,
		Permanent: true,
		Routing:   core.DHTOption,
	}
	plugins, _ := loader.NewPluginLoader("")
	plugins.Initialize()
	plugins.Run()

	conf, err := config.Init(os.Stdout, 2048)
	require.Nil(t, err)

	for _, profile := range []string{"badgerds"} {
		transformer, ok := config.Profiles[profile]
		require.True(t, ok)

		err := transformer.Transform(conf)
		require.Nil(t, err)
	}

	err = fsrepo.Init(repoPath, conf)
	require.Nil(t, err)

	ctx := context.Background()
	node, err := core.NewNode(ctx, ncfg)
	require.Nil(t, err)
	store := NewIpldStore(node)

	SubtestAll(t, store)
}
