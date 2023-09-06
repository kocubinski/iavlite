package v6

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	opts := testutil.NewTreeBuildOptions(&MutableTree{
		pool: newNodePool(),
	}).With10_000()
	testutil.TestTreeBuild(t, opts)
}