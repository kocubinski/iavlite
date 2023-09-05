package v0

import (
	"bytes"
	"fmt"
)

type Tree struct {
	version int64
	root    *Node
	db      *db

	sequence uint32
	added    []*DbNode
	deleted  []*DbNode
}

func NewTree(db *db) *Tree {
	return &Tree{
		db: db,
	}
}

func (nk *NodeKey) String() string {
	return fmt.Sprintf("(%d, %d)", nk.version, nk.sequence)
}

func (t *Tree) Get(key []byte) (value []byte, err error) {
	return []byte{}, nil
}

func (t *Tree) Set(key []byte, value []byte) (updated bool, err error) {
	if t.root == nil {
		t.root = t.NewNode(key, value)
		return false, nil
	}
	t.root, updated = t.set(t.root, key, value)
	return updated, nil
}

func (t *Tree) set(node *Node, key []byte, value []byte) (*Node, bool) {
	if node.IsLeaf() {
		switch bytes.Compare(key, node.key) {
		case 0: // setKey == leafKey
			n := t.FadeNode(node)
			n.value = value
			return n, true
		case -1: // setKey < leafKey
			n := t.NewNode(key, nil)
			n.height = 1
			n.size = 2
			n.leftNode = t.NewNode(key, value)
			n.rightNode = node
			return n, false
		case 1: // setKey > leafKey
			n := t.NewNode(key, nil)
			n.height = 1
			n.size = 2
			n.leftNode = node
			n.rightNode = t.NewNode(key, value)
			return n, false
		}
	}

	newNode := t.FadeNode(node)
	var updated bool
	if bytes.Compare(key, node.key) == -1 {
		newNode.leftNode, updated = t.set(newNode.left(t.db), key, value)
	} else {
		newNode.rightNode, updated = t.set(newNode.right(t.db), key, value)
	}

	if !updated {
		newNode.calcHeightAndSize()
		newNode = t.rebalance(newNode)
	}

	return newNode, updated
}

func (t *Tree) Remove(key []byte) (value []byte, deleted bool, err error) {
	value, t.root = t.remove(t.root, key)
	return value, true, nil
}

func (t *Tree) remove(node *Node, key []byte) (value []byte, updated *Node) {
	if node == nil {
		return nil, nil
	}

	if node.IsLeaf() {
		if bytes.Equal(node.key, key) {
			return node.value, nil
		}
		return nil, node
	}

	if bytes.Compare(key, node.key) == -1 {
		value, newLeft := t.remove(node.left(t.db), key)
		if value == nil {
			return nil, node
		}
		if newLeft == nil {
			return value, node.right(t.db)
		}
		newNode := t.FadeNode(node)
		newNode.leftNode = newLeft
		newNode.calcHeightAndSize()
		return value, t.rebalance(newNode)
	}

	value, newRight := t.remove(node.right(t.db), key)
	if value == nil {
		return nil, node
	}
	if newRight == nil {
		return value, node.left(t.db)
	}

	newNode := t.FadeNode(node)
	newNode.rightNode = newRight
	newNode.calcHeightAndSize()
	return value, t.rebalance(newNode)
}

func (t *Tree) SaveVersion() (rootHash []byte, version int64, err error) {
	rootHash, _, err = t.deepHash(t.root)
	if err != nil {
		return nil, 0, err
	}
	if t.version%1000 == 0 {
		fmt.Printf("version %d; hashIter: %d; hashCacl: %d\n", t.version, deepHashIter, deepHashCount)
	}

	deepHashIter = 0
	deepHashCount = 0

	t.version++
	t.sequence = 1
	t.added = nil
	t.deleted = nil
	return rootHash, t.version, nil
}

func (t *Tree) pushChanged(old *Node, new *Node) {
	t.added = append(t.added, &DbNode{new.NodeData, new.nodeKey})
	// a node without a hash is node which has been added and deleted in the same block
	// and therefore does not need to be persisted
	if old.hash != nil {
		t.deleted = append(t.deleted, &DbNode{NodeKey: old.nodeKey})
	}
}

func (t *Tree) pushAdded(node *Node) {
	t.added = append(t.added, &DbNode{node.NodeData, node.nodeKey})
}

func (t *Tree) rebalance(n *Node) *Node {
	balance := n.calcBalance()
	leftNode := n.left(t.db)
	rightNode := n.right(t.db)

	switch {
	case balance > 1: // left heavy
		leftBalance := leftNode.calcBalance()
		if leftBalance >= 0 {
			return t.rotateRight(n)
		} else {
			newLeft := t.FadeNode(leftNode)
			n.leftNode = t.rotateLeft(newLeft)
			return t.rotateRight(n)
		}
	case balance < -1: // right heavy
		rightBalance := rightNode.calcBalance()
		if rightBalance <= 0 {
			return t.rotateLeft(n)
		} else {
			newRight := t.FadeNode(rightNode)
			n.rightNode = t.rotateRight(newRight)
			return t.rotateLeft(n)
		}
	default: // balanced
		return n
	}
}

func (t *Tree) rotateLeft(n *Node) *Node {
	rightNode := n.right(t.db)
	node := t.FadeNode(rightNode)
	n.rightNode = rightNode.left(t.db)
	node.leftNode = n
	n.calcHeightAndSize()
	node.calcHeightAndSize()
	return node
}

func (t *Tree) rotateRight(n *Node) *Node {
	leftNode := n.left(t.db)
	node := t.FadeNode(leftNode)
	n.leftNode = leftNode.right(t.db)
	node.rightNode = n
	n.calcHeightAndSize()
	node.calcHeightAndSize()
	return node
}

var deepHashCount int
var deepHashIter int

func (t *Tree) deepHash(node *Node) (hash []byte, nodeKey *NodeKey, err error) {
	deepHashIter++
	if node.nodeKey == nil {
		return nil, nil, fmt.Errorf("node key is nil")
	}

	if node.hash != nil {
		return node.hash, node.nodeKey, nil
	}

	// TODO may need to fetch left/right from db after evictions are in place
	if node.height > 0 {
		_, node.leftNodeKey, err = t.deepHash(node.leftNode)
		if err != nil {
			return nil, nil, err
		}
		_, node.rightNodeKey, err = t.deepHash(node.rightNode)
		if err != nil {
			return nil, nil, err
		}
	}
	hash, err = node.computeHash(t.version)
	if err != nil {
		return nil, nil, err
	}
	deepHashCount++
	return hash, node.nodeKey, nil
}

func (t *Tree) NextNodeKey() *NodeKey {
	nk := &NodeKey{
		version:  t.version,
		sequence: t.sequence,
	}
	t.sequence++
	return nk
}

func (t *Tree) NewNode(key []byte, value []byte) *Node {
	n := &Node{
		NodeData: &NodeData{
			key:   key,
			value: value,
			size:  1,
		},
		nodeKey: t.NextNodeKey(),
	}
	t.pushAdded(n)
	return n
}

func (t *Tree) FadeNode(node *Node) *Node {
	n := &Node{
		NodeData:  node.NodeData,
		nodeKey:   t.NextNodeKey(),
		leftNode:  node.leftNode,
		rightNode: node.rightNode,
	}
	n.hash = nil
	t.pushChanged(node, n)
	return n
}

func (t *Tree) Size() int64 {
	return t.root.size
}

func (t *Tree) Height() int8 {
	return t.root.height
}
