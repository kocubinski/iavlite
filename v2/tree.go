package v2

import (
	"bytes"
	"fmt"
)

type MutableTree struct {
	version  int64
	sequence uint32

	root     *GhostNode
	pool     *trivialNodePool
	orphans  []*Node
	newNodes []*Node
}

func NewMutableTree() *MutableTree {
	return &MutableTree{
		pool:     newNodePool(),
		sequence: 1,
		version:  1,
	}
}

func (tree *MutableTree) NextNodeKey() *NodeKey {
	nk := &NodeKey{
		version: tree.version,
		nonce:   tree.sequence,
	}
	tree.sequence++
	return nk
}

func (tree *MutableTree) SaveVersion() ([]byte, int64, error) {
	version := tree.version
	if err := tree.saveNewNodes(version); err != nil {
		return nil, 0, err
	}

	for _, orphan := range tree.orphans {
		if orphan.nodeKey == nil {
			continue
		}
		tree.pool.DeleteNode(orphan)
	}

	tree.orphans = nil
	tree.sequence = 1
	tree.version++

	return tree.root.Incorporate(tree.pool).hash, version, nil
}

// saveNewNodes save new created nodes by the changes of the working tree.
// NOTE: This function clears leftNode/rigthNode recursively and
// calls _hash() on the given node.
func (tree *MutableTree) saveNewNodes(version int64) error {
	newNodes := make([]*Node, 0)
	var deepHash func(*Node) ([]byte, error)
	deepHash = func(node *Node) ([]byte, error) {
		if node.hash != nil {
			return node.nodeKey.GetKey(), nil
		}
		newNodes = append(newNodes, node)

		var err error
		// the inner nodes should have two children.
		if node.subtreeHeight > 0 {
			node.leftNodeKey, err = deepHash(node.leftNode)
			if err != nil {
				return nil, err
			}
			node.rightNodeKey, err = deepHash(node.rightNode)
			if err != nil {
				return nil, err
			}
		}

		node._hash(version)
		return node.nodeKey.GetKey(), nil
	}

	rootNode := tree.root.Incorporate(tree.pool)
	if _, err := deepHash(rootNode); err != nil {
		return err
	}

	for _, node := range newNodes {
		tree.pool.SaveNode(node)
		node.leftNode, node.rightNode = nil, nil
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
	panic("implement me")
}

func (tree *MutableTree) set(key []byte, value []byte) (updated bool, err error) {
	if value == nil {
		return updated, fmt.Errorf("attempt to store nil value at key '%s'", key)
	}

	var rootNode *Node
	if tree.root == nil {
		rootNode = tree.NewPoolNode()
		tree.root = rootNode.Fade()
		return updated, nil
	}
	rootNode = tree.root.Incorporate(tree.pool)
	rootNode, updated, err = tree.recursiveSet(rootNode, key, value)
	if err != nil {
		return false, err
	}
	tree.root = rootNode.Fade()

	return updated, err
}

func (tree *MutableTree) recursiveSet(node *Node, key []byte, value []byte) (
	newSelf *Node, updated bool, err error,
) {
	if node.isLeaf() {
		switch bytes.Compare(key, node.key) {
		case -1: // setKey < leafKey
			leftNode := tree.NewPoolNode()
			leftNode.key = key
			leftNode.value = value
			parent := tree.NewPoolNode()
			parent.key = node.key
			parent.subtreeHeight = 1
			parent.size = 2
			parent.leftNode = leftNode
			parent.rightNode = node
			return parent, false, nil
		case 1: // setKey > leafKey
			rightNode := tree.NewPoolNode()
			rightNode.key = key
			rightNode.value = value
			parent := tree.NewPoolNode()
			parent.key = key
			parent.subtreeHeight = 1
			parent.size = 2
			parent.leftNode = node
			parent.rightNode = rightNode
			return parent, false, nil
		default:
			tree.addOrphan(node)
			newNode := tree.NewPoolNode()
			newNode.key = key
			newNode.value = value
			return newNode, true, nil
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
	rootNode := tree.root.Incorporate(tree.pool)
	newRoot, _, value, removed, err := tree.recursiveRemove(rootNode, key)
	if err != nil {
		return nil, false, err
	}
	if !removed {
		return nil, false, nil
	}

	tree.root = newRoot.Fade()
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

func (tree *MutableTree) NewPoolNode() *Node {
	return tree.pool.NewNode(tree.NextNodeKey())
}
func (tree *MutableTree) Size() int64 {
	root := tree.root.Incorporate(tree.pool)
	return root.size
}

func (tree *MutableTree) Height() int8 {
	root := tree.root.Incorporate(tree.pool)
	return root.subtreeHeight
}
