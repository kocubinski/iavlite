package v3

import (
	"fmt"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kocubinski/costor-api/compact"
	"github.com/stretchr/testify/require"
)

const logDir = "/Users/mattk/src/scratch/osmosis-hist/bank-ordered/"

const until = 10

//const until = 300_000
//const until = 1_500_000

func TestTree_Build(t *testing.T) {
	tree := MutableTree{}

	lastVersion := int64(1)
	var (
		hash    []byte
		version int64
		cnt     int
		since   = time.Now()
	)

	stream := &compact.StreamingContext{}
	itr, err := stream.NewIterator(logDir)
	require.NoError(t, err)
	for ; itr.Valid(); err = itr.Next() {
		require.NoError(t, err)
		node := itr.Node
		if !node.Delete {
			_, err = tree.Set(node.Key, node.Value)
			require.NoError(t, err)
		} else {
			tree.Remove(node.Key)
		}

		if node.Block > lastVersion {
			hash, version, err = tree.SaveVersion()
			require.NoError(t, err)
			lastVersion = node.Block
			if version%1 == 0 {
				fmt.Printf("treeVersion: %d, blockHeight: %d, hash: %x\n", version, node.Block, hash)
			}
			if version == until {
				break
			}
		}
		if cnt%600_000 == 0 {
			fmt.Printf("processed %s leaves in %s; %s leaves/s\n",
				humanize.Comma(int64(cnt)),
				time.Since(since),
				humanize.Comma(int64(600_000/time.Since(since).Seconds())))
			since = time.Now()
		}
		cnt++
	}
	fmt.Printf("final version: %d, hash: %x\n", version, hash)
	//require.Equal(t, fmt.Sprintf("%x", hash), "ebc23d2e4e43075bae7ebc1e5db9d5e99acbafaa644b7c710213e109c8592099")
	//require.Equal(t, version, int64(1_500_000))

	//require.Equal(t, fmt.Sprintf("%x", hash), "50a08008a29d76f3502d0a60c9e193a13efa6037a79a9f794652e1f97c2bbc16")
	//require.Equal(t, version, int64(300_000))

	require.Equal(t, "2b63f92e4362cde030d18752afcc86fca7d405ebfe69825605180f7037007672", fmt.Sprintf("%x", hash))
	require.Equal(t, int64(10), version)
}
