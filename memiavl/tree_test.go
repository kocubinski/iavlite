package memiavl_test

import (
	"testing"

	"github.com/kocubinski/iavlite/memiavl"
	"github.com/kocubinski/iavlite/testutil"
)

func Test_BuildTree(t *testing.T) {
	tree := &memiavl.Tree{}
	opts := testutil.NewTreeBuildOptions(tree).With1_500_000()
	testutil.TestTreeBuild(t, opts)
}
