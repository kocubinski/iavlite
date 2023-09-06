package v6

import "github.com/kocubinski/iavlite/core"

const poolSize = 3_000_000

type nodePool struct {
	free    chan int
	nodes   [poolSize]*Node
	metrics *core.TreeMetrics
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
	np.metrics.PoolGets++
	return n
}

func (np *nodePool) Return(n *Node) {
	return
	np.free <- n.frameId
	np.metrics.PoolReturns++
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
