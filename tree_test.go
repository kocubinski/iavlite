package iavlite_test

import (
	"testing"

	"github.com/kocubinski/costor-api/compact"
	"github.com/stretchr/testify/require"
)

const logDir = "./testdata/00000001-00347691.pb.gz"

func TestTree_Build(t *testing.T) {
	stream := &compact.StreamingContext{}
	itr, err := stream.NewIterator(logDir)
	require.NoError(t, err)
	for ; itr.Valid(); err = itr.Next() {
		require.NoError(t, err)
	}
}
