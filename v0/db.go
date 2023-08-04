package v0

import "fmt"

type db struct {
}

func (db *db) Get(nodeKey []byte) (*Node, error) {
	return nil, fmt.Errorf("node not found")
}
