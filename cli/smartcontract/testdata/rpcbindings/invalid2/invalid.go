package invalid2

import "github.com/epicchainlabs/epicchain-go/pkg/interop/runtime"

func Main() {
	runtime.Notify("SomeEvent", "p1", "p2")
}
