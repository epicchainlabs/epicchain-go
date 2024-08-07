package transaction

import (
	"github.com/epicchainlabs/epicchain-go/pkg/io"
)

// NotaryAssisted represents attribute for notary service transactions.
type NotaryAssisted struct {
	NKeys uint8 `json:"nkeys"`
}

// DecodeBinary implements the io.Serializable interface.
func (n *NotaryAssisted) DecodeBinary(br *io.BinReader) {
	n.NKeys = br.ReadB()
}

// EncodeBinary implements the io.Serializable interface.
func (n *NotaryAssisted) EncodeBinary(w *io.BinWriter) {
	w.WriteB(n.NKeys)
}

func (n *NotaryAssisted) toJSONMap(m map[string]any) {
	m["nkeys"] = n.NKeys
}

// Copy implements the AttrValue interface.
func (n *NotaryAssisted) Copy() AttrValue {
	return &NotaryAssisted{
		NKeys: n.NKeys,
	}
}
