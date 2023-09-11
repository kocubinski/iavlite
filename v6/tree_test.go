package v6

import (
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
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
	//just a little bigger than the size of the initial changeset. evictions will occur slowly.
	//poolSize := 210_050
	// no evictions
	poolSize := 500_000
	// overflow on initial changeset
	//poolSize := 100_000

	db := newMemDB()
	tree := &MutableTree{
		pool:               newNodePool(db, poolSize),
		metrics:            &core.TreeMetrics{},
		db:                 db,
		checkpointInterval: 10_000,
	}
	tree.pool.metrics = tree.metrics

	opts := testutil.NewTreeBuildOptions(tree)
	opts.Report = func() {
		tree.metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)

	err := tree.Checkpoint()
	require.NoError(t, err)

	// don't evict root on iteration, it interacts with the node pool
	tree.root.dirty = true
	count := pooledTreeCount(tree, *tree.root)
	height := pooledTreeHeight(tree, *tree.root)

	workingSetCount := -1 // offset the dirty root above.
	for _, n := range tree.pool.nodes {
		if n.dirty {
			workingSetCount++
		}
	}

	fmt.Printf("workingSetCount: %d\n", workingSetCount)
	fmt.Printf("treeCount: %d\n", count)
	fmt.Printf("treeHeight: %d\n", height)
	fmt.Printf("db stats:\n sets: %s, deletes: %s\n",
		humanize.Comma(int64(db.setCount)),
		humanize.Comma(int64(db.deleteCount)))

	require.Equal(t, height, tree.root.subtreeHeight+1)
	require.Equal(t, count, len(tree.db.nodes))
	require.Equal(t, tree.pool.dirtyCount, workingSetCount)

	treeAndDbEqual(t, tree, *tree.root)
}

func treeCount(node *Node) int {
	if node == nil {
		return 0
	}
	return 1 + treeCount(node.leftNode) + treeCount(node.rightNode)
}

func pooledTreeCount(tree *MutableTree, node Node) int {
	if node.isLeaf() {
		return 1
	}
	left := *node.left(tree)
	right := *node.right(tree)
	return 1 + pooledTreeCount(tree, left) + pooledTreeCount(tree, right)
}

func pooledTreeHeight(tree *MutableTree, node Node) int8 {
	if node.isLeaf() {
		return 1
	}
	left := *node.left(tree)
	right := *node.right(tree)
	return 1 + maxInt8(pooledTreeHeight(tree, left), pooledTreeHeight(tree, right))
}

func treeAndDbEqual(t *testing.T, tree *MutableTree, node Node) {
	dbNode := tree.db.Get(*node.nodeKey)
	require.NotNil(t, dbNode)
	require.Equal(t, dbNode.hash, node.hash)
	require.Equal(t, dbNode.nodeKey, node.nodeKey)
	require.Equal(t, dbNode.key, node.key)
	require.Equal(t, dbNode.value, node.value)
	require.Equal(t, dbNode.size, node.size)
	require.Equal(t, dbNode.subtreeHeight, node.subtreeHeight)
	require.Equal(t, dbNode.leftNodeKey, node.leftNodeKey)
	require.Equal(t, dbNode.rightNodeKey, node.rightNodeKey)
	if node.isLeaf() {
		return
	}
	leftNode := *node.left(tree)
	rightNode := *node.right(tree)
	treeAndDbEqual(t, tree, leftNode)
	treeAndDbEqual(t, tree, rightNode)
}
