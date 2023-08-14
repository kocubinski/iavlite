package v0

import (
	"testing"

	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	opts := testutil.NewTreeBuildOptions(&Tree{
		version:  1,
		sequence: 1,
	})
	testutil.TestTreeBuild(t, opts)
}
