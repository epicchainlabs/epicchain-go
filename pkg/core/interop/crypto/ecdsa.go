package crypto

import (
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/epicchainlabs/epicchain-go/pkg/core/fee"
	"github.com/epicchainlabs/epicchain-go/pkg/core/interop"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/hash"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/vm"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/stackitem"
)

// ECDSASecp256r1CheckMultisig checks multiple ECDSA signatures at once using
// Secp256r1 elliptic curve.
func ECDSASecp256r1CheckMultisig(ic *interop.Context) error {
	pkeys, err := ic.VM.Estack().PopSigElements()
	if err != nil {
		return fmt.Errorf("wrong key parameters: %w", err)
	}
	if !ic.VM.AddGas(ic.BaseExecFee() * fee.ECDSAVerifyPrice * int64(len(pkeys))) {
		return errors.New("gas limit exceeded")
	}
	sigs, err := ic.VM.Estack().PopSigElements()
	if err != nil {
		return fmt.Errorf("wrong signature parameters: %w", err)
	}
	// It's ok to have more keys than there are signatures (it would
	// just mean that some keys didn't sign), but not the other way around.
	if len(pkeys) < len(sigs) {
		return errors.New("more signatures than there are keys")
	}
	sigok := vm.CheckMultisigPar(ic.VM, elliptic.P256(), hash.NetSha256(ic.Network, ic.Container).BytesBE(), pkeys, sigs)
	ic.VM.Estack().PushItem(stackitem.Bool(sigok))
	return nil
}

// ECDSASecp256r1CheckSig checks ECDSA signature using Secp256r1 elliptic curve.
func ECDSASecp256r1CheckSig(ic *interop.Context) error {
	keyb := ic.VM.Estack().Pop().Bytes()
	signature := ic.VM.Estack().Pop().Bytes()
	pkey, err := keys.NewPublicKeyFromBytes(keyb, elliptic.P256())
	if err != nil {
		return err
	}
	res := pkey.VerifyHashable(signature, ic.Network, ic.Container)
	ic.VM.Estack().PushItem(stackitem.Bool(res))
	return nil
}
