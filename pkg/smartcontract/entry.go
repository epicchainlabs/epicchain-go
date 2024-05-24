package smartcontract

import (
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/core/interop/interopnames"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
)

// CreateCallAndUnwrapIteratorScript creates a script that calls 'operation' method
// of the 'contract' with the specified arguments. This method is expected to return
// an iterator that then is traversed (using iterator.Next) with values (iterator.Value)
// extracted and added into array. At most maxIteratorResultItems number of items is
// processed this way (and this number can't exceed VM limits), the result of the
// script is an array containing extracted value elements. This script can be useful
// for interactions with RPC server that have iterator sessions disabled.
func CreateCallAndUnwrapIteratorScript(contract util.Uint160, operation string, maxIteratorResultItems int, params ...any) ([]byte, error) {
	script := io.NewBufBinWriter()
	jmpIfNotOffset, jmpIfMaxReachedOffset := emitCallAndUnwrapIteratorScript(script, contract, operation, maxIteratorResultItems, params...)

	// End of the program: push the result on stack and return.
	loadResultOffset := script.Len()
	emit.Opcodes(script.BinWriter, opcode.NIP, // Remove iterator from the 1-st cell of estack
		opcode.NIP) // Remove maxIteratorResultItems from the 1-st cell of estack, so that only resulting array is left on estack.
	if err := script.Err; err != nil {
		return nil, fmt.Errorf("emitting iterator unwrapper script: %w", err)
	}

	// Fill in JMPIFNOT instruction parameter.
	bytes := script.Bytes()
	bytes[jmpIfNotOffset+1] = uint8(loadResultOffset - jmpIfNotOffset) // +1 is for JMPIFNOT itself; offset is relative to JMPIFNOT position.
	// Fill in jmpIfMaxReachedOffset instruction parameter.
	bytes[jmpIfMaxReachedOffset+1] = uint8(loadResultOffset - jmpIfMaxReachedOffset) // +1 is for JMPIF itself; offset is relative to JMPIF position.
	return bytes, nil
}

// CreateCallAndPrefetchIteratorScript creates a script that calls 'operation' method
// of the 'contract' with the specified arguments. This method is expected to return
// an array of the first iterator items (up to maxIteratorResultItems, which cannot exceed VM limits)
// and, optionally, an iterator that then is traversed (using iterator.Next).
// The result of the script is an array containing extracted value elements and an iterator, if it can contain more items.
// If the iterator is present, it lies on top of the stack.
// Note, however, that if an iterator is returned, the number of remaining items can still be 0.
// This script should only be used for interactions with RPC server that have iterator sessions enabled.
func CreateCallAndPrefetchIteratorScript(contract util.Uint160, operation string, maxIteratorResultItems int, params ...any) ([]byte, error) {
	script := io.NewBufBinWriter()
	jmpIfNotOffset, jmpIfMaxReachedOffset := emitCallAndUnwrapIteratorScript(script, contract, operation, maxIteratorResultItems, params...)

	// 1st possibility: jump here when the maximum number of items was reached.
	retainIteratorOffset := script.Len()
	emit.Opcodes(script.BinWriter, opcode.ROT, // Put maxIteratorResultItems from the 2-nd cell of estack, to the top
		opcode.DROP, // ... and then drop it.
		opcode.SWAP, // Put the iterator on top of the stack.
		opcode.RET)

	// 2nd possibility: jump here when the iterator has no more items.
	loadResultOffset := script.Len()
	emit.Opcodes(script.BinWriter, opcode.ROT, // Put maxIteratorResultItems from the 2-nd cell of estack, to the top
		opcode.DROP, // ... and then drop it.
		opcode.NIP)  // Drop iterator as the 1-st cell on the stack.
	if err := script.Err; err != nil {
		return nil, fmt.Errorf("emitting iterator unwrapper script: %w", err)
	}

	// Fill in JMPIFNOT instruction parameter.
	bytes := script.Bytes()
	bytes[jmpIfNotOffset+1] = uint8(loadResultOffset - jmpIfNotOffset) // +1 is for JMPIFNOT itself; offset is relative to JMPIFNOT position.
	// Fill in jmpIfMaxReachedOffset instruction parameter.
	bytes[jmpIfMaxReachedOffset+1] = uint8(retainIteratorOffset - jmpIfMaxReachedOffset) // +1 is for JMPIF itself; offset is relative to JMPIF position.
	return bytes, nil
}

