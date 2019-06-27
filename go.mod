module github.com/quorumcontrol/chaintree

go 1.12

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/golang/snappy v0.0.1 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-ipfs v0.0.0-20190623000000-810cb607ede890684932b7875008d2a73387fa8d // 0.4.21 + badger fix ( https://github.com/ipfs/go-ipfs/pull/6461 )
	github.com/ipfs/go-ipfs-config v0.0.6
	github.com/ipfs/go-ipfs-http-client v0.0.3
	github.com/ipfs/go-ipld-cbor v0.0.2
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/interface-go-ipfs-core v0.1.0
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/multiformats/go-multihash v0.0.5
	github.com/polydawn/refmt v0.0.0-20190408063855-01bf1e26dd14
	github.com/quorumcontrol/messages/build/go v0.0.0-20190530182608-30c127bffefb
	github.com/quorumcontrol/namedlocker v0.0.0-20180808140020-3f797c8b12b1
	github.com/quorumcontrol/storage v1.1.4-0.20190627145136-66f3c501461d
	github.com/stretchr/testify v1.3.0
)

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.0.3

replace github.com/ipfs/go-ds-badger v0.0.4 => github.com/quorumcontrol/go-ds-badger v0.0.5-0.20190627151331-3e538695d3b3
