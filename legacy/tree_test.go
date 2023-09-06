package legacy

import (
	"testing"

	"github.com/kocubinski/iavlite/core"
	"github.com/kocubinski/iavlite/testutil"
)

func TestTree_Build(t *testing.T) {
	metrics := &core.TreeMetrics{}
	opts := testutil.NewTreeBuildOptions(&MutableTree{metrics: metrics})
	opts.Report = func() {
		metrics.Report()
	}
	testutil.TestTreeBuild(t, opts)
}
