package nef

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/crypto/hash"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
)

// NEO Executable Format 3 (NEF3)
// Standard: https://github.com/neo-project/proposals/pull/121/files
// Implementation: https://github.com/neo-project/neo/blob/v3.0.0-preview2/src/neo/SmartContract/NefFile.cs#L8
// +------------+-----------+------------------------------------------------------------+
// |   Field    |  Length   |                          Comment                           |
// +------------+-----------+------------------------------------------------------------+
// | Magic      | 4 bytes   | Magic header                                               |
// | Compiler   | 64 bytes  | Compiler used and it's version                             |
// | Source     | Var bytes | Source file URL.                                           |
// +------------+-----------+------------------------------------------------------------+
// | Reserved   | 1 byte    | Reserved for extensions. Must be 0.                        |
// | Tokens     | Var array | List of method tokens                                      |
// | Reserved   | 2-bytes   | Reserved for extensions. Must be 0.                        |
// | Script     | Var bytes | Var bytes for the payload                                  |
// +------------+-----------+------------------------------------------------------------+
// | Checksum   | 4 bytes   | First four bytes of double SHA256 hash of the header       |
// +------------+-----------+------------------------------------------------------------+

const (
	// Magic is a magic File header constant.
	Magic uint32 = 0x3346454E
	// MaxSourceURLLength is the maximum allowed source URL length.
	MaxSourceURLLength = 256
	// compilerFieldSize is the length of `Compiler` File header field in bytes.
	compilerFieldSize = 64
)

// File represents a compiled contract file structure according to the NEF3 standard.
type File struct {
	Header
	Source   string        `json:"source"`
	Tokens   []MethodToken `json:"tokens"`
	Script   []byte        `json:"script"`
	Checksum uint32        `json:"checksum"`
}

// Header represents a File header.
type Header struct {
	Magic    uint32 `json:"magic"`
	Compiler string `json:"compiler"`
}

// NewFile returns a new NEF3 file with the script specified.
func NewFile(script []byte) (*File, error) {
	file := &File{
		Header: Header{
			Magic:    Magic,
			Compiler: "neo-go-" + config.Version,
		},
		Tokens: []MethodToken{},
		Script: script,
	}
	if len(file.Compiler) > compilerFieldSize {
		return nil, errors.New("too long compiler field")
	}
	file.Checksum = file.CalculateChecksum()
	return file, nil
}

// EncodeBinary implements the io.Serializable interface.
func (h *Header) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(h.Magic)
	if len(h.Compiler) > compilerFieldSize {
		w.Err = errors.New("invalid compiler name length")
		return
	}
	var b = make([]byte, compilerFieldSize)
	copy(b, []byte(h.Compiler))
	w.WriteBytes(b)
}

// DecodeBinary implements the io.Serializable interface.
func (h *Header) DecodeBinary(r *io.BinReader) {
	h.Magic = r.ReadU32LE()
	if h.Magic != Magic {
		r.Err = errors.New("invalid Magic")
		return
	}
	buf := make([]byte, compilerFieldSize)
	r.ReadBytes(buf)
	buf = bytes.TrimRightFunc(buf, func(r rune) bool {
		return r == 0
	})
	h.Compiler = string(buf)
}

// CalculateChecksum returns first 4 bytes of double-SHA256(Header) converted to uint32.
// CalculateChecksum doesn't perform the resulting serialized NEF size check, and return
// the checksum irrespectively to the size limit constraint. It's a caller's duty to check
// the resulting NEF size.
func (n *File) CalculateChecksum() uint32 {
	bb, err := n.BytesLong()
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(hash.Checksum(bb[:len(bb)-4]))
}

// EncodeBinary implements the io.Serializable interface.
func (n *File) EncodeBinary(w *io.BinWriter) {
	n.Header.EncodeBinary(w)
	if len(n.Source) > MaxSourceURLLength {
		w.Err = errors.New("source url too long")
		return
	}
	w.WriteString(n.Source)
	w.WriteB(0)
	w.WriteArray(n.Tokens)
	w.WriteU16LE(0)
	w.WriteVarBytes(n.Script)
	w.WriteU32LE(n.Checksum)
}

var errInvalidReserved = errors.New("reserved bytes must be 0")

// DecodeBinary implements the io.Serializable interface.
func (n *File) DecodeBinary(r *io.BinReader) {
	n.Header.DecodeBinary(r)
	n.Source = r.ReadString(MaxSourceURLLength)
	reservedB := r.ReadB()
	if r.Err == nil && reservedB != 0 {
		r.Err = errInvalidReserved
		return
	}
	r.ReadArray(&n.Tokens)
	reserved := r.ReadU16LE()
	if r.Err == nil && reserved != 0 {
		r.Err = errInvalidReserved
		return
	}
	n.Script = r.ReadVarBytes(stackitem.MaxSize)
	if r.Err == nil && len(n.Script) == 0 {
		r.Err = errors.New("empty script")
		return
	}
	n.Checksum = r.ReadU32LE()
	checksum := n.CalculateChecksum()
	if r.Err == nil && checksum != n.Checksum {
		r.Err = errors.New("checksum verification failure")
		return
	}
}

// Bytes returns a byte array with a serialized NEF File. It performs the
// resulting NEF file size check and returns an error if serialized slice length
// exceeds [stackitem.MaxSize].
func (n File) Bytes() ([]byte, error) {
	return n.bytes(true)
}

// BytesLong returns a byte array with a serialized NEF File. It performs no
// resulting slice check.
func (n File) BytesLong() ([]byte, error) {
	return n.bytes(false)
}

// bytes returns the serialized NEF File representation and performs the resulting
// byte array size check if needed.
func (n File) bytes(checkSize bool) ([]byte, error) {
	buf := io.NewBufBinWriter()
	n.EncodeBinary(buf.BinWriter)
	if buf.Err != nil {
		return nil, buf.Err
	}
	res := buf.Bytes()
	if checkSize && len(res) > stackitem.MaxSize {
		return nil, fmt.Errorf("serialized NEF size exceeds VM stackitem limits: %d bytes is allowed at max, got %d", stackitem.MaxSize, len(res))
	}
	return res, nil
}

// FileFromBytes returns a NEF File deserialized from the given bytes.
func FileFromBytes(source []byte) (File, error) {
	result := File{}
	if len(source) > stackitem.MaxSize {
		return result, fmt.Errorf("invalid NEF file size: expected %d at max, got %d", stackitem.MaxSize, len(source))
	}
	r := io.NewBinReaderFromBuf(source)
	result.DecodeBinary(r)
	if r.Err != nil {
		return result, r.Err
	}
	return result, nil
}
