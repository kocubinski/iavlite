package v6

import "fmt"

// memDB approximates a database with a map.
// it used to store nodes in memory so that pool size can be constrained and tested.
type memDB struct {
	nodes       map[nodeKey]Node
	setCount    int
	deleteCount int
}

func newMemDB() *memDB {
	return &memDB{
		nodes: make(map[nodeKey]Node),
	}
}

func (db *memDB) Set(node *Node) {
	nk := *node.nodeKey
	n := *node
	n.overflow = false
	n.dirty = false
	n.leftNode = nil
	n.rightNode = nil
	n.frameId = -1
	db.nodes[nk] = n
	db.setCount++
}

func (db *memDB) Get(nk nodeKey) *Node {
	n, ok := db.nodes[nk]
	if !ok {
		return nil
	}
	return &n
}

func (db *memDB) Delete(nk nodeKey) {
	if nk.String() == "(770, 220)" {
		fmt.Println("delete (770, 220)")
	}
	delete(db.nodes, nk)
	db.deleteCount++
}
