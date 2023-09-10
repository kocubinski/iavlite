package v6

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	encoding "github.com/kocubinski/iavlite/internal"
)

const nodeKeySize = 12

type nodeKey [nodeKeySize]byte

func newNodeKey(version int64, sequence uint32) *nodeKey {
	nk := nodeKey{}
	binary.BigEndian.PutUint64(nk[:], uint64(version))
	binary.BigEndian.PutUint32(nk[8:], sequence)
	return &nk
}

func (nk *nodeKey) Version() int64 {
	return int64(binary.BigEndian.Uint64(nk[:]))
}

func (nk *nodeKey) Sequence() uint32 {
	return binary.BigEndian.Uint32(nk[8:])
}

// Node represents a node in a Tree.
type Node struct {
	key           []byte
	value         []byte
	hash          []byte
	nodeKey       *nodeKey
	leftNodeKey   *nodeKey
	rightNodeKey  *nodeKey
	size          int64
	leftNode      *Node
	rightNode     *Node
	subtreeHeight int8

	frameId int
	use     bool
	dirty   bool
}

// String returns a string representation of the node key.
func (nk *nodeKey) String() string {
	return fmt.Sprintf("(%d, %d)", nk.Version(), nk.Sequence())
}

func (node *Node) reset() {
	node.mustChildren(nil)
	node.hash = nil
	node.nodeKey = nil
}

// TODO remove?
func (node *Node) mustChildren(t *MutableTree) {
	var err error
	if node.leftNode == nil {
		node.leftNode, err = node.getLeftNode(t)
		if err != nil {
			panic(err)
		}
	}
	if node.rightNode == nil {
		node.rightNode, err = node.getRightNode(t)
		if err != nil {
			panic(err)
		}
	}
}

// getLeftNode will never be called on leaf nodes. all tree nodes have 2 children.
func (node *Node) getLeftNode(t *MutableTree) (*Node, error) {
	if node.leftNode == nil {
		return nil, fmt.Errorf("left node is nil")
	}
	return node.leftNode, nil
}

func (node *Node) left(t *MutableTree) *Node {
	leftNode, err := node.getLeftNode(nil)
	if err != nil {
		panic(err)
	}
	return leftNode
}

func (node *Node) setLeft(leftNode *Node) {
	node.leftNodeKey = leftNode.nodeKey
	node.leftNode = leftNode
}

func (node *Node) getRightNode(t *MutableTree) (*Node, error) {
	if node.rightNode == nil {
		return nil, fmt.Errorf("right node is nil")
	}
	return node.rightNode, nil
}

func (node *Node) right(t *MutableTree) *Node {
	rightNode, err := node.getRightNode(nil)
	if err != nil {
		panic(err)
	}
	return rightNode
}

func (node *Node) setRight(rightNode *Node) {
	node.rightNodeKey = rightNode.nodeKey
	node.rightNode = rightNode
}

// NOTE: mutates height and size
func (node *Node) calcHeightAndSize(t *MutableTree) error {
	leftNode, err := node.getLeftNode(t)
	if err != nil {
		return err
	}

	rightNode, err := node.getRightNode(t)
	if err != nil {
		return err
	}

	node.subtreeHeight = maxInt8(leftNode.subtreeHeight, rightNode.subtreeHeight) + 1
	node.size = leftNode.size + rightNode.size
	return nil
}

func maxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}

// NOTE: assumes that node can be modified
// TODO: optimize balance & rotate
func (tree *MutableTree) balance(node *Node) (newSelf *Node, err error) {
	if node.nodeKey != nil {
		return nil, fmt.Errorf("unexpected balance() call on persisted node")
	}
	balance, err := node.calcBalance(tree)
	if err != nil {
		return nil, err
	}

	if balance > 1 {
		lftBalance, err := node.left(tree).calcBalance(tree)
		if err != nil {
			return nil, err
		}

		if lftBalance >= 0 {
			// Left Left Case
			newNode, err := tree.rotateRight(node)
			if err != nil {
				return nil, err
			}
			return newNode, nil
		}
		// Left Right Case
		node.leftNodeKey = nil
		newLeftNode, err := tree.rotateLeft(node.left(tree))
		if err != nil {
			return nil, err
		}
		node.setLeft(newLeftNode)

		newNode, err := tree.rotateRight(node)
		if err != nil {
			return nil, err
		}

		return newNode, nil
	}
	if balance < -1 {
		rightNode, err := node.getRightNode(tree)
		if err != nil {
			return nil, err
		}

		rightBalance, err := rightNode.calcBalance(tree)
		if err != nil {
			return nil, err
		}
		if rightBalance <= 0 {
			// Right Right Case
			newNode, err := tree.rotateLeft(node)
			if err != nil {
				return nil, err
			}
			return newNode, nil
		}
		// Right Left Case
		node.rightNodeKey = nil
		newRightNode, err := tree.rotateRight(rightNode)
		if err != nil {
			return nil, err
		}
		node.setRight(newRightNode)

		newNode, err := tree.rotateLeft(node)
		if err != nil {
			return nil, err
		}
		return newNode, nil
	}
	// Nothing changed
	return node, nil
}

