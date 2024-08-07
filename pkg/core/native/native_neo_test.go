package native

import (
	"math/big"
	"testing"

	"github.com/epicchainlabs/epicchain-go/internal/testserdes"
)

func TestCandidate_Bytes(t *testing.T) {
	expected := &candidate{
		Registered: true,
		Votes:      *big.NewInt(0x0F),
	}
	actual := new(candidate)
	testserdes.ToFromStackItem(t, expected, actual)
}
