package messages

import (
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
)

func init() {
	cbornode.RegisterCborType(Any{})
	cbornode.RegisterCborType(GetNode{})
	cbornode.RegisterCborType(GetNodeResponse{})
	cbornode.RegisterCborType(Start{})
	cbornode.RegisterCborType(Finished{})
}

type Any struct {
	Type    int
	Payload []byte
}

type Start struct {
	Tip   cid.Cid
	Nodes [][]byte
}

type Finished struct {
	Result []byte
}

type GetNode struct {
	Cid cid.Cid
}

type GetNodeResponse struct {
	Cid  cid.Cid
	Node []byte
}

// type GetTip struct {
// 	Did string
// }

// type GetTipResponse struct {
// 	Did string
// 	Cid cid.Cid
// }
