package iavlite

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	encoding "github.com/kocubinski/iavlite/internal"
)

type NodeData struct {
	key          []byte
	value        []byte
	hash         []byte
	leftNodeKey  *NodeKey
	rightNodeKey *NodeKey
	size         int64
	height       int8
}

func (pn *NodeData) IsLeaf() bool {
	if pn == nil {
		panic("node is nil")
	}
	return pn.height == 0
}

type DbNode struct {
	*NodeData
	*NodeKey
}

type Node struct {
	*NodeData
	nodeKey   *NodeKey
	leftNode  *Node
	rightNode *Node
	persisted bool
}

type NodeKey struct {
	version  int64
	sequence uint32
}

func (nk *NodeKey) Bytes() []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint64(b, uint64(nk.version))
	binary.BigEndian.PutUint32(b[8:], nk.sequence)
	return b
}

func (n *Node) left(db *db) *Node {
	if n.leftNode != nil {
		return n.leftNode
	}
	left, err := db.Get(n.leftNodeKey.Bytes())
	if err != nil {
		panic(fmt.Sprintf("failed to get left node: %v", err))
	}
	return left
}

func (n *Node) right(db *db) *Node {
	if n.rightNode != nil {
		return n.rightNode
	}
	right, err := db.Get(n.rightNodeKey.Bytes())
	if err != nil {
		panic(fmt.Sprintf("failed to get right node: %v", err))
	}
	return right
}

func (n *Node) calcHeightAndSize() {
	if n.leftNode == nil || n.rightNode == nil {
		panic("left or right node is nil")
	}
	n.height = maxInt8(n.leftNode.height, n.rightNode.height)
	n.size = n.leftNode.size + n.rightNode.size
}

func (n *Node) calcBalance() int {
	if n.leftNode == nil || n.rightNode == nil {
		panic("left or right node is nil")
	}
	return int(n.leftNode.height) - int(n.rightNode.height)
}

// Computes the hash of the node without computing its descendants. Must be
// called on nodes which have descendant node hashes already computed.
func (node *Node) computeHash(version int64) ([]byte, error) {
	if node.hash != nil {
		return node.hash, nil
	}

	h := sha256.New()
	if err := node.writeHashBytes(h, version); err != nil {
		return nil, err
	}
	node.hash = h.Sum(nil)

	return node.hash, nil
}

// Writes the node's hash to the given io.Writer. This function expects
// child hashes to be already set.
func (node *Node) writeHashBytes(w io.Writer, version int64) error {
	err := encoding.EncodeVarint(w, int64(node.height))
	if err != nil {
		return fmt.Errorf("writing height, %w", err)
	}
	err = encoding.EncodeVarint(w, node.size)
	if err != nil {
		return fmt.Errorf("writing size, %w", err)
	}
	err = encoding.EncodeVarint(w, version)
	if err != nil {
		return fmt.Errorf("writing version, %w", err)
	}

	// Key is not written for inner nodes, unlike writeBytes.

	if node.IsLeaf() {
		err = encoding.EncodeBytes(w, node.key)
		if err != nil {
			return fmt.Errorf("writing key, %w", err)
		}

		// Indirection needed to provide proofs without values.
		// (e.g. ProofLeafNode.ValueHash)
		valueHash := sha256.Sum256(node.value)

		err = encoding.EncodeBytes(w, valueHash[:])
		if err != nil {
			return fmt.Errorf("writing value, %w", err)
		}
	} else {
		if node.leftNode == nil || node.rightNode == nil {
			return fmt.Errorf("found an empty child")
		}
		err = encoding.EncodeBytes(w, node.leftNode.hash)
		if err != nil {
			return fmt.Errorf("writing left hash, %w", err)
		}
		err = encoding.EncodeBytes(w, node.rightNode.hash)
		if err != nil {
			return fmt.Errorf("writing right hash, %w", err)
		}
	}

	return nil
}

func maxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}
