package invalid7

import "github.com/epicchainlabs/epicchain-go/pkg/interop/runtime"

type SomeStruct struct {
	Field int
	// RPC binding generator will convert this field into exported, which matches
	// exactly the existing Field.
	field int
}

func Main() {
	runtime.Notify("SomeEvent", SomeStruct{Field: 123, field: 123})
}
