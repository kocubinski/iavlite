package v4

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	opts := testutil.NewTreeBuildOptions(&MutableTree{}).With1_500_000()
	testutil.TestTreeBuild(t, opts)
}
