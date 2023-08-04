package v1

type nodePool interface {
	Get(nodeCacheKey) *Node
}

type trivialNodePool struct {
	nodes map[nodeCacheKey]*Node
}

func (p *trivialNodePool) Get(nk nodeCacheKey) *Node {
	return p.nodes[nk]
}