func emitCallAndUnwrapIteratorScript(script *io.BufBinWriter, contract util.Uint160, operation string, maxIteratorResultItems int, params ...any) (int, int) {
	emit.Int(script.BinWriter, int64(maxIteratorResultItems))
	emit.AppCall(script.BinWriter, contract, operation, callflag.All, params...) // The System.Contract.Call itself, it will push Iterator on estack.
	emit.Opcodes(script.BinWriter, opcode.NEWARRAY0)                             // Push new empty array to estack. This array will store iterator's elements.

	// Start the iterator traversal cycle.
	iteratorTraverseCycleStartOffset := script.Len()
	emit.Opcodes(script.BinWriter, opcode.OVER)                     // Load iterator from 1-st cell of estack.
	emit.Syscall(script.BinWriter, interopnames.SystemIteratorNext) // Call System.Iterator.Next, it will pop the iterator from estack and push `true` or `false` to estack.
	jmpIfNotOffset := script.Len()
	emit.Instruction(script.BinWriter, opcode.JMPIFNOT, // Pop boolean value (from the previous step) from estack, if `false`, then iterator has no more items => jump to the end of program.
		[]byte{
			0x00, // jump to loadResultOffset, but we'll fill this byte after script creation.
		})
	emit.Opcodes(script.BinWriter, opcode.DUP, // Duplicate the resulting array from 0-th cell of estack and push it to estack.
		opcode.PUSH2, opcode.PICK) // Pick iterator from the 2-nd cell of estack.
	emit.Syscall(script.BinWriter, interopnames.SystemIteratorValue) // Call System.Iterator.Value, it will pop the iterator from estack and push its current value to estack.
	emit.Opcodes(script.BinWriter, opcode.APPEND)                    // Pop iterator value and the resulting array from estack. Append value to the resulting array. Array is a reference type, thus, value stored at the 1-th cell of local slot will also be updated.
	emit.Opcodes(script.BinWriter, opcode.DUP,                       // Duplicate the resulting array from 0-th cell of estack and push it to estack.
		opcode.SIZE,               // Pop array from estack and push its size to estack.
		opcode.PUSH3, opcode.PICK, // Pick maxIteratorResultItems from the 3-d cell of estack.
		opcode.GE) // Compare len(arr) and maxIteratorResultItems
	jmpIfMaxReachedOffset := script.Len()
	emit.Instruction(script.BinWriter, opcode.JMPIF, // Pop boolean value (from the previous step) from estack, if `false`, then max array elements is reached => jump to the end of program.
		[]byte{
			0x00, // jump to loadResultOffset, but we'll fill this byte after script creation.
		})
	jmpOffset := script.Len()
	emit.Instruction(script.BinWriter, opcode.JMP, // Jump to the start of iterator traverse cycle.
		[]byte{
			uint8(iteratorTraverseCycleStartOffset - jmpOffset), // jump to iteratorTraverseCycleStartOffset; offset is relative to JMP position.
		})
	return jmpIfNotOffset, jmpIfMaxReachedOffset
}

// CreateCallScript returns a script that calls contract's method with
// the specified parameters. Whatever this method returns remains on the stack.
// See also (*Builder).InvokeMethod.
func CreateCallScript(contract util.Uint160, method string, params ...any) ([]byte, error) {
	b := NewBuilder()
	b.InvokeMethod(contract, method, params...)
	return b.Script()
}

// CreateCallWithAssertScript returns a script that calls contract's method with
// the specified parameters expecting a Boolean value to be return that then is
// used for ASSERT. See also (*Builder).InvokeWithAssert.
func CreateCallWithAssertScript(contract util.Uint160, method string, params ...any) ([]byte, error) {
	b := NewBuilder()
	b.InvokeWithAssert(contract, method, params...)
	return b.Script()
}
