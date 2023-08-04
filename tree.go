package iavlite

import "bytes"

type Tree struct {
	version int64
	root    *Node
	db      *db

	sequence uint32
	added    []*NodeData
	deleted  []*NodeData
}

func NewTree(db *db) *Tree {
	return &Tree{
		db: db,
	}
}

func (t *Tree) Get(key []byte) (value []byte, err error) {
	return []byte{}, nil
}

func (t *Tree) Set(key []byte, value []byte) (err error) {
	t.root, _ = t.set(t.root, key, value)
	return nil
}

func (t *Tree) set(node *Node, key []byte, value []byte) (*Node, bool) {
	if node == nil {
		return t.NewNode(key, value), false
	}
	if node.IsLeaf() {
		switch bytes.Compare(key, node.key) {
		case 0: // setKey == leafKey
			n := t.NewNode(key, value)
			t.addChanged(node, n)
			return n, true
		case -1: // setKey < leafKey
			n := &Node{
				NodeData: NodeData{
					key:    node.key,
					height: 1,
					size:   2,
				},
				leftNode:  t.NewNode(key, value),
				rightNode: node,
			}
			return n, false
		case 1: // setKey > leafKey
			n := &Node{
				NodeData: NodeData{
					key:    key,
					height: 1,
					size:   2,
				},
				leftNode:  node,
				rightNode: t.NewNode(key, value),
			}
			return n, false
		}
	}

	newNode := t.FadeNode(node)
	var updated bool
	if bytes.Compare(key, node.key) == -1 {
		node.leftNode, updated = t.set(node.left(t.db), key, value)
	} else {
		node.rightNode, updated = t.set(node.right(t.db), key, value)
	}

	if !updated {
		newNode.calcHeightAndSize()
		newNode = t.rebalance(node)
	}

	return newNode, updated
}

func (t *Tree) Remove(key []byte) (value []byte) {
	value, t.root = t.remove(t.root, key)
	return value
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

func (t *Tree) Commit() (version int64, err error) {
	return 0, nil
}

func (t *Tree) addChanged(old *Node, new *Node) {
	t.added = append(t.added, &new.NodeData)
	// a node without a hash is node which has been added and deleted in the same block
	// and therefore does not need to be persisted
	if old.hash != nil {
		t.deleted = append(t.deleted, &old.NodeData)
	}
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

func (t *Tree) deepHash(node *Node) []byte {
	if node.hash != nil {
		return node.hash
	}
	// TODO may need to fetch left/right from db after evictions are in place
	leftHash := t.deepHash(node.leftNode)
	rightHash := t.deepHash(node.rightNode)
	node.hash = t.hasher.Hash(leftHash, rightHash, node.key, node.value)
	return node.hash
}

func (t *Tree) NewNode(key []byte, value []byte) *Node {
	n := &Node{
		NodeData: NodeData{
			key:   key,
			value: value,
			size:  1,
		},
		nodeKey: NodeKey{
			version:  t.version,
			sequence: t.sequence,
		},
	}
	t.sequence++
	return n
}

func (t *Tree) FadeNode(node *Node) *Node {
	n := &Node{
		NodeData: node.NodeData,
		nodeKey: NodeKey{
			version:  t.version,
			sequence: t.sequence,
		},
		leftNode:  node.leftNode,
		rightNode: node.rightNode,
	}
	t.sequence++
	t.addChanged(node, n)
	return n
}
