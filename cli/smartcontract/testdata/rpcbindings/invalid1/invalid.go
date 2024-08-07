package invalid1

import "github.com/epicchainlabs/epicchain-go/pkg/interop/runtime"

func Main() {
	runtime.Notify("Non declared event")
}
