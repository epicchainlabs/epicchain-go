package payload

import (
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/network/capability"
)

// MaxUserAgentLength is the limit for the user agent field.
const MaxUserAgentLength = 1024

// Version payload.
type Version struct {
	// NetMode of the node
	Magic netmode.Magic
	// currently the version of the protocol is 0
	Version uint32
	// timestamp
	Timestamp uint32
	// it's used to distinguish several nodes using the same public IP (or different ones)
	Nonce uint32
	// client id
	UserAgent []byte
	// List of available network services
	Capabilities capability.Capabilities
}

// NewVersion returns a pointer to a Version payload.
func NewVersion(magic netmode.Magic, id uint32, ua string, c []capability.Capability) *Version {
	return &Version{
		Magic:        magic,
		Version:      0,
		Timestamp:    uint32(time.Now().UTC().Unix()),
		Nonce:        id,
		UserAgent:    []byte(ua),
		Capabilities: c,
	}
}

// DecodeBinary implements the Serializable interface.
func (p *Version) DecodeBinary(br *io.BinReader) {
	p.Magic = netmode.Magic(br.ReadU32LE())
	p.Version = br.ReadU32LE()
	p.Timestamp = br.ReadU32LE()
	p.Nonce = br.ReadU32LE()
	p.UserAgent = br.ReadVarBytes(MaxUserAgentLength)
	p.Capabilities.DecodeBinary(br)
}

// EncodeBinary implements the Serializable interface.
func (p *Version) EncodeBinary(bw *io.BinWriter) {
	bw.WriteU32LE(uint32(p.Magic))
	bw.WriteU32LE(p.Version)
	bw.WriteU32LE(p.Timestamp)
	bw.WriteU32LE(p.Nonce)
	bw.WriteVarBytes(p.UserAgent)
	p.Capabilities.EncodeBinary(bw)
}
