package interop

import (
	"github.com/epicchainlabs/epicchain-go/pkg/core/fee"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/opcode"
)

// GetPrice returns a price for executing op with the provided parameter.
func (ic *Context) GetPrice(op opcode.Opcode, parameter []byte) int64 {
	return fee.Opcode(ic.baseExecFee, op)
}
