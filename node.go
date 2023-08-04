package iavlite

import "fmt"

type NodeData struct {
	key          []byte
	value        []byte
	hash         []byte
	leftNodeKey  []byte
	rightNodeKey []byte
	size         int64
	height       int8
}

func (pn *NodeData) IsLeaf() bool {
	return pn.height == 0
}

type Node struct {
	NodeData
	nodeKey   NodeKey
	leftNode  *Node
	rightNode *Node
	persisted bool
}

type NodeKey struct {
	version  int64
	sequence uint32
}

func (n *Node) left(db *db) *Node {
	if n.leftNode == nil {
		return n.leftNode
	}
	left, err := db.Get(n.leftNodeKey)
	if err != nil {
		panic(fmt.Sprintf("failed to get left node: %v", err))
	}
	return left
}

func (n *Node) right(db *db) *Node {
	if n.rightNode == nil {
		return n.rightNode
	}
	right, err := db.Get(n.rightNodeKey)
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

func maxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}
