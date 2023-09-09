package v6

import "github.com/kocubinski/iavlite/core"

type nodePool struct {
	free    chan int
	nodes   []*Node
	metrics *core.TreeMetrics
	//hotSet  []*Node
	//coldSet []*Node
}

func newNodePool(size int) *nodePool {
	np := &nodePool{
		nodes: make([]*Node, size),
		free:  make(chan int, size),
	}
	for i := 0; i < size; i++ {
		np.free <- i
		np.nodes[i] = &Node{}
	}
	return np
}

func (np *nodePool) HotGet() *Node {
	if len(np.free) == 0 {
		// TODO eviction
		panic("pool exhausted")
	}
	id := <-np.free
	n := np.nodes[id]
	n.frameId = id
	np.metrics.PoolGets++
	//np.hotSet = append(np.hotSet, n)
	return n
}

func (np *nodePool) Return(n *Node) {
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
