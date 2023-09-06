package v4

import (
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	metrics := &core.TreeMetrics{}
	tree := &MutableTree{
		db:      newMemDB(),
		pool:    newNodePool(),
		metrics: metrics,
	}
	tree.pool.metrics = metrics
	opts := testutil.NewTreeBuildOptions(tree).With100_000()
	opts.Report = func() {
		metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)
}
