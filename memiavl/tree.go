package memiavl

import (
	"crypto/sha256"
	"errors"
	"math"
)

var emptyHash = sha256.New().Sum(nil)

// verify change sets by replay them to rebuild iavl tree and verify the root hashes
type Tree struct {
	version uint32
	// root node of empty tree is represented as `nil`
	root Node

	initialVersion, cowVersion uint32
}

type cacheNode struct {
	key, value []byte
}

func (n *cacheNode) GetKey() []byte {
	return n.key
}

// NewEmptyTree creates an empty tree at an arbitrary version.
func NewEmptyTree(version uint64, initialVersion uint32, cacheSize int) *Tree {
	if version >= math.MaxUint32 {
		panic("version overflows uint32")
	}

	return &Tree{
		version:        uint32(version),
		initialVersion: initialVersion,
	}
}

func (t *Tree) Set(key, value []byte) (updated bool, err error) {
	if value == nil {
		// the value could be nil when replaying changes from write-ahead-log because of protobuf decoding
		value = []byte{}
	}
	t.root, updated = setRecursive(t.root, key, value, t.version+1, t.cowVersion)
	return updated, nil
}

func (t *Tree) Remove(key []byte) ([]byte, bool, error) {
	var v []byte
	v, t.root, _ = removeRecursive(t.root, key, t.version+1, t.cowVersion)
	return v, v != nil, nil
}

// saveVersion increases the version number and optionally updates the hashes
func (t *Tree) SaveVersion() ([]byte, int64, error) {
	var hash []byte
	hash = t.RootHash()

	if t.version >= uint32(math.MaxUint32) {
		return nil, 0, errors.New("version overflows uint32")
	}
	t.version++

	// to be compatible with existing golang iavl implementation.
	// see: https://github.com/cosmos/iavl/pull/660
	if t.version == 1 && t.initialVersion > 0 {
		t.version = t.initialVersion
	}

	return hash, int64(t.version), nil
}

// Version returns the current tree version
func (t *Tree) Version() int64 {
	return int64(t.version)
}

// RootHash updates the hashes and return the current root hash
func (t *Tree) RootHash() []byte {
	if t.root == nil {
		return emptyHash
	}
	return t.root.Hash()
}

func (t *Tree) GetWithIndex(key []byte) (int64, []byte) {
	if t.root == nil {
		return 0, nil
	}

	value, index := t.root.Get(key)

	return int64(index), value
}

func (t *Tree) GetByIndex(index int64) ([]byte, []byte) {
	if index > math.MaxUint32 {
		return nil, nil
	}
	if t.root == nil {
		return nil, nil
	}

	key, value := t.root.GetByIndex(uint32(index))

	return key, value
}

func (t *Tree) Get(key []byte) []byte {
	_, value := t.GetWithIndex(key)
	if value == nil {
		return nil
	}

	return value
}

func (t *Tree) Has(key []byte) bool {
	return t.Get(key) != nil
}

// ScanPostOrder scans the tree in post-order, and call the callback function on each node.
// If the callback function returns false, the scan will be stopped.
func (t *Tree) ScanPostOrder(callback func(node Node) bool) {
	if t.root == nil {
		return
	}

	stack := []*stackEntry{{node: t.root}}

	for len(stack) > 0 {
		entry := stack[len(stack)-1]

		if entry.node.IsLeaf() || entry.expanded {
			callback(entry.node)
			stack = stack[:len(stack)-1]
			continue
		}

		entry.expanded = true
		stack = append(stack, &stackEntry{node: entry.node.Right()})
		stack = append(stack, &stackEntry{node: entry.node.Left()})
	}
}

type stackEntry struct {
	node     Node
	expanded bool
}
