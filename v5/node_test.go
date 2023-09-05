package v5

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RebuildTree(t *testing.T) {
	root, err := RebuildTree()
	require.NoError(t, err)
	fmt.Printf("root: %v\n", root)
}
