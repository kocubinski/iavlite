package v6

import (
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	tree := &MutableTree{
		pool:    newNodePool(),
		metrics: &core.TreeMetrics{},
	}
	tree.pool.metrics = tree.metrics

	opts := testutil.NewTreeBuildOptions(tree)
	opts.Report = func() {
		tree.metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)
}
