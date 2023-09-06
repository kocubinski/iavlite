package testutil

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kocubinski/iavl-bench/bench"
	"github.com/stretchr/testify/require"
)

type TreeBuildOptions struct {
	Tree      Tree
	Until     int64
	UntilHash string
	Iterator  bench.ChangesetIterator
}

func (opts TreeBuildOptions) With10_000() TreeBuildOptions {
	o := &opts
	o.Until = 10_000
	o.UntilHash = "460a9098015aef66f2da7f3d81fedf9a439ea3c3cf61723d535d2d94367858d5"
	return *o
}

func (opts TreeBuildOptions) With20_000() TreeBuildOptions {
	o := &opts
	o.Until = 20_000
	o.UntilHash = "be50f7b2bdb5362f76f47a215bb4b8cc4a387bbc2478e75dcc68255e8690ac92"
	return *o
}

func (opts TreeBuildOptions) With100_000() TreeBuildOptions {
	o := &opts
	o.Until = 100_000
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
	var seed int64 = 1234
	var versions int64 = 10_000_000
	bankGen := bench.BankLikeGenerator(seed, versions)
	lockupGen := bench.LockupLikeGenerator(seed, versions)
	stakingGen := bench.StakingLikeGenerator(seed, versions)
	itr, err := bench.NewChangesetIterators([]bench.ChangesetGenerator{bankGen, lockupGen, stakingGen})
	if err != nil {
		panic(err)
	}
	opts := TreeBuildOptions{
		Tree:     tree,
		Iterator: itr,
	}
	return opts.With20_000()
}

type Tree interface {
	Remove([]byte) ([]byte, bool, error)
	Set([]byte, []byte) (bool, error)
	SaveVersion() ([]byte, int64, error)
	Size() int64
	Height() int8
}

func TestTreeBuild(t *testing.T, opts TreeBuildOptions) {
	tree := opts.Tree

	var (
		hash    []byte
		version int64
		cnt     int64 = 1
		since         = time.Now()
		err     error
	)

	itrStart := time.Now()
	itr := opts.Iterator
	for ; itr.Valid(); err = itr.Next() {
		require.NoError(t, err)
		for _, node := range itr.GetChangeset().Nodes {
			var keyBz bytes.Buffer
			keyBz.Write([]byte(node.StoreKey))
			keyBz.Write(node.Key)
			key := keyBz.Bytes()

			if !node.Delete {
				_, err = tree.Set(key, node.Value)
				require.NoError(t, err)
			} else {
				_, _, err := tree.Remove(key)
				require.NoError(t, err)
			}

			if cnt%100_000 == 0 {
				fmt.Printf("processed %s leaves in %s; %s leaves/s; version=%d\n",
					humanize.Comma(int64(cnt)),
					time.Since(since),
					humanize.Comma(int64(100_000/time.Since(since).Seconds())),
					version)
				since = time.Now()
			}
			cnt++
		}
		hash, version, err = tree.SaveVersion()
		require.NoError(t, err)
		if version == opts.Until {
			break
		}
	}
	fmt.Printf("final version: %d, hash: %x\n", version, hash)
	fmt.Printf("height: %d, size: %d\n", tree.Height(), tree.Size())
	fmt.Printf("mean leaves/ms %s\n", humanize.Comma(cnt/time.Since(itrStart).Milliseconds()))
	require.Equal(t, opts.UntilHash, fmt.Sprintf("%x", hash))
	require.Equal(t, version, opts.Until)
}
