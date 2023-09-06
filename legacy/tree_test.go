package legacy

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	opts := testutil.NewTreeBuildOptions(&MutableTree{}).With100_000()
	testutil.TestTreeBuild(t, opts)
}
