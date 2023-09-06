package v3

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	tree := &MutableTree{
		pool: newNodePool(),
	}
	opts := testutil.NewTreeBuildOptions(tree).With300_000()
	testutil.TestTreeBuild(t, opts)
}
