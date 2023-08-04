package v1

type nodePool interface {
	Get(nodeKey []byte) *Node
	SaveNode(*Node)
	DeleteNode(*Node)
}

type trivialNodePool struct {
	nodes map[nodeCacheKey]*Node
}

func (p *trivialNodePool) Get(nodeKey []byte) *Node {
	var nk nodeCacheKey
	copy(nk[:], nodeKey)
	return p.nodes[nk]
}

func (p *trivialNodePool) Set(nk nodeCacheKey, n *Node) {
	p.nodes[nk] = n
}

func (p *trivialNodePool) DeleteNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	delete(p.nodes, nk)
}

func (p *trivialNodePool) SaveNode(node *Node) {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	p.nodes[nk] = node
}
