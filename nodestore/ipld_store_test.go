package nodestore

import (
	"fmt"
	"net"
	"testing"
	"time"

	config "github.com/ipfs/go-ipfs-config"
	ipfsHttpClient "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	corehttp "github.com/ipfs/go-ipfs/core/corehttp"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestIpldBased(t *testing.T) {
	node, err := coremock.NewMockNode()
	require.Nil(t, err)
	api, err := coreapi.NewCoreAPI(node)
	require.Nil(t, err)
	store := NewIpldStore(api)
	SubtestAll(t, store)
}

func TestIpldHttpBased(t *testing.T) {
	node, err := coremock.NewMockNode()
	require.Nil(t, err)
	defer node.Close()

	freePort, err := getFreePort()
	require.Nil(t, err)

	apiMaddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", freePort))
	require.Nil(t, err)

	cfg, err := node.Repo.Config()
	require.Nil(t, err)

	cfg.Addresses.API = []string{apiMaddr.String()}

	cmdContext := commands.Context{
		ConfigRoot: "/tmp/.mockipfsconfig",
		ReqLog:     &commands.ReqLog{},
		LoadConfig: func(path string) (*config.Config, error) {
			return cfg, nil
		},
		ConstructNode: func() (*core.IpfsNode, error) {
			return node, nil
		},
	}

	go func() {
		err := corehttp.ListenAndServe(node, apiMaddr.String(), []corehttp.ServeOption{corehttp.CommandsOption(cmdContext)}...)
		require.Nil(t, err)
	}()

	time.Sleep(50 * time.Millisecond)

	api, _ := ipfsHttpClient.NewApi(apiMaddr)

	store := NewIpldStore(api)

	SubtestAll(t, store)
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
