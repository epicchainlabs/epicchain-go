package network

import (
	"github.com/epicchainlabs/epicchain-go/pkg/core/mpt"
	"github.com/epicchainlabs/epicchain-go/pkg/network/bqueue"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
)

// StateSync represents state sync module.
type StateSync interface {
	AddMPTNodes([][]byte) error
	bqueue.Blockqueuer
	Init(currChainHeight uint32) error
	IsActive() bool
	IsInitialized() bool
	GetUnknownMPTNodesBatch(limit int) []util.Uint256
	NeedHeaders() bool
	NeedMPTNodes() bool
	Traverse(root util.Uint256, process func(node mpt.Node, nodeBytes []byte) bool) error
}
