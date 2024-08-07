package standard

import (
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract/manifest"
)

// Nep11Payable contains NEP-11's onNEP11Payment method definition.
var Nep11Payable = &Standard{
	Manifest: manifest.Manifest{
		ABI: manifest.ABI{
			Methods: []manifest.Method{{
				Name: manifest.MethodOnNEP11Payment,
				Parameters: []manifest.Parameter{
					{Name: "from", Type: smartcontract.Hash160Type},
					{Name: "amount", Type: smartcontract.IntegerType},
					{Name: "tokenid", Type: smartcontract.ByteArrayType},
					{Name: "data", Type: smartcontract.AnyType},
				},
				ReturnType: smartcontract.VoidType,
			}},
		},
	},
}

// Nep17Payable contains NEP-17's onNEP17Payment method definition.
var Nep17Payable = &Standard{
	Manifest: manifest.Manifest{
		ABI: manifest.ABI{
			Methods: []manifest.Method{{
				Name: manifest.MethodOnNEP17Payment,
				Parameters: []manifest.Parameter{
					{Name: "from", Type: smartcontract.Hash160Type},
					{Name: "amount", Type: smartcontract.IntegerType},
					{Name: "data", Type: smartcontract.AnyType},
				},
				ReturnType: smartcontract.VoidType,
			}},
		},
	},
}
