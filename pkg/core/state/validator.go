package state

import (
	"math/big"

	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
)

// Validator holds the state of a validator (its key and votes balance).
type Validator struct {
	Key   *keys.PublicKey
	Votes *big.Int
}
