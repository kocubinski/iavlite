package legacy

import (
	"fmt"
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	metrics := &core.TreeMetrics{}
	tree := &MutableTree{metrics: metrics}
	opts := testutil.NewTreeBuildOptions(tree)
	opts.Report = func() {
		metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)
	fmt.Printf("treeCount: %d\n", treeCount(tree.root))
	height := treeHeight(tree.root)
	fmt.Printf("treeHeight: %d\n", height)

	heightCounts := map[int]int{}
	for i := 0; i < int(height); i++ {
		heightCounts[i] = 0
	}
	treeHeightCounts(tree.root, heightCounts, 0)
	fmt.Printf("treeHeightCounts: %v\n", heightCounts)
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

func treeHeightCounts(node *Node, heightCounts map[int]int, depth int) {
	if node == nil {
		return
	}
	heightCounts[depth]++
	treeHeightCounts(node.leftNode, heightCounts, depth+1)
	treeHeightCounts(node.rightNode, heightCounts, depth+1)
}
