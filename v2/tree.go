package v2

import (
	"bytes"
	"fmt"
)

type MutableTree struct {
	version int64
	root    *Node
	pool    nodePool
	orphans []*Node
}

func (tree *MutableTree) SaveVersion() ([]byte, int64, error) {
	version := tree.version + 1
	if err := tree.saveNewNodes(version); err != nil {
		return nil, 0, err
	}
	tree.version = version

	for _, orphan := range tree.orphans {
		if orphan.nodeKey == nil {
			continue
		}
		tree.pool.DeleteNode(orphan)
	}

	tree.orphans = nil

	return tree.root.hash, version, nil
}

// saveNewNodes save new created nodes by the changes of the working tree.
// NOTE: This function clears leftNode/rigthNode recursively and
// calls _hash() on the given node.
func (tree *MutableTree) saveNewNodes(version int64) error {
	nonce := uint32(0)
	newNodes := make([]*Node, 0)
	var recursiveAssignKey func(*Node) ([]byte, error)
	recursiveAssignKey = func(node *Node) ([]byte, error) {
		if node.nodeKey != nil {
			if node.nodeKey.nonce != 0 {
				return node.nodeKey.GetKey(), nil
			}
			return node.hash, nil
		}
		nonce++
		node.nodeKey = &NodeKey{
			version: version,
			nonce:   nonce,
		}
		newNodes = append(newNodes, node)

		var err error
		// the inner nodes should have two children.
		if node.subtreeHeight > 0 {
			node.leftNodeKey, err = recursiveAssignKey(node.leftNode)
			if err != nil {
				return nil, err
			}
			node.rightNodeKey, err = recursiveAssignKey(node.rightNode)
			if err != nil {
				return nil, err
			}
		}

		node._hash(version)
		return node.nodeKey.GetKey(), nil
	}

	if _, err := recursiveAssignKey(tree.root); err != nil {
		return err
	}

	for _, node := range newNodes {
		tree.pool.SaveNode(node)
		//node.leftNode, node.rightNode = nil, nil
	}

	return nil
}

func (node *Node) isLeaf() bool {
	return node.subtreeHeight == 0
}

// Set sets a key in the working tree. Nil values are invalid. The given
// key/value byte slices must not be modified after this call, since they point
// to slices stored within IAVL. It returns true when an existing value was
// updated, while false means it was a new key.
func (tree *MutableTree) Set(key, value []byte) (updated bool, err error) {
	updated, err = tree.set(key, value)
	if err != nil {
		return false, err
	}
	return updated, nil
}

// Get returns the value of the specified key if it exists, or nil otherwise.
// The returned value must not be modified, since it may point to data stored within IAVL.
func (tree *MutableTree) Get(key []byte) ([]byte, error) {
	if tree.root == nil {
		return nil, nil
	}

	return tree.Get(key)
}

func (tree *MutableTree) set(key []byte, value []byte) (updated bool, err error) {
	if value == nil {
		return updated, fmt.Errorf("attempt to store nil value at key '%s'", key)
	}

	if tree.root == nil {
		tree.root = NewNode(key, value)
		return updated, nil
	}

	tree.root, updated, err = tree.recursiveSet(tree.root, key, value)
	return updated, err
}

func (tree *MutableTree) recursiveSet(node *Node, key []byte, value []byte) (
	newSelf *Node, updated bool, err error,
) {
	if node.isLeaf() {
		switch bytes.Compare(key, node.key) {
		case -1: // setKey < leafKey
			return &Node{
				key:           node.key,
				subtreeHeight: 1,
				size:          2,
				nodeKey:       nil,
				leftNode:      NewNode(key, value),
				rightNode:     node,
			}, false, nil
		case 1: // setKey > leafKey
			return &Node{
				key:           key,
				subtreeHeight: 1,
				size:          2,
				nodeKey:       nil,
				leftNode:      node,
				rightNode:     NewNode(key, value),
			}, false, nil
		default:
			tree.addOrphan(node)
			return NewNode(key, value), true, nil
		}
	} else {
		tree.addOrphan(node)
		node, err = node.clone(tree)
		if err != nil {
			return nil, false, err
		}

		if bytes.Compare(key, node.key) < 0 {
			node.leftNode, updated, err = tree.recursiveSet(node.leftNode, key, value)
			if err != nil {
				return nil, updated, err
			}
		} else {
			node.rightNode, updated, err = tree.recursiveSet(node.rightNode, key, value)
			if err != nil {
				return nil, updated, err
			}
		}

		if updated {
			return node, updated, nil
		}
		err = node.calcHeightAndSize(tree)
		if err != nil {
			return nil, false, err
		}
		newNode, err := tree.balance(node)
		if err != nil {
			return nil, false, err
		}
		return newNode, updated, err
	}
}

// Remove removes a key from the working tree. The given key byte slice should not be modified
// after this call, since it may point to data stored inside IAVL.
func (tree *MutableTree) Remove(key []byte) ([]byte, bool, error) {
	if tree.root == nil {
		return nil, false, nil
	}
	newRoot, _, value, removed, err := tree.recursiveRemove(tree.root, key)
	if err != nil {
		return nil, false, err
	}
	if !removed {
		return nil, false, nil
	}

	tree.root = newRoot
	return value, true, nil
}

// removes the node corresponding to the passed key and balances the tree.
// It returns:
// - the hash of the new node (or nil if the node is the one removed)
// - the node that replaces the orig. node after remove
// - new leftmost leaf key for tree after successfully removing 'key' if changed.
// - the removed value
func (tree *MutableTree) recursiveRemove(node *Node, key []byte) (newSelf *Node, newKey []byte, newValue []byte, removed bool, err error) {
	tree.addOrphan(node)
	if node.isLeaf() {
		if bytes.Equal(key, node.key) {
			return nil, nil, node.value, true, nil
		}
		return node, nil, nil, false, nil
	}

	node, err = node.clone(tree)
	if err != nil {
		return nil, nil, nil, false, err
	}

	// node.key < key; we go to the left to find the key:
	if bytes.Compare(key, node.key) < 0 {
		newLeftNode, newKey, value, removed, err := tree.recursiveRemove(node.leftNode, key)
		if err != nil {
			return nil, nil, nil, false, err
		}

		if !removed {
			return node, nil, value, removed, nil
		}

		if newLeftNode == nil { // left node held value, was removed
			return node.rightNode, node.key, value, removed, nil
		}

		node.leftNode = newLeftNode
		err = node.calcHeightAndSize(tree)
		if err != nil {
			return nil, nil, nil, false, err
		}
		node, err = tree.balance(node)
		if err != nil {
			return nil, nil, nil, false, err
		}

		return node, newKey, value, removed, nil
	}
	// node.key >= key; either found or look to the right:
	newRightNode, newKey, value, removed, err := tree.recursiveRemove(node.rightNode, key)
	if err != nil {
		return nil, nil, nil, false, err
	}

	if !removed {
		return node, nil, value, removed, nil
	}

	if newRightNode == nil { // right node held value, was removed
		return node.leftNode, nil, value, removed, nil
	}

	node.rightNode = newRightNode
	if newKey != nil {
		node.key = newKey
	}
	err = node.calcHeightAndSize(tree)
	if err != nil {
		return nil, nil, nil, false, err
	}

	node, err = tree.balance(node)
	if err != nil {
		return nil, nil, nil, false, err
	}

	return node, nil, value, removed, nil
}

func (tree *MutableTree) addOrphan(node *Node) {
	tree.orphans = append(tree.orphans, node)
}
