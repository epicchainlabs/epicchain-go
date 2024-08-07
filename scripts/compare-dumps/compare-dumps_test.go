package main

import (
	"testing"

	"github.com/epicchainlabs/epicchain-go/pkg/config"
	"github.com/epicchainlabs/epicchain-go/pkg/core/native"
	"github.com/stretchr/testify/require"
)

func TestCompatibility(t *testing.T) {
	cs := native.NewContracts(config.ProtocolConfiguration{})
	require.Equal(t, cs.Ledger.ID, int32(ledgerContractID))
}
