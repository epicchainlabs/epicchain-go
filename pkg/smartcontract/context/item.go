package context

import (
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
)

// Item represents a transaction context item.
type Item struct {
	Script     []byte                    `json:"script"`
	Parameters []smartcontract.Parameter `json:"parameters"`
	Signatures map[string][]byte         `json:"signatures"`
}

// GetSignature returns a signature for the pub if present.
func (it *Item) GetSignature(pub *keys.PublicKey) []byte {
	return it.Signatures[pub.StringCompressed()]
}

// AddSignature adds a signature for the pub.
func (it *Item) AddSignature(pub *keys.PublicKey, sig []byte) {
	pubHex := pub.StringCompressed()
	it.Signatures[pubHex] = sig
}
