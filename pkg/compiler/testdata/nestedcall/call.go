package nestedcall

import "github.com/nspcc-dev/neo-go/pkg/compiler/testdata/nestedcall/inner"

// X is what we use.
const X = 42

// N returns inner.A().
func N() int {
	return inner.Return65()
}

// F calls G.
func F() int {
	a := 1
	return G() + a
}

// G calls x and returns y().
func G() int {
	x()
	z := 3
	return y() + z
}

func x() {}
func y() int {
	tmp := 10
	return tmp
}

// Token is stateless token.
type Token struct{}

// Method is a method.
func (t Token) Method() int {
	return t.Inner()
}

// Inner is a function to be called in Method.
func (t Token) Inner() int {
	return 2231
}
