package mpt

import (
	"encoding/json"
	"errors"

	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

const (
	// childrenCount represents the number of children of a branch node.
	childrenCount = 17
	// lastChild is the index of the last child.
	lastChild = childrenCount - 1
)

// BranchNode represents an MPT's branch node.
type BranchNode struct {
	BaseNode
	Children [childrenCount]Node
}

var _ Node = (*BranchNode)(nil)

// NewBranchNode returns a new branch node.
func NewBranchNode() *BranchNode {
	b := new(BranchNode)
	for i := 0; i < childrenCount; i++ {
		b.Children[i] = EmptyNode{}
	}
	return b
}

// Type implements the Node interface.
func (b *BranchNode) Type() NodeType { return BranchT }

// Hash implements the BaseNode interface.
func (b *BranchNode) Hash() util.Uint256 {
	return b.getHash(b)
}

// Bytes implements the BaseNode interface.
func (b *BranchNode) Bytes() []byte {
	return b.getBytes(b)
}

// Size implements the Node interface.
func (b *BranchNode) Size() int {
	sz := childrenCount
	for i := range b.Children {
		if !isEmpty(b.Children[i]) {
			sz += util.Uint256Size
		}
	}
	return sz
}

// EncodeBinary implements io.Serializable.
func (b *BranchNode) EncodeBinary(w *io.BinWriter) {
	for i := 0; i < childrenCount; i++ {
		encodeBinaryAsChild(b.Children[i], w)
	}
}

// DecodeBinary implements io.Serializable.
func (b *BranchNode) DecodeBinary(r *io.BinReader) {
	for i := 0; i < childrenCount; i++ {
		no := new(NodeObject)
		no.DecodeBinary(r)
		b.Children[i] = no.Node
	}
}

// MarshalJSON implements the json.Marshaler.
func (b *BranchNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.Children)
}

// UnmarshalJSON implements the json.Unmarshaler.
func (b *BranchNode) UnmarshalJSON(data []byte) error {
	var obj NodeObject
	if err := obj.UnmarshalJSON(data); err != nil {
		return err
	} else if u, ok := obj.Node.(*BranchNode); ok {
		*b = *u
		return nil
	}
	return errors.New("expected branch node")
}

// Clone implements Node interface.
func (b *BranchNode) Clone() Node {
	res := *b
	return &res
}

// splitPath splits path for a branch node.
func splitPath(path []byte) (byte, []byte) {
	if len(path) != 0 {
		return path[0], path[1:]
	}
	return lastChild, path
}
