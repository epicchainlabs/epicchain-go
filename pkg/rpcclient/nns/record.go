package nns

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/stackitem"
)

// RecordState is a type that registered entities are saved as.
type RecordState struct {
	Name string
	Type RecordType
	Data string
}

// RecordType is domain name service record types.
type RecordType byte

// Ensure RecordType implements smartcontract.Convertible for proper handling as
// a parameter to invoker.Invoker methods.
var _ = smartcontract.Convertible(RecordType(0))

// Record types are defined in [RFC 1035](https://tools.ietf.org/html/rfc1035)
const (
	// A represents address record type.
	A RecordType = 1
	// CNAME represents canonical name record type.
	CNAME RecordType = 5
	// TXT represents text record type.
	TXT RecordType = 16
)

// Record types are defined in [RFC 3596](https://tools.ietf.org/html/rfc3596)
const (
	// AAAA represents IPv6 address record type.
	AAAA RecordType = 28
)

// ToSCParameter implements smartcontract.Convertible interface.
func (r RecordType) ToSCParameter() (smartcontract.Parameter, error) {
	return smartcontract.Parameter{
		Type:  smartcontract.IntegerType,
		Value: big.NewInt(int64(r)),
	}, nil
}

// FromStackItem fills RecordState with data from the given stack item if it can
// be correctly converted to RecordState.
func (r *RecordState) FromStackItem(itm stackitem.Item) error {
	rs, ok := itm.Value().([]stackitem.Item)
	if !ok {
		return errors.New("not a struct")
	}
	if len(rs) != 3 {
		return errors.New("wrong number of elements")
	}
	name, err := rs[0].TryBytes()
	if err != nil {
		return fmt.Errorf("bad name: %w", err)
	}
	typ, err := rs[1].TryInteger()
	if err != nil {
		return fmt.Errorf("bad type: %w", err)
	}
	data, err := rs[2].TryBytes()
	if err != nil {
		return fmt.Errorf("bad data: %w", err)
	}
	u64Typ := typ.Uint64()
	if !typ.IsUint64() || u64Typ > 255 {
		return errors.New("bad type")
	}
	r.Name = string(name)
	r.Type = RecordType(u64Typ)
	r.Data = string(data)
	return nil
}
