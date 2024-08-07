package payload

import (
	"testing"

	"github.com/epicchainlabs/epicchain-go/internal/testserdes"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/hash"
	"github.com/stretchr/testify/require"
)

func TestGetBlockEncodeDecode(t *testing.T) {
	start := hash.Sha256([]byte("a"))

	p := NewGetBlocks(start, 124)
	testserdes.EncodeDecodeBinary(t, p, new(GetBlocks))

	// invalid count
	p = NewGetBlocks(start, -2)
	data, err := testserdes.EncodeBinary(p)
	require.NoError(t, err)
	require.Error(t, testserdes.DecodeBinary(data, new(GetBlocks)))

	// invalid count
	p = NewGetBlocks(start, 0)
	data, err = testserdes.EncodeBinary(p)
	require.NoError(t, err)
	require.Error(t, testserdes.DecodeBinary(data, new(GetBlocks)))
}
