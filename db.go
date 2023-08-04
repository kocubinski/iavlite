package iavlite

type db struct {
}

func (db *db) Get(nodeKey []byte) (*Node, error) {
	return &Node{}, nil
}
