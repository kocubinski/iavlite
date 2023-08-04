package iavlite_test

import (
	"testing"

	"github.com/kocubinski/costor-api/compact"
)

const logDir = "/Users/mattk/src/scratch/osmosis-hist/brief"

func TestTree_Build(t *testing.T) {
	stream := &compact.StreamingContext{}
	itr, err := stream.NewIterator(logDir)
}
