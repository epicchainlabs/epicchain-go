package transaction

import (
	"github.com/epicchainlabs/epicchain-go/pkg/io"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
)

// Conflicts represents attribute for conflicting transactions.
type Conflicts struct {
	Hash util.Uint256 `json:"hash"`
}

// DecodeBinary implements the io.Serializable interface.
func (c *Conflicts) DecodeBinary(br *io.BinReader) {
	c.Hash.DecodeBinary(br)
}

// EncodeBinary implements the io.Serializable interface.
func (c *Conflicts) EncodeBinary(w *io.BinWriter) {
	c.Hash.EncodeBinary(w)
}

func (c *Conflicts) toJSONMap(m map[string]any) {
	m["hash"] = c.Hash
}

// Copy implements the AttrValue interface.
func (c *Conflicts) Copy() AttrValue {
	return &Conflicts{
		Hash: c.Hash,
	}
}
