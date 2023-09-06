package v6

type nodeCacheKey [12]byte

// memDB approximates a database with a map.
// it used to store nodes in memory so that pool size can be constrained and tested.
type memDB struct {
	nodes map[nodeCacheKey]*Node
}

func newMemDB() *memDB {
	return &memDB{
		nodes: make(map[nodeCacheKey]*Node),
	}
}

//func (db *memDB) Set(node *Node) {
//	db.nodes[nodeCacheKey(node.nodeKey.GetKey())] = node
//}
//
//func (db *memDB) Get(key []byte) *Node {
//	return db.nodes[nodeCacheKey(key)]
//}
//
//func (db *memDB) Delete(key []byte) {
//	delete(db.nodes, nodeCacheKey(key))
//}
