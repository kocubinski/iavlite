package testutil

import (
	"crypto/md5"
	"fmt"
	"testing"

	api "github.com/kocubinski/costor-api"
	"github.com/kocubinski/costor-api/compact"
	"github.com/stretchr/testify/require"
)

func Test_Rebuild(t *testing.T) {
	stream := &compact.StreamingContext{}
	opts := NewTreeBuildOptions(nil).With300_000()
	itr, err := stream.NewIterator(opts.ChangelogDir)

	state := map[[16]byte]*api.Node{}

	require.NoError(t, err)
	for ; itr.Valid(); err = itr.Next() {
		require.NoError(t, err)
		n := itr.Node
		keyHash := md5.Sum(n.Key)
		if n.Delete {
			delete(state, keyHash)
		} else {
			state[keyHash] = itr.Node
		}

		if n.Block > opts.Until {
			break
		}
	}
	fmt.Printf("state has %d entries\n", len(state))
}
