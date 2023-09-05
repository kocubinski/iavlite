package v5

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	api "github.com/kocubinski/costor-api"
	"github.com/kocubinski/costor-api/compact"
	"github.com/kocubinski/iavlite/testutil"
	"golang.org/x/exp/slices"
)

type Node struct {
	key       []byte
	value     []byte
	hash      []byte
	size      int64
	leftNode  *Node
	rightNode *Node
	height    int8
}

func (node *Node) isLeaf() bool {
	return node.height == 0
}

// Computes the hash of the node without computing its descendants. Must be
// called on nodes which have descendant node hashes already computed.
func (node *Node) _hash(version int64) []byte {
	if node.hash != nil {
		return node.hash
	}

	h := sha256.New()
	if err := node.writeHashBytes2(h, version); err != nil {
		return nil
	}
	node.hash = h.Sum(nil)

	return node.hash
}

// EncodeBytes writes a varint length-prefixed byte slice to the writer,
// it's used for hash computation, must be compactible with the official IAVL implementation.
func EncodeBytes(w io.Writer, bz []byte) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], uint64(len(bz)))
	if _, err := w.Write(buf[0:n]); err != nil {
		return err
	}
	_, err := w.Write(bz)
	return err
}

func (node *Node) writeHashBytes2(w io.Writer, version int64) error {
	var (
		n   int
		buf [binary.MaxVarintLen64]byte
	)

	n = binary.PutVarint(buf[:], int64(node.height))
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing height, %w", err)
	}
	n = binary.PutVarint(buf[:], node.size)
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing size, %w", err)
	}
	n = binary.PutVarint(buf[:], version)
	if _, err := w.Write(buf[0:n]); err != nil {
		return fmt.Errorf("writing version, %w", err)
	}

	// Key is not written for inner nodes, unlike writeBytes.

	if node.isLeaf() {
		if err := EncodeBytes(w, node.key); err != nil {
			return fmt.Errorf("writing key, %w", err)
		}

		// Indirection needed to provide proofs without values.
		// (e.g. ProofLeafNode.ValueHash)
		valueHash := sha256.Sum256(node.value)

		if err := EncodeBytes(w, valueHash[:]); err != nil {
			return fmt.Errorf("writing value, %w", err)
		}
	} else {
		if err := EncodeBytes(w, node.leftNode.hash); err != nil {
			return fmt.Errorf("writing left hash, %w", err)
		}
		if err := EncodeBytes(w, node.rightNode.hash); err != nil {
			return fmt.Errorf("writing right hash, %w", err)
		}
	}

	return nil
}

func RebuildTree() (*Node, error) {
	stream := &compact.StreamingContext{}
	opts := testutil.NewTreeBuildOptions(nil).With300_000()
	itr, err := stream.NewIterator(opts.ChangelogDir)

	if err != nil {
		return nil, err
	}

	state := map[[16]byte]*api.Node{}
	version := int64(1)

	for ; itr.Valid(); err = itr.Next() {
		if err != nil {
			return nil, err
		}

		n := itr.Node
		keyHash := md5.Sum(n.Key)
		if n.Delete {
			delete(state, keyHash)
		} else {
			state[keyHash] = itr.Node
		}

		if n.Block > version {
			version++
			if version > opts.Until {
				break
			}
		}
	}
	fmt.Printf("state has %d entries\n", len(state))

	var nodes []*api.Node
	for _, node := range state {
		nodes = append(nodes, node)
	}

	return rebuildFromLeaves(nodes, opts.Until), nil
}

func rebuildFromLeaves(nodes []*api.Node, version int64) *Node {
	slices.SortFunc(nodes, func(a, b *api.Node) bool {
		return bytes.Compare(a.Key, b.Key) < 0
	})
	var leaves []*Node
	for _, node := range nodes {
		n := &Node{
			key:   node.Key,
			value: node.Value,
			size:  1,
		}
		n._hash(version)
		leaves = append(leaves, n)
	}

	var iterate func([]*Node, int8) *Node
	iterate = func(nodes []*Node, height int8) *Node {
		var next []*Node
		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]
			var right *Node
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}
			n := &Node{
				leftNode:  left,
				rightNode: right,
				height:    height,
				size:      left.size + right.size,
			}
			n._hash(version)
			next = append(next, n)
		}
		if len(next) == 1 {
			return next[0]
		}
		return iterate(next, height+1)
	}
	return iterate(leaves, 0)
}
