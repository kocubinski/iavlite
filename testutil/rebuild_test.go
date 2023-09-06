package testutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	api "github.com/kocubinski/costor-api"
	"github.com/kocubinski/iavl-bench/bench"
	"github.com/stretchr/testify/require"
)

func Test_Rebuild(t *testing.T) {
	//stream := &compact.StreamingContext{}
	opts := NewTreeBuildOptions(nil).With300_000()
	//itr, err := stream.NewIterator(opts.ChangelogDir)
	var (
		seed     int64 = 1234
		versions int64 = 3_000_000
		cnt      int64 = 1
	)

	bankGen := bench.BankLikeGenerator(seed, versions)
	lockupGen := bench.LockupLikeGenerator(seed, versions)
	stakingGen := bench.StakingLikeGenerator(seed, versions)
	itr, err := bench.NewChangesetIterators([]bench.ChangesetGenerator{bankGen, lockupGen, stakingGen})
	if err != nil {
		panic(err)
	}

	state := map[[16]byte]*api.Node{}
	since := time.Now()

	require.NoError(t, err)
	for ; itr.Valid(); err = itr.Next() {
		done := false
		for _, n := range itr.GetChangeset().Nodes {
			require.NoError(t, err)
			var keyBz bytes.Buffer
			keyBz.Write([]byte(n.StoreKey))
			keyBz.Write(n.Key)
			key := keyBz.Bytes()

			keyHash := md5.Sum(key)
			if n.Delete {
				delete(state, keyHash)
			} else {
				state[keyHash] = n
			}

			if n.Block > opts.Until {
				done = true
				break
			}
			cnt++
			if cnt%100_000 == 0 {
				fmt.Printf("processed %s leaves in %s; %s leaves/s; version=%d\n",
					humanize.Comma(int64(cnt)),
					time.Since(since),
					humanize.Comma(int64(100_000/time.Since(since).Seconds())),
					n.Block)
				since = time.Now()
			}
			cnt++
		}
		if done {
			break
		}
	}
	fmt.Printf("state has %d entries\n", len(state))
}
