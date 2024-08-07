package result

import (
	"github.com/epicchainlabs/epicchain-go/pkg/core/mempoolevent"
	"github.com/epicchainlabs/epicchain-go/pkg/network/payload"
)

// NotaryRequestEvent represents a P2PNotaryRequest event either added or removed
// from the notary payload pool.
type NotaryRequestEvent struct {
	Type          mempoolevent.Type         `json:"type"`
	NotaryRequest *payload.P2PNotaryRequest `json:"notaryrequest"`
}
