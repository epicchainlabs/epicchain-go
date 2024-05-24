package testdata

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
)

func Verify() bool {
	return true
}

func OnNEP17Payment(from interop.Hash160, amount int, data any) {
}

// OnNEP11Payment notifies about NEP-11 payment. You don't call this method directly,
// instead it's called by NEP-11 contract when you transfer funds from your address
// to the address of this NFT contract.
func OnNEP11Payment(from interop.Hash160, amount int, token []byte, data any) {
	runtime.Notify("OnNEP11Payment", from, amount, token, data)
}
