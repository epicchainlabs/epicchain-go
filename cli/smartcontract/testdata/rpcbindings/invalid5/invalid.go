package invalid5

import "github.com/epicchainlabs/epicchain-go/pkg/interop/runtime"

type NamedStruct struct {
	SomeInt int
}

func Main() NamedStruct {
	runtime.Notify("SomeEvent", []interface{}{123})
	return NamedStruct{SomeInt: 123}
}
