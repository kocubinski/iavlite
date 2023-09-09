package v6

import (
	"fmt"
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
	"github.com/stretchr/testify/require"
)

func TestTree_Build(t *testing.T) {
	tree := &MutableTree{
		pool:    newNodePool(3_000_000),
		metrics: &core.TreeMetrics{},
		db:      newMemDB(),
	}
	tree.pool.metrics = tree.metrics

	opts := testutil.NewTreeBuildOptions(tree)
	opts.Report = func() {
		tree.metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)

	height := treeHeight(tree.root)
	count := treeCount(tree.root)

	workingSetCount := 0
	for _, hn := range tree.pool.hotSet {
		if hn.nodeKey != nil {
			workingSetCount++
		}
	}

	fmt.Printf("workingSetCount: %d\n", workingSetCount)
	fmt.Printf("treeCount: %d\n", count)
	fmt.Printf("treeHeight: %d\n", height)

	require.Equal(t, height, tree.root.subtreeHeight+1)
	require.Equal(t, count, len(tree.db.nodes))
	require.Equal(t, count, len(tree.pool.nodes)-len(tree.pool.free))
	require.Equal(t, count, len(tree.pool.hotSet))
}

func treeCount(node *Node) int {
	if node == nil {
		return 0
	}
	return 1 + treeCount(node.leftNode) + treeCount(node.rightNode)
}

func treeHeight(node *Node) int8 {
	if node == nil {
		return 0
	}
	return 1 + maxInt8(treeHeight(node.leftNode), treeHeight(node.rightNode))
}
