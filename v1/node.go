package legacy

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	encoding "github.com/kocubinski/iavlite/internal"
)

const (
	// ModeLegacyLeftNode is the mode for legacy left child in the node encoding/decoding.
	ModeLegacyLeftNode = 0x01
	// ModeLegacyRightNode is the mode for legacy right child in the node encoding/decoding.
	ModeLegacyRightNode = 0x02
)

// NodeKey represents a key of node in the DB.
type NodeKey struct {
	version int64
	nonce   uint32
}

// GetKey returns a byte slice of the NodeKey.
func (nk *NodeKey) GetKey() []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint64(b, uint64(nk.version))
	binary.BigEndian.PutUint32(b[8:], nk.nonce)
	return b
}

// GetNodeKey returns a NodeKey from a byte slice.
func GetNodeKey(key []byte) *NodeKey {
	return &NodeKey{
		version: int64(binary.BigEndian.Uint64(key)),
		nonce:   binary.BigEndian.Uint32(key[8:]),
	}
}

// GetRootKey returns a byte slice of the root node key for the given version.
func GetRootKey(version int64) []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint64(b, uint64(version))
	binary.BigEndian.PutUint32(b[8:], 1)
	return b
}

// Node represents a node in a Tree.
type Node struct {
	key           []byte
	value         []byte
	hash          []byte
	nodeKey       *NodeKey
	leftNodeKey   []byte
	rightNodeKey  []byte
	size          int64
	leftNode      *Node
	rightNode     *Node
	subtreeHeight int8
}

// String returns a string representation of the node key.
func (nk *NodeKey) String() string {
	return fmt.Sprintf("(%d, %d)", nk.version, nk.nonce)
}

// NewNode returns a new node from a key, value and version.
func NewNode(key []byte, value []byte) *Node {
	return &Node{
		key:           key,
		value:         value,
		subtreeHeight: 0,
		size:          1,
	}
}

// clone creates a shallow copy of a node with its hash set to nil.
func (node *Node) clone(tree *MutableTree) (*Node, error) {
	if node.isLeaf() {
		return nil, fmt.Errorf("cannot clone leaf node")
	}

	// ensure get children
	var err error
	leftNode := node.leftNode
	rightNode := node.rightNode
	if node.nodeKey != nil {
		leftNode, err = node.getLeftNode(tree)
		if err != nil {
			return nil, err
		}
		rightNode, err = node.getRightNode(tree)
		if err != nil {
			return nil, err
		}
		//node.leftNode = nil
		//node.rightNode = nil
	}

	return &Node{
		key:           node.key,
		subtreeHeight: node.subtreeHeight,
		size:          node.size,
		hash:          nil,
		nodeKey:       nil,
		leftNodeKey:   node.leftNodeKey,
		rightNodeKey:  node.rightNodeKey,
		leftNode:      leftNode,
		rightNode:     rightNode,
	}, nil
}

func (node *Node) getLeftNode(t *MutableTree) (*Node, error) {
	if node.leftNode != nil {
		return node.leftNode, nil
	}
	return nil, fmt.Errorf("node not found")
	//leftNode, err := t.ndb.GetNode(node.leftNodeKey)
	//if err != nil {
	//	return nil, err
	//}
	//return leftNode, nil
}

func (node *Node) getRightNode(t *MutableTree) (*Node, error) {
	if node.rightNode != nil {
		return node.rightNode, nil
	}
	return nil, fmt.Errorf("node not found")
	//rightNode, err := t.ndb.GetNode(node.rightNodeKey)
	//if err != nil {
	//	return nil, err
	//}
	//return rightNode, nil
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
		lftBalance, err := node.leftNode.calcBalance(tree)
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
		node.leftNode, err = tree.rotateLeft(node.leftNode)
		if err != nil {
			return nil, err
		}

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
		node.rightNode, err = tree.rotateRight(rightNode)
		if err != nil {
			return nil, err
		}
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
	node, err = node.clone(tree)
	if err != nil {
		return nil, err
	}

	newNode, err := node.leftNode.clone(tree)
	if err != nil {
		return nil, err
	}

	node.leftNode = newNode.rightNode
	newNode.rightNode = node

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
	node, err = node.clone(tree)
	if err != nil {
		return nil, err
	}

	newNode, err := node.rightNode.clone(tree)
	if err != nil {
		return nil, err
	}

	node.rightNode = newNode.leftNode
	newNode.leftNode = node

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
func (node *Node) _hash(version int64) []byte {
	if node.hash != nil {
		return node.hash
	}

	h := sha256.New()
	if err := node.writeHashBytes(h, version); err != nil {
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
