package v3

import "bytes"

type poolNode struct {
	*Node
	pinned bool
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
	n := p.NewNode(node.nodeKey)
	n.key = node.key
	n.value = node.value
	n.hash = node.hash
	n.leftNodeKey = node.leftNodeKey
	n.rightNodeKey = node.rightNodeKey
	n.subtreeHeight = node.subtreeHeight
	n.size = node.size

	n.leftFrameId = node.leftFrameId
	n.rightFrameId = node.rightFrameId
	return n
}

const poolSize = 3_000_000

type naivePool struct {
	// simulates a backing database
	nodeDb map[nodeCacheKey]*Node

	freeList  chan int
	nodeTable map[nodeCacheKey]int
	nodes     [poolSize]*Node
}

func newNodePool() *naivePool {
	pool := &naivePool{
		nodeDb:    make(map[nodeCacheKey]*Node),
		freeList:  make(chan int, poolSize),
		nodeTable: make(map[nodeCacheKey]int),
		nodes:     [poolSize]*Node{},
	}
	for i := 0; i < poolSize; i++ {
		pool.freeList <- i
		pool.nodes[i] = &Node{}
	}
	return pool
}

func (p *naivePool) NewNode(nodeKey *NodeKey) *Node {
	if len(p.freeList) == 0 {
		panic("pool exhausted")
	}
	id := <-p.freeList

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
	p.freeList <- id
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
