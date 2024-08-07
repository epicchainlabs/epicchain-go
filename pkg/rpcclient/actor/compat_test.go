package actor_test

import (
	"testing"

	"github.com/epicchainlabs/epicchain-go/pkg/rpcclient"
	"github.com/epicchainlabs/epicchain-go/pkg/rpcclient/actor"
)

func TestRPCActorRPCClientCompat(t *testing.T) {
	_ = actor.RPCActor(&rpcclient.WSClient{})
	_ = actor.RPCActor(&rpcclient.Client{})
}
