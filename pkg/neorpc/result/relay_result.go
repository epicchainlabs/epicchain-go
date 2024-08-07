package result

import "github.com/epicchainlabs/epicchain-go/pkg/util"

// RelayResult ia a result of `sendrawtransaction` or `submitblock` RPC calls.
type RelayResult struct {
	Hash util.Uint256 `json:"hash"`
}
