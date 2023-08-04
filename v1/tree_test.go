package legacy

import (
	"fmt"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kocubinski/costor-api/compact"
	"github.com/stretchr/testify/require"
)

const logDir = "../testdata/changelogs"

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
			//time.Sleep(100 * time.Millisecond)
			require.NoError(t, err)
			if version%20000 == 0 {
				fmt.Printf("%d:%x\n", version, hash)
				break
			}
			lastVersion = node.Block
		}
		if cnt%10_000 == 0 {
			fmt.Printf("processed %s leaves in %s; %s leaves/s\n",
				humanize.Comma(int64(cnt)),
				time.Since(since),
				humanize.Comma(int64(10_000/time.Since(since).Seconds())))
			since = time.Now()
		}
		cnt++
	}
	fmt.Printf("final version: %d, hash: %x\n", version, hash)
}
