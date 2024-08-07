package result

import "github.com/epicchainlabs/epicchain-go/pkg/util"

// RawMempool represents a result of getrawmempool RPC call.
type RawMempool struct {
	Height     uint32         `json:"height"`
	Verified   []util.Uint256 `json:"verified"`
	Unverified []util.Uint256 `json:"unverified"`
}
