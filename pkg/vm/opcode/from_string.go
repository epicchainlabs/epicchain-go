package opcode

import "errors"

var stringToOpcode = make(map[string]Opcode)

func init() {
	for i := 0; i < 255; i++ {
		op := Opcode(i)
		stringToOpcode[op.String()] = op
	}
}

// FromString converts string representation to an opcode itself.
func FromString(s string) (Opcode, error) {
	if op, ok := stringToOpcode[s]; ok {
		return op, nil
	}
	return 0, errors.New("invalid opcode")
}
