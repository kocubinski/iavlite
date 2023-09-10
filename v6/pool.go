package v6

import (
	"github.com/kocubinski/iavlite/core"
)

type nodePool struct {
	db        *memDB
	free      chan int
	nodes     []*Node
	metrics   *core.TreeMetrics
	evict     func(*nodePool) *Node
	clockHand int
}

func (np *nodePool) clockEvict() *Node {
	itr := 0
	for {
		itr++
		if itr > len(np.nodes)*2 {
			panic("eviction failed, pool exhausted")
		}

		n := np.nodes[np.clockHand]
		np.clockHand++
		if np.clockHand == len(np.nodes) {
			np.clockHand = 0
		}

		switch {
		case n.use:
			// always clear the use bit, dirty nodes included.
			np.metrics.PoolEvictMiss++
			n.use = false
			continue
		case n.dirty:
			// never evict dirty nodes
			np.metrics.PoolEvictMiss++
			// TODO async write and atomic bool
			np.db.Set(n)
			continue
		default:
			np.metrics.PoolEvict++
			np.Return(n)
			return n
		}
	}
}

func newNodePool(db *memDB, size int) *nodePool {
	np := &nodePool{
		nodes: make([]*Node, size),
		free:  make(chan int, size),
		db:    db,
	}
	for i := 0; i < size; i++ {
		np.free <- i
		np.nodes[i] = &Node{}
	}
	return np
}

func (np *nodePool) Get() *Node {
	var n *Node
	if len(np.free) == 0 {
		n = np.clockEvict()
	} else {
		id := <-np.free
		n = np.nodes[id]
		n.frameId = id
	}
	n.use = true
	n.dirty = true

	np.metrics.PoolGet++
	return n
}

func (np *nodePool) Return(n *Node) {
	np.free <- n.frameId
	np.metrics.PoolReturn++
	n.clear()
}

func (np *nodePool) Put(n *Node) {
	np.metrics.PoolFault++
	var frameId int
	if len(np.free) == 0 {
		frameId = np.clockEvict().frameId
	} else {
		frameId = <-np.free
	}
	// replace node in page cache. presumably n was fetched and unmarshalled from an external source.
	// an optimization may be unmarshalling directly into the page cache.
	np.nodes[frameId] = n
	if n == nil {
		panic("nodePool.Put() with nil node")
	}
	n.frameId = frameId
	n.use = true
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
	node.use = false
	node.dirty = false
}
