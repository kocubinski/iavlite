package v2

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	tree := NewMutableTree()
	tree.pool = newNodePool()
	opts := testutil.NewTreeBuildOptions(tree).With1_500_000()
	testutil.TestTreeBuild(t, opts)
}
