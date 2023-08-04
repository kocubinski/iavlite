package v2

import (
	"bytes"
	"fmt"
	"time"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/gogo/protobuf/proto"
)

type nodeCacheKey [12]byte

type NodeDb interface {
	// QueueNode queues a node for storage.
	QueueNode(*Node) error

	// QueueOrphan queues a node for orphaning.
	QueueOrphan(*Node) error

	// Commit commits all queued nodes to storage.
	Commit(int64) error

	GetNode(nodeKey []byte) (*Node, error)
}

var _ NodeDb = (*KeyValueBackend)(nil)

type KeyValueBackend struct {
	nodes   []*Node
	orphans []*Node
	db      dbm.DB
	walBuf  *bytes.Buffer
	wal     *Wal
	walIdx  uint64

	// metrics
	MetricBlockCount      CountMetric
	MetricCacheSize       GaugeMetric
	MetricCacheMiss       CountMetric
	MetricCacheHit        CountMetric
	MetricDbFetch         CountMetric
	MetricDbFetchDuration HistogramMetric
}

func NewKeyValueBackend(db dbm.DB, wal *Wal) (*KeyValueBackend, error) {
	walIdx, err := wal.FirstIndex()
	if err != nil {
		return nil, err
	}
	if walIdx == 0 {
		walIdx = 1
	}

	return &KeyValueBackend{
		db:     db,
		wal:    wal,
		walIdx: walIdx,
		walBuf: new(bytes.Buffer),
	}, nil
}

func (kv *KeyValueBackend) QueueNode(node *Node) error {
	if node.nodeKey == nil {
		return fmt.Errorf("empty node key")
	}
	kv.nodes = append(kv.nodes, node)
	return nil
}

func (kv *KeyValueBackend) QueueOrphan(node *Node) error {
	if node.nodeKey == nil {
		return fmt.Errorf("empty node key")
	}
	kv.orphans = append(kv.orphans, node)
	return nil
}

func (kv *KeyValueBackend) Commit(version int64) error {
	var nk nodeCacheKey

	changeset := &ChangeSet{}
	for _, node := range kv.nodes {
		changeset.Pairs = append(changeset.Pairs, &KVPair{Key: node.key, Value: node.value})

		copy(nk[:], node.nodeKey.GetKey())
		dn := &deferredNode{nodeKey: nk, node: node}
		kv.wal.CachePut(dn)
	}

	for _, node := range kv.orphans {
		copy(nk[:], node.nodeKey.GetKey())
		changeset.Pairs = append(changeset.Pairs, &KVPair{Key: node.key, Value: node.value, Delete: true})
		dn := &deferredNode{nodeKey: nk, deleted: true, node: node}
		kv.wal.CachePut(dn)
	}

	walBz, err := proto.Marshal(changeset)
	if err != nil {
		return err
	}
	kv.walBuf.Write(walBz)

	if kv.walBuf.Len() > 50*1024*1024 {
		err = kv.wal.Write(kv.walIdx, kv.walBuf.Bytes())
		if err != nil {
			return err
		}
		// TODO: support single threaded checkpoint by configuration
		err = kv.wal.MaybeCheckpoint(kv.walIdx, version)
		if err != nil {
			return err
		}
		// ASYNC checkpoint
		//kv.wal.checkpointCh <- &checkpointArgs{kv.walIdx, version, kv.nodeCache}
		kv.walBuf.Reset()
		kv.walIdx++
	}

	kv.nodes = nil
	kv.orphans = nil

	if kv.MetricBlockCount != nil {
		kv.MetricBlockCount.Inc()
	}

	return nil
}

func (kv *KeyValueBackend) GetNode(nodeKey []byte) (*Node, error) {
	var nk nodeCacheKey
	copy(nk[:], nodeKey)

	// fetch from commitment store
	if kv.MetricDbFetch != nil {
		kv.MetricDbFetch.Inc()
	}
	since := time.Now()
	value, err := kv.db.Get(nodeKey)
	if err != nil {
		return nil, err
	}
	if kv.MetricDbFetchDuration != nil {
		kv.MetricDbFetchDuration.Observe(time.Since(since).Seconds())
	}
	if value == nil {
		return nil, fmt.Errorf("kv/GetNode; node not found; nodeKey: %s [%X]", GetNodeKey(nodeKey), nodeKey)
	}
	node, err := MakeNode(nodeKey, value)
	if err != nil {
		return nil, fmt.Errorf("kv/GetNode/MakeNode; nodeKey: %s [%X]; bytes: %X; %w",
			GetNodeKey(nodeKey), nodeKey, value, err)
	}

	return node, nil
}