func (node *Node) calcBalance(t *MutableTree) (int, error) {
	leftNode, err := node.getLeftNode(t)
	if err != nil {
		return 0, err
	}

	rightNode, err := node.getRightNode(t)
	if err != nil {
		return 0, err
	}

	return int(leftNode.subtreeHeight) - int(rightNode.subtreeHeight), nil
}

// Rotate right and return the new node and orphan.
func (tree *MutableTree) rotateRight(node *Node) (*Node, error) {
	var err error
	// TODO: optimize balance & rotate.
	tree.addOrphan(node)
	node.reset()

	tree.addOrphan(node.left(tree))
	newNode := node.left(tree)
	newNode.reset()

	node.setLeft(newNode.right(tree))
	newNode.setRight(node)

	err = node.calcHeightAndSize(tree)
	if err != nil {
		return nil, err
	}

	err = newNode.calcHeightAndSize(tree)
	if err != nil {
		return nil, err
	}

	return newNode, nil
}

// Rotate left and return the new node and orphan.
func (tree *MutableTree) rotateLeft(node *Node) (*Node, error) {
	var err error
	// TODO: optimize balance & rotate.
	tree.addOrphan(node)
	node.reset()

	tree.addOrphan(node.right(tree))
	newNode := node.right(tree)
	newNode.reset()

	node.setRight(newNode.left(tree))
	newNode.setLeft(node)

	err = node.calcHeightAndSize(tree)
	if err != nil {
		return nil, err
	}

	err = newNode.calcHeightAndSize(tree)
	if err != nil {
		return nil, err
	}

	return newNode, nil
}

// Computes the hash of the node without computing its descendants. Must be
// called on nodes which have descendant node hashes already computed.
func (node *Node) _hash(tree *MutableTree, version int64) []byte {
	if node.hash != nil {
		return node.hash
	}

	h := sha256.New()
	if err := node.writeHashBytes2(tree, h, version); err != nil {
		return nil
	}
	node.hash = h.Sum(nil)

	return node.hash
}

// Writes the node's hash to the given io.Writer. This function expects
// child hashes to be already set.
func (node *Node) writeHashBytes(w io.Writer, version int64) error {
	err := encoding.EncodeVarint(w, int64(node.subtreeHeight))
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

	if node.isLeaf() {
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
			return fmt.Errorf("node is missing children")
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

func (node *Node) writeHashBytes2(tree *MutableTree, w io.Writer, version int64) error {
	var (
		n   int
		buf [binary.MaxVarintLen64]byte
	)

	n = binary.PutVarint(buf[:], int64(node.subtreeHeight))
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing height, %w", err)
	}
	n = binary.PutVarint(buf[:], node.size)
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing size, %w", err)
	}
	n = binary.PutVarint(buf[:], version)
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing version, %w", err)
	}

	// Key is not written for inner nodes, unlike writeBytes.

	if node.isLeaf() {
		if err := EncodeBytes(w, node.key); err != nil {
			return fmt.Errorf("writing key, %w", err)
		}

		// Indirection needed to provide proofs without values.
		// (e.g. ProofLeafNode.ValueHash)
		valueHash := sha256.Sum256(node.value)

		if err := EncodeBytes(w, valueHash[:]); err != nil {
			return fmt.Errorf("writing value, %w", err)
		}
	} else {
		if err := EncodeBytes(w, node.left(tree).hash); err != nil {
			return fmt.Errorf("writing left hash, %w", err)
		}
		if err := EncodeBytes(w, node.right(tree).hash); err != nil {
			return fmt.Errorf("writing right hash, %w", err)
		}
	}

	return nil
}

func (node *Node) isLeaf() bool {
	return node.subtreeHeight == 0
}

// EncodeBytes writes a varint length-prefixed byte slice to the writer,
// it's used for hash computation, must be compactible with the official IAVL implementation.
func EncodeBytes(w io.Writer, bz []byte) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], uint64(len(bz)))
	if _, err := w.Write(buf[0:n]); err != nil {
		return err
	}
	_, err := w.Write(bz)
	return err
}
