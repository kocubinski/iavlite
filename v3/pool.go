package v3

import "bytes"

type nodePool interface {
	Get(nodeKey []byte) *Node
	SaveNode(*Node)
	DeleteNode(*Node)
}

type GhostNode []byte

func (g *GhostNode) Incorporate(pool nodePool) *Node {
	return pool.Get(*g)
}

func (node *Node) Fade() *GhostNode {
	nk := node.nodeKey.GetKey()
	return (*GhostNode)(&nk)
}

func (node *Node) Reset() {
	node.key = nil
	node.value = nil
	node.hash = nil
	node.nodeKey = nil
	node.leftNode = nil
	node.rightNode = nil
	node.rightNodeKey = nil
	node.leftNodeKey = nil
	node.subtreeHeight = 0
	node.size = 0
	node.frameId = 0
	node.leftFrameId = 0
	node.rightFrameId = 0
}

func (p *naivePool) ClonePoolNode(node *Node) *Node {
	poolNode := p.NewNode(node.nodeKey)
	poolNode.key = node.key
	poolNode.value = node.value
	poolNode.hash = node.hash
	poolNode.leftNodeKey = node.leftNodeKey
	poolNode.rightNodeKey = node.rightNodeKey
	poolNode.subtreeHeight = node.subtreeHeight
	poolNode.size = node.size

	poolNode.leftFrameId = node.leftFrameId
	poolNode.rightFrameId = node.rightFrameId
	return poolNode
}

func (node *Node) IsGhost() bool {
	return node.nodeKey == nil
}

const poolSize = 3_000_000

type naivePool struct {
	// simulates a backing database
	nodeDb map[nodeCacheKey]*Node

	freeList  []int
	nodeTable map[nodeCacheKey]int
	nodes     [poolSize]*Node
}

func newNodePool() *naivePool {
	pool := &naivePool{
		nodeDb:    make(map[nodeCacheKey]*Node),
		freeList:  make([]int, 0, poolSize),
		nodeTable: make(map[nodeCacheKey]int),
		nodes:     [poolSize]*Node{},
	}
	for i := 0; i < poolSize; i++ {
		pool.freeList = append(pool.freeList, i)
		pool.nodes[i] = &Node{}
	}
	return pool
}

func (p *naivePool) NewNode(nodeKey *NodeKey) *Node {
	if len(p.freeList) == 0 {
		panic("pool exhausted")
	}
	var id int
	id, p.freeList = p.freeList[0], p.freeList[1:]

	var nk nodeCacheKey
	copy(nk[:], nodeKey.GetKey())
	p.nodeTable[nk] = id
	n := p.nodes[id]
	n.Reset()
	n.nodeKey = nodeKey
	n.frameId = id
	return n
}

func (p *naivePool) ReturnNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	id, ok := p.nodeTable[nk]
	if !ok {
		panic("something awful; node not found in nodeTable")
	}
	p.freeList = append(p.freeList, id)
	delete(p.nodeTable, nk)
}

func (p *naivePool) Get(nodeKey []byte) *Node {
	var nk nodeCacheKey
	copy(nk[:], nodeKey)
	id, ok := p.nodeTable[nk]
	if ok {
		return p.nodes[id]
	} else {
		panic("TODO; fetch from db")
	}
}

func (p *naivePool) GetByFrameId(frameId int, nodeKey []byte) *Node {
	fn := p.nodes[frameId]
	if bytes.Compare(fn.nodeKey.GetKey(), nodeKey) != 0 {
		panic("TODO; fetch from db and push to pool")
	}
	return fn
}

func (p *naivePool) DeleteNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	delete(p.nodeDb, nk)
	p.ReturnNode(node)
}

func (p *naivePool) SaveNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	p.nodeDb[nk] = node
}
