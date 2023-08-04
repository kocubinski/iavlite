package v1

type nodePool interface {
	Get(nodeKey []byte) *Node
	SaveNode(*Node)
	DeleteNode(*Node)
}

const poolSize = 2_000_000

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
	}
	return pool
}

func (p *trivialNodePool) Get(nodeKey []byte) *Node {
	var nk nodeCacheKey
	copy(nk[:], nodeKey)
	return p.nodeDb[nk]
}

func (p *trivialNodePool) Set(nk nodeCacheKey, n *Node) {
}

func (p *trivialNodePool) DeleteNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	delete(p.nodeDb, nk)
}

func (p *trivialNodePool) SaveNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	p.nodeDb[nk] = node
}
