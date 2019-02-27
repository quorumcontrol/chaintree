package nodestore

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ipsn/go-ipfs/commands"
	"github.com/ipsn/go-ipfs/core"
	corehttp "github.com/ipsn/go-ipfs/core/corehttp"
	coremock "github.com/ipsn/go-ipfs/core/mock"
	config "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-ipfs-config"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

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
		Online:     true,
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

	store := NewIpldHttpStore(&IpldHttpStoreConfig{
		Address: apiMaddr,
	})

	SubtestAll(t, store)
}
