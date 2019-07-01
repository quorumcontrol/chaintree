package nodestore

import (
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
)

// CidString is the KeyString() of a CID
type CidString string

// Cid returns the CID from the CidString (which is the KeyString format)
func (cs CidString) Cid() cid.Cid {
	cID, _ := cid.Cast([]byte(string(cs)))
	return cID
}

// ToCidString takes a CID and returns its map key (CidString)
func ToCidString(id cid.Cid) CidString {
	return CidString(id.KeyString())
}

type DagStore interface {
	format.DAGService
}
