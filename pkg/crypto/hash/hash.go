package hash

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/nspcc-dev/neo-go/pkg/util"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // SA1019: package golang.org/x/crypto/ripemd160 is deprecated
)

// Hashable represents an object which can be hashed. Usually, these objects
// are io.Serializable and signable. They tend to cache the hash inside for
// effectiveness, providing this accessor method. Anything that can be
// identified with a hash can then be signed and verified.
type Hashable interface {
	Hash() util.Uint256
}

// GetSignedData returns the concatenated byte slice containing of the network
// magic in constant-length 4-bytes LE representation and hashable item hash in BE
// representation.
func GetSignedData(net uint32, hh Hashable) []byte {
	var b = make([]byte, 4+util.Uint256Size)
	binary.LittleEndian.PutUint32(b, net)
	h := hh.Hash()
	copy(b[4:], h[:])
	return b
}

// NetSha256 calculates a network-specific hash of the Hashable item that can then
// be signed/verified.
func NetSha256(net uint32, hh Hashable) util.Uint256 {
	return Sha256(GetSignedData(net, hh))
}

// Sha256 hashes the incoming byte slice
// using the sha256 algorithm.
func Sha256(data []byte) util.Uint256 {
	hash := sha256.Sum256(data)
	return hash
}

// DoubleSha256 performs sha256 twice on the given data.
func DoubleSha256(data []byte) util.Uint256 {
	var hash util.Uint256

	h1 := Sha256(data)
	hash = Sha256(h1.BytesBE())
	return hash
}

// RipeMD160 performs the RIPEMD160 hash algorithm
// on the given data.
func RipeMD160(data []byte) util.Uint160 {
	var hash util.Uint160
	hasher := ripemd160.New()
	_, _ = hasher.Write(data)

	hasher.Sum(hash[:0])
	return hash
}

// Hash160 performs sha256 and then ripemd160
// on the given data.
func Hash160(data []byte) util.Uint160 {
	h1 := Sha256(data)
	h2 := RipeMD160(h1.BytesBE())

	return h2
}

// Checksum returns the checksum for a given piece of data
// using DoubleSha256 as the hash algorithm. It returns the
// first 4 bytes of the resulting slice.
func Checksum(data []byte) []byte {
	hash := DoubleSha256(data)
	return hash[:4]
}
