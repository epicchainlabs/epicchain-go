package vm

import (
	"math/big"
	"testing"

	"github.com/epicchainlabs/epicchain-go/pkg/vm/stackitem"
	"github.com/stretchr/testify/require"
)

func TestSlot_Get(t *testing.T) {
	rc := newRefCounter()
	var s slot
	require.Panics(t, func() { s.Size() })

	s.init(3, rc)
	require.Equal(t, 3, s.Size())
	require.Equal(t, 3, int(*rc))

	// Null is the default
	item := s.Get(2)
	require.Equal(t, stackitem.Null{}, item)

	s.Set(1, stackitem.NewBigInteger(big.NewInt(42)), rc)
	require.Equal(t, stackitem.NewBigInteger(big.NewInt(42)), s.Get(1))
	require.Equal(t, 3, int(*rc))
}
