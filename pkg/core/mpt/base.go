package mpt

import (
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/crypto/hash"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

// BaseNode implements basic things every node needs like caching hash and
// serialized representation. It's a basic node building block intended to be
// included into all node types.
type BaseNode struct {
	hash       util.Uint256
	bytes      []byte
	hashValid  bool
	bytesValid bool
}

// BaseNodeIface abstracts away basic Node functions.
type BaseNodeIface interface {
	Hash() util.Uint256
	Type() NodeType
	Bytes() []byte
}

type flushedNode interface {
	setCache([]byte, util.Uint256)
}

func (b *BaseNode) setCache(bs []byte, h util.Uint256) {
	b.bytes = bs
	b.hash = h
	b.bytesValid = true
	b.hashValid = true
}

// getHash returns the hash of this BaseNode.
func (b *BaseNode) getHash(n Node) util.Uint256 {
	if !b.hashValid {
		b.updateHash(n)
	}
	return b.hash
}

// getBytes returns a slice of bytes representing this node.
func (b *BaseNode) getBytes(n Node) []byte {
	if !b.bytesValid {
		b.updateBytes(n)
	}
	return b.bytes
}

// updateHash updates the hash field for this BaseNode.
func (b *BaseNode) updateHash(n Node) {
	if n.Type() == HashT || n.Type() == EmptyT {
		panic("can't update hash for empty or hash node")
	}
	b.hash = hash.DoubleSha256(b.getBytes(n))
	b.hashValid = true
}

// updateCache updates the hash and bytes fields for this BaseNode.
func (b *BaseNode) updateBytes(n Node) {
	bw := io.NewBufBinWriter()
	bw.Grow(1 + n.Size())
	encodeNodeWithType(n, bw.BinWriter)
	b.bytes = bw.Bytes()
	b.bytesValid = true
}

// invalidateCache sets all cache fields to invalid state.
func (b *BaseNode) invalidateCache() {
	b.bytesValid = false
	b.hashValid = false
}

func encodeBinaryAsChild(n Node, w *io.BinWriter) {
	if isEmpty(n) {
		w.WriteB(byte(EmptyT))
		return
	}
	w.WriteB(byte(HashT))
	w.WriteBytes(n.Hash().BytesBE())
}

// encodeNodeWithType encodes the node together with its type.
func encodeNodeWithType(n Node, w *io.BinWriter) {
	w.WriteB(byte(n.Type()))
	n.EncodeBinary(w)
}

// DecodeNodeWithType decodes the node together with its type.
func DecodeNodeWithType(r *io.BinReader) Node {
	if r.Err != nil {
		return nil
	}
	var n Node
	switch typ := NodeType(r.ReadB()); typ {
	case BranchT:
		n = new(BranchNode)
	case ExtensionT:
		n = new(ExtensionNode)
	case HashT:
		n = &HashNode{
			BaseNode: BaseNode{
				hashValid: true,
			},
		}
	case LeafT:
		n = new(LeafNode)
	case EmptyT:
		n = EmptyNode{}
	default:
		r.Err = fmt.Errorf("invalid node type: %x", typ)
		return nil
	}
	n.DecodeBinary(r)
	return n
}
