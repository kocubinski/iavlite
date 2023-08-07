package v3

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
}

func (p *trivialNodePool) ClonePoolNode(node *Node) *Node {
	poolNode := p.NewNode(node.nodeKey)
	poolNode.key = node.key
	poolNode.value = node.value
	poolNode.hash = node.hash
	poolNode.leftNodeKey = node.leftNodeKey
	poolNode.rightNodeKey = node.rightNodeKey
	poolNode.subtreeHeight = node.subtreeHeight
	poolNode.size = node.size
	return poolNode
}

func (node *Node) IsGhost() bool {
	return node.nodeKey == nil
}

const poolSize = 3_000_000

type trivialNodePool struct {
	// simulates a backing database
	nodeDb map[nodeCacheKey]*Node

	freeList  []int
	nodeTable map[nodeCacheKey]int
	nodes     [poolSize]*Node
}

func newNodePool() *trivialNodePool {
	pool := &trivialNodePool{
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

func (p *trivialNodePool) NewNode(nodeKey *NodeKey) *Node {
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
	return n
}

func (p *trivialNodePool) ReturnNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	id, ok := p.nodeTable[nk]
	if !ok {
		panic("something awful; node not found in nodeTable")
	}
	p.freeList = append(p.freeList, id)
	delete(p.nodeTable, nk)
}

func (p *trivialNodePool) Get(nodeKey []byte) *Node {
	var nk nodeCacheKey
	copy(nk[:], nodeKey)
	id, ok := p.nodeTable[nk]
	if ok {
		return p.nodes[id]
	} else {
		panic("TODO; fetch from db")
	}
}

func (p *trivialNodePool) DeleteNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	delete(p.nodeDb, nk)
	p.ReturnNode(node)
}

func (p *trivialNodePool) SaveNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	p.nodeDb[nk] = node
}
