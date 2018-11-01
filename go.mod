module github.com/QuorumControl/chaintree

require (
	github.com/AndreasBriese/bbloom v0.0.0-20180913140656-343706a395b7 // indirect
	github.com/coreos/bbolt v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.5.4 // indirect
	github.com/dgryski/go-farm v0.0.0-20180109070241-2de33835d102 // indirect
	github.com/ethereum/go-ethereum v1.8.17 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/gxed/hashland v0.0.0-20180221191214-d9f6b97f8db2 // indirect
	github.com/ipfs/go-block-format v0.2.0 // indirect
	github.com/ipfs/go-cid v0.9.0
	github.com/ipfs/go-ipfs-util v1.2.8 // indirect
	github.com/ipfs/go-ipld-cbor v0.0.0-20181027011403-4f6fa492b20f
	github.com/ipfs/go-ipld-format v0.0.0-20181027011403-b2d848bada9dbd11ec27b9a38e8726555e249fd4
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.0.0-20181005183134-51976451ce19 // indirect
	github.com/mr-tron/base58 v1.1.0 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multihash v1.0.8
	github.com/pkg/errors v0.8.0 // indirect
	github.com/polydawn/refmt v0.0.0-20181010100905-57f76afce908
	github.com/quorumcontrol/chaintree v0.0.4
	github.com/quorumcontrol/namedlocker v0.0.0-20180808140020-3f797c8b12b1
	github.com/quorumcontrol/storage v0.0.0-20181008140602-64192a5e84b2
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/stretchr/testify v1.2.2
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	golang.org/x/crypto v0.0.0-20181030102418-4d3f4d9ffa16 // indirect
	golang.org/x/net v0.0.0-20181029044818-c44066c5c816 // indirect
	golang.org/x/sys v0.0.0-20181030150119-7e31e0c00fa0 // indirect
)

replace github.com/quorumcontrol/chaintree => ./

replace github.com/quorumcontrol/chaintree/safewrap => ./safewrap

replace github.com/quorumcontrol/chaintree/nodestore => ./nodestore

replace github.com/quorumcontrol/chaintree/dag => ./dag

replace github.com/quorumcontrol/chaintree/chaintree => ./chaintree
