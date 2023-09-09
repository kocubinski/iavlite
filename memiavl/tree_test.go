package memiavl

import (
	"fmt"
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func Test_BuildTree(t *testing.T) {
	tree := &Tree{}
	opts := testutil.NewTreeBuildOptions(tree)
	testutil.TestTreeBuild(t, opts)

	root := tree.root.(*MemNode)
	fmt.Printf("treeCount: %d\n", treeCount(root))
	height := treeHeight(tree.root)
	fmt.Printf("treeHeight: %d\n", height)

	heightCounts := map[int]int{}
	for i := 0; i < int(height); i++ {
		heightCounts[i] = 0
	}
	treeHeightCounts(tree.root, heightCounts, 0)
	fmt.Printf("treeHeightCounts: %v\n", heightCounts)
}

func treeCount(node Node) int {
	if node == nil {
		return 0
	}
	return 1 + treeCount(node.Left()) + treeCount(node.Right())
}

func treeHeight(node Node) uint8 {
	if node == nil {
		return 0
	}
	return 1 + maxUInt8(treeHeight(node.Left()), treeHeight(node.Right()))
}

func treeHeightCounts(node Node, heightCounts map[int]int, depth int) {
	if node == nil {
		return
	}
	heightCounts[depth]++
	treeHeightCounts(node.Left(), heightCounts, depth+1)
	treeHeightCounts(node.Right(), heightCounts, depth+1)
}
