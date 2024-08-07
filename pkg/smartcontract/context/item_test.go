package context

import (
	"testing"

	"github.com/epicchainlabs/epicchain-go/internal/random"
	"github.com/epicchainlabs/epicchain-go/internal/testserdes"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
	"github.com/stretchr/testify/require"
)

func TestContextItem_AddSignature(t *testing.T) {
	item := &Item{Signatures: make(map[string][]byte)}

	priv1, err := keys.NewPrivateKey()
	require.NoError(t, err)

	pub1 := priv1.PublicKey()
	sig1 := []byte{1, 2, 3}
	item.AddSignature(pub1, sig1)
	require.Equal(t, sig1, item.GetSignature(pub1))

	priv2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	pub2 := priv2.PublicKey()
	sig2 := []byte{5, 6, 7}
	item.AddSignature(pub2, sig2)
	require.Equal(t, sig2, item.GetSignature(pub2))
	require.Equal(t, sig1, item.GetSignature(pub1))
}

func TestContextItem_MarshalJSON(t *testing.T) {
	priv1, err := keys.NewPrivateKey()
	require.NoError(t, err)

	priv2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	expected := &Item{
		Script: []byte{1, 2, 3},
		Parameters: []smartcontract.Parameter{{
			Type:  smartcontract.SignatureType,
			Value: random.Bytes(keys.SignatureLen),
		}},
		Signatures: map[string][]byte{
			priv1.PublicKey().StringCompressed(): random.Bytes(keys.SignatureLen),
			priv2.PublicKey().StringCompressed(): random.Bytes(keys.SignatureLen),
		},
	}

	testserdes.MarshalUnmarshalJSON(t, expected, new(Item))

	// Empty script.
	expected.Script = nil
	testserdes.MarshalUnmarshalJSON(t, expected, new(Item))
}
