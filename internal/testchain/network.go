package testchain

import "github.com/epicchainlabs/epicchain-go/pkg/config/netmode"

// Network returns testchain network's magic number.
func Network() netmode.Magic {
	return netmode.UnitTestNet
}
