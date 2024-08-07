package params

import (
	"errors"
	"fmt"

	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/io"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract/callflag"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/emit"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/opcode"
)

// ExpandFuncParameterIntoScript pushes provided FuncParam parameter
// into the given buffer.
func ExpandFuncParameterIntoScript(script *io.BinWriter, fp FuncParam) error {
	switch fp.Type {
	case smartcontract.ByteArrayType:
		str, err := fp.Value.GetBytesBase64()
		if err != nil {
			return err
		}
		emit.Bytes(script, str)
	case smartcontract.SignatureType:
		str, err := fp.Value.GetBytesBase64()
		if err != nil {
			return err
		}
		emit.Bytes(script, str)
	case smartcontract.StringType:
		str, err := fp.Value.GetString()
		if err != nil {
			return err
		}
		emit.String(script, str)
	case smartcontract.Hash160Type:
		hash, err := fp.Value.GetUint160FromHex()
		if err != nil {
			return err
		}
		emit.Bytes(script, hash.BytesBE())
	case smartcontract.Hash256Type:
		hash, err := fp.Value.GetUint256()
		if err != nil {
			return err
		}
		emit.Bytes(script, hash.BytesBE())
	case smartcontract.PublicKeyType:
		str, err := fp.Value.GetString()
		if err != nil {
			return err
		}
		key, err := keys.NewPublicKeyFromString(string(str))
		if err != nil {
			return err
		}
		emit.Bytes(script, key.Bytes())
	case smartcontract.IntegerType:
		bi, err := fp.Value.GetBigInt()
		if err != nil {
			return err
		}
		emit.BigInt(script, bi)
	case smartcontract.BoolType:
		val, err := fp.Value.GetBoolean() // not GetBooleanStrict(), because that's the way C# code works
		if err != nil {
			return errors.New("not a bool")
		}
		emit.Bool(script, val)
	case smartcontract.ArrayType:
		val, err := fp.Value.GetArray()
		if err != nil {
			return err
		}
		err = ExpandArrayIntoScriptAndPack(script, val)
		if err != nil {
			return err
		}
	case smartcontract.MapType:
		val, err := fp.Value.GetArray()
		if err != nil {
			return err
		}
		err = ExpandMapIntoScriptAndPack(script, val)
		if err != nil {
			return err
		}
	case smartcontract.AnyType:
		if fp.Value.IsNull() || len(fp.Value.RawMessage) == 0 {
			emit.Opcodes(script, opcode.PUSHNULL)
		}
	default:
		return fmt.Errorf("parameter type %v is not supported", fp.Type)
	}
	return script.Err
}

// ExpandArrayIntoScript pushes all FuncParam parameters from the given array
// into the given buffer in the reverse order.
func ExpandArrayIntoScript(script *io.BinWriter, slice []Param) error {
	for j := len(slice) - 1; j >= 0; j-- {
		fp, err := slice[j].GetFuncParam()
		if err != nil {
			return err
		}
		err = ExpandFuncParameterIntoScript(script, fp)
		if err != nil {
			return fmt.Errorf("param %d: %w", j, err)
		}
	}
	return script.Err
}

// ExpandArrayIntoScriptAndPack expands provided array into script and packs the
// resulting items in the array.
func ExpandArrayIntoScriptAndPack(script *io.BinWriter, slice []Param) error {
	if len(slice) == 0 {
		emit.Opcodes(script, opcode.NEWARRAY0)
		return script.Err
	}
	err := ExpandArrayIntoScript(script, slice)
	if err != nil {
		return err
	}
	emit.Int(script, int64(len(slice)))
	emit.Opcodes(script, opcode.PACK)
	return script.Err
}

// ExpandMapIntoScriptAndPack expands provided array of key-value items into script
// and packs the resulting pairs in the [stackitem.Map].
func ExpandMapIntoScriptAndPack(script *io.BinWriter, slice []Param) error {
	if len(slice) == 0 {
		emit.Opcodes(script, opcode.NEWMAP)
		return script.Err
	}
	for i := len(slice) - 1; i >= 0; i-- {
		pair, err := slice[i].GetFuncParamPair()
		if err != nil {
			return err
		}
		err = ExpandFuncParameterIntoScript(script, pair.Value)
		if err != nil {
			return fmt.Errorf("map value %d: %w", i, err)
		}
		err = ExpandFuncParameterIntoScript(script, pair.Key)
		if err != nil {
			return fmt.Errorf("map key %d: %w", i, err)
		}
	}
	emit.Int(script, int64(len(slice)))
	emit.Opcodes(script, opcode.PACKMAP)
	return script.Err
}

// CreateFunctionInvocationScript creates a script to invoke the given contract with
// the given parameters.
func CreateFunctionInvocationScript(contract util.Uint160, method string, param *Param) ([]byte, error) {
	script := io.NewBufBinWriter()
	if param == nil {
		emit.Opcodes(script.BinWriter, opcode.NEWARRAY0)
	} else if slice, err := param.GetArray(); err == nil {
		err = ExpandArrayIntoScriptAndPack(script.BinWriter, slice)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("failed to convert %s to script parameter", param)
	}

	emit.AppCallNoArgs(script.BinWriter, contract, method, callflag.All)
	return script.Bytes(), nil
}
