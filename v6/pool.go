package v6

const poolSize = 3_000_000

type nodePool struct {
	free  chan int
	nodes [poolSize]*Node
}

func newNodePool() *nodePool {
	np := &nodePool{
		nodes: [poolSize]*Node{},
		free:  make(chan int, poolSize),
	}
	for i := 0; i < poolSize; i++ {
		np.free <- i
		np.nodes[i] = &Node{}
	}
	return np
}

func (np *nodePool) Get() *Node {
	if len(np.free) == 0 {
		panic("pool exhausted")
	}
	id := <-np.free
	n := np.nodes[id]
	n.frameId = id
	return n
}

func (np *nodePool) Return(n *Node) {
	np.free <- n.frameId
	n.clear()
}

func (node *Node) clear() {
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
}
