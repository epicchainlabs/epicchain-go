package neo

import "github.com/epicchainlabs/epicchain-go/pkg/interop"

// Candidate represents a single native Neo candidate.
type Candidate struct {
	Key   interop.PublicKey
	Votes int
}
