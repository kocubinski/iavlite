package v4

import "fmt"

type nodeCacheKey [12]byte

type memDB struct {
	db map[nodeCacheKey][]byte
}

func newMemDB() *memDB {
	return &memDB{
		db: make(map[nodeCacheKey][]byte),
	}
}

func (db *memDB) SaveNode(node *Node) error {
	var nk nodeCacheKey
	copy(nk[:], node.nodeKey.GetKey())
	db.db[nk] = node.hash
	return nil
}

func (db *memDB) GetNode(nodeKey *NodeKey) (*Node, error) {
	var nk nodeCacheKey
	copy(nk[:], nodeKey.GetKey())
	hash, ok := db.db[nk]
	if !ok {
		return nil, fmt.Errorf("node not found")
	}
	return &Node{
		nodeKey: nodeKey,
		hash:    hash,
	}, nil
}

func (db *memDB) DeleteNode(nodeKey *NodeKey) error {
	var nk nodeCacheKey
	copy(nk[:], nodeKey.GetKey())
	delete(db.db, nk)
	return nil
}
