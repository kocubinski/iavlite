package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kocubinski/costor-api/compact"
	"github.com/stretchr/testify/require"
)

type TreeBuildOptions struct {
	ChangelogDir string
	Tree         Tree
	Until        int64
	UntilHash    string
}

func (opts TreeBuildOptions) With20_000() TreeBuildOptions {
	o := &opts
	o.Until = 20_000
	o.UntilHash = "be50f7b2bdb5362f76f47a215bb4b8cc4a387bbc2478e75dcc68255e8690ac92"
	return *o
}

func (opts TreeBuildOptions) With300_000() TreeBuildOptions {
	o := &opts
	o.Until = 300_000
	o.UntilHash = "50a08008a29d76f3502d0a60c9e193a13efa6037a79a9f794652e1f97c2bbc16"
	return *o
}

func (opts TreeBuildOptions) With1_500_000() TreeBuildOptions {
	o := &opts
	o.Until = 1_500_000
	o.UntilHash = "ebc23d2e4e43075bae7ebc1e5db9d5e99acbafaa644b7c710213e109c8592099"
	return *o
}

func NewTreeBuildOptions(tree Tree) TreeBuildOptions {
	opts := TreeBuildOptions{
		ChangelogDir: "../testdata/changelogs/full/",
		Tree:         tree,
	}
	return opts.With20_000()
}

type Tree interface {
	Remove([]byte) ([]byte, bool, error)
	Set([]byte, []byte) (bool, error)
	SaveVersion() ([]byte, int64, error)
}

func TestTreeBuild(t *testing.T, opts TreeBuildOptions) {
	tree := opts.Tree

	lastVersion := int64(1)
	var (
		hash    []byte
		version int64
		cnt     int
		since   = time.Now()
	)

	stream := &compact.StreamingContext{}
	itr, err := stream.NewIterator(opts.ChangelogDir)
	require.NoError(t, err)
	for ; itr.Valid(); err = itr.Next() {
		require.NoError(t, err)
		node := itr.Node
		if !node.Delete {
			_, err = tree.Set(node.Key, node.Value)
			require.NoError(t, err)
		} else {
			_, _, err := tree.Remove(node.Key)
			require.NoError(t, err)
		}

		if node.Block > lastVersion {
			hash, version, err = tree.SaveVersion()
			require.NoError(t, err)
			if version == opts.Until {
				break
			}
			lastVersion = node.Block
		}
		if cnt%100_000 == 0 {
			fmt.Printf("processed %s leaves in %s; %s leaves/s\n",
				humanize.Comma(int64(cnt)),
				time.Since(since),
				humanize.Comma(int64(100_000/time.Since(since).Seconds())))
			since = time.Now()
		}
		cnt++
	}
	fmt.Printf("final version: %d, hash: %x\n", version, hash)
	require.Equal(t, fmt.Sprintf("%x", hash), opts.UntilHash)
	require.Equal(t, version, opts.Until)
}
