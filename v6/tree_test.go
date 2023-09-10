package v6

import (
	"fmt"
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	nk1 := &nodeKey{1, 1}
	nk2 := &nodeKey{1, 2}
	nk1_1 := &nodeKey{1, 1}

	require.Equal(t, nk1, nk1_1)
	require.NotEqual(t, nk1, nk2)
}

func TestTree_Build(t *testing.T) {
	db := newMemDB()
	tree := &MutableTree{
		pool:    newNodePool(db, 500_000),
		metrics: &core.TreeMetrics{},
		db:      db,
	}
	tree.pool.metrics = tree.metrics

	opts := testutil.NewTreeBuildOptions(tree)
	opts.Report = func() {
		tree.metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)

	height := treeHeight(tree.root)
	count := pooledTreeCount(tree, tree.root)

	workingSetCount := 0
	for _, n := range tree.pool.nodes {
		if n.dirty {
			workingSetCount++
		}
	}

	fmt.Printf("workingSetCount: %d\n", workingSetCount)
	fmt.Printf("treeCount: %d\n", count)
	fmt.Printf("treeHeight: %d\n", height)

	require.Equal(t, height, tree.root.subtreeHeight+1)
	require.Equal(t, count, len(tree.db.nodes))
	require.Equal(t, count, len(tree.pool.nodes)-len(tree.pool.free))
	require.Equal(t, tree.pool.dirtyCount, workingSetCount)
}

func treeCount(node *Node) int {
	if node == nil {
		return 0
	}
	return 1 + treeCount(node.leftNode) + treeCount(node.rightNode)
}

func pooledTreeCount(tree *MutableTree, node *Node) int {
	if node.isLeaf() {
		return 1
	}
	return 1 + pooledTreeCount(tree, node.left(tree)) + pooledTreeCount(tree, node.right(tree))
}

func treeHeight(node *Node) int8 {
	if node == nil {
		return 0
	}
	return 1 + maxInt8(treeHeight(node.leftNode), treeHeight(node.rightNode))
}
