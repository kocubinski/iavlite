package core

import (
	"fmt"

	"github.com/dustin/go-humanize"
)

type TreeMetrics struct {
	PoolGets    int64
	PoolReturns int64
	TreeUpdate  int64
	TreeNewNode int64
	TreeDelete  int64
}

func (m *TreeMetrics) Report() {
	fmt.Printf("Pool:\n gets: %s, returns: %s\nTree:\n update: %s, new node: %s, delete: %s\n",
		humanize.Comma(m.PoolGets),
		humanize.Comma(m.PoolReturns),
		humanize.Comma(m.TreeUpdate),
		humanize.Comma(m.TreeNewNode),
		humanize.Comma(m.TreeDelete))
}
