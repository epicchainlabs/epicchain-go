package consensus

import (
	"github.com/nspcc-dev/dbft"
	"github.com/epicchainlabs/epicchain-go/pkg/io"
)

// recoveryRequest represents dBFT RecoveryRequest message.
type recoveryRequest struct {
	timestamp uint64
}

var _ dbft.RecoveryRequest = (*recoveryRequest)(nil)

// DecodeBinary implements the io.Serializable interface.
func (m *recoveryRequest) DecodeBinary(r *io.BinReader) {
	m.timestamp = r.ReadU64LE()
}

// EncodeBinary implements the io.Serializable interface.
func (m *recoveryRequest) EncodeBinary(w *io.BinWriter) {
	w.WriteU64LE(m.timestamp)
}

// Timestamp implements the payload.RecoveryRequest interface.
func (m *recoveryRequest) Timestamp() uint64 { return m.timestamp * nsInMs }
