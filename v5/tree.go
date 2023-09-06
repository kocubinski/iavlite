package v4

import (
	"bytes"
	"fmt"
)

type MutableTree struct {
	version int64
	root    *Node
	db      *memDB
	orphans []*NodeKey
	pool    *nodePool
}

func (tree *MutableTree) SaveVersion() ([]byte, int64, error) {
	tree.version++
	//if err := tree.saveNewNodes(); err != nil {
	//	tree.version--
	//	return nil, 0, err
	//}

	tree.deepHash(0, tree.root)
	tree.orphans = nil

	return tree.root.hash, tree.version, nil
}

func (tree *MutableTree) deepHash(sequence uint32, node *Node) (hash []byte) {
	if node.hash != nil {
		return node.hash
	}
	sequence++
	node.nodeKey = &NodeKey{
		version: tree.version,
		nonce:   sequence,
	}
	if node.subtreeHeight > 0 {
		// wrong, should be nodekey assignment, but just profiling for now.
		node.leftNodeKey = tree.deepHash(sequence, node.leftNode)
		node.rightNodeKey = tree.deepHash(sequence, node.rightNode)
	}
	node._hash(tree.version)
	return node.hash
}

// saveNewNodes save new created nodes by the changes of the working tree.
// NOTE: This function clears leftNode/rigthNode recursively and
// calls _hash() on the given node.
func (tree *MutableTree) saveNewNodes() error {
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
			version: tree.version,
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

		node._hash(tree.version)
		return node.nodeKey.GetKey(), nil
	}

	if _, err := recursiveAssignKey(tree.root); err != nil {
		return err
	}

	//for _, node := range newNodes {
	//	if err := tree.db.SaveNode(node); err != nil {
	//		return err
	//	}
	// node.leftNode, node.rightNode = nil, nil
	//}
	//for _, orphan := range tree.orphans {
	//	if err := tree.db.DeleteNode(orphan); err != nil {
	//		return err
	//	}
	//}

	tree.orphans = nil

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
		rootNode := tree.pool.Get()
		rootNode.key = key
		rootNode.value = value
		tree.root = rootNode
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
			leftNode := tree.pool.Get()
			leftNode.key = key
			leftNode.value = value

			n := tree.pool.Get()
			n.key = node.key
			n.subtreeHeight = 1
			n.size = 2
			n.leftNode = leftNode
			n.rightNode = node
			return n, false, nil
		case 1: // setKey > leafKey
			rightNode := tree.pool.Get()
			rightNode.key = key
			rightNode.value = value

			n := tree.pool.Get()
			n.key = node.key
			n.subtreeHeight = 1
			n.size = 2
			n.leftNode = node
			n.rightNode = rightNode
			return n, false, nil
		default:
			node.hash = nil
			node.nodeKey = nil
			node.value = value
			return node, true, nil
		}
	} else {
		tree.addOrphan(node)
		node.reset()

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
	if node.isLeaf() {
		if bytes.Equal(key, node.key) {
			tree.addOrphan(node)
			tree.pool.Return(node)
			return nil, nil, node.value, true, nil
		}
		return node, nil, nil, false, nil
	}

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

		tree.addOrphan(node)

		if newLeftNode == nil { // left node held value, was removed
			return node.rightNode, node.key, value, removed, nil
		}

		node.hash = nil
		node.nodeKey = nil

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

	tree.addOrphan(node)

	if newRightNode == nil { // right node held value, was removed
		return node.leftNode, nil, value, removed, nil
	}

	node.hash = nil
	node.nodeKey = nil

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

func (tree *MutableTree) Size() int64 {
	return tree.root.size
}

func (tree *MutableTree) Height() int8 {
	return tree.root.subtreeHeight
}

func (tree *MutableTree) addOrphan(n *Node) {
	//if n.hash != nil {
	//	tree.orphans = append(tree.orphans, n.nodeKey)
	//}
}
