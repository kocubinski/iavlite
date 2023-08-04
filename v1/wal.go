package v1

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/gogo/protobuf/proto"
	"github.com/tidwall/wal"
)

type CountMetric interface {
	Inc()
}

type GaugeMetric interface {
	Add(float64)
	Sub(float64)
	Set(float64)
}

type HistogramMetric interface {
	Observe(float64)
}

type NaiveWal struct {
	logDir string
}

var _ proto.Message = (*WalNode)(nil)

type WalNode struct {
	Height   int64  `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	Size     int64  `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	Key      []byte `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
	Value    []byte `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
	Hash     []byte `protobuf:"bytes,5,opt,name=hash,proto3" json:"hash,omitempty"`
	LeftKey  []byte `protobuf:"bytes,6,opt,name=left_key,proto3" json:"left_key,omitempty"`
	RightKey []byte `protobuf:"bytes,7,opt,name=right_key,proto3" json:"right_key,omitempty"`
}

func (w *WalNode) Reset() { *w = WalNode{} }

func (w *WalNode) String() string { return "" }

func (w *WalNode) ProtoMessage() {}

func NewTidwalLog(logDir string) (*wal.Log, error) {
	walOpts := wal.DefaultOptions
	walOpts.NoSync = true
	walOpts.NoCopy = true
	log, err := wal.Open(fmt.Sprintf("%s/iavl.wal", logDir), walOpts)
	return log, err
}

type walCache struct {
	puts         map[nodeCacheKey]*deferredNode
	deletes      []*deferredNode
	sinceVersion int64
}

type checkpointArgs struct {
	index   uint64
	version int64
}

type Wal struct {
	wal                *wal.Log
	commitment         dbm.DB
	checkpointInterval int
	checkpointHead     uint64
	checkpointCh       chan *checkpointArgs
	CheckpointSignal   chan struct{}

	cacheLock sync.RWMutex
	hotCache  *walCache
	coldCache *walCache

	MetricNodesRead CountMetric
	MetricWalSize   GaugeMetric
	MetricCacheMiss CountMetric
	MetricCacheHit  CountMetric
	MetricCacheSize GaugeMetric
}

func NewWal(wal *wal.Log, commitment dbm.DB) *Wal {
	hot := &walCache{
		puts:    make(map[nodeCacheKey]*deferredNode),
		deletes: []*deferredNode{},
	}
	cold := &walCache{
		puts:    make(map[nodeCacheKey]*deferredNode),
		deletes: []*deferredNode{},
	}
	return &Wal{
		wal:                wal,
		commitment:         commitment,
		hotCache:           hot,
		coldCache:          cold,
		checkpointCh:       make(chan *checkpointArgs, 10),
		checkpointInterval: 10,
		CheckpointSignal:   make(chan struct{}, 2),
	}
}

func (r *Wal) Write(idx uint64, bz []byte) error {
	if r.MetricWalSize != nil {
		r.MetricWalSize.Add(float64(len(bz)))
	}
	return r.wal.Write(idx, bz)
}

func (r *Wal) CacheGet(key nodeCacheKey) (*Node, error) {
	r.cacheLock.RLock()
	hot := *r.hotCache
	cold := *r.coldCache
	r.cacheLock.RUnlock()

	if dn, ok := hot.puts[key]; ok {
		if r.MetricCacheHit != nil {
			r.MetricCacheHit.Inc()
		}
		return dn.node, nil
	}

	if dn, ok := cold.puts[key]; ok {
		if r.MetricCacheHit != nil {
			r.MetricCacheHit.Inc()
		}
		return dn.node, nil
	}

	if r.MetricCacheMiss != nil {
		r.MetricCacheMiss.Inc()
	}
	return nil, nil
}

func (r *Wal) CachePut(node *deferredNode) {
	r.cacheLock.Lock()
	cache := r.hotCache
	r.cacheLock.Unlock()

	nk := node.nodeKey
	if !node.deleted {
		cache.puts[nk] = node
	} else {
		delete(cache.puts, nk)
		nodeKey := GetNodeKey(nk[:])
		if nodeKey.version < cache.sinceVersion {
			cache.deletes = append(cache.deletes, node)
		}
	}
	if r.MetricNodesRead != nil {
		r.MetricNodesRead.Inc()
	}
	if r.MetricCacheSize != nil {
		r.MetricCacheSize.Set(float64(len(cache.puts)))
	}
}

func (r *Wal) FirstIndex() (uint64, error) {
	return r.wal.FirstIndex()
}

type deferredNode struct {
	nodeBz  *[]byte
	node    *Node
	nodeKey nodeCacheKey
	deleted bool
}

func (r *Wal) CheckpointRunner(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case args := <-r.checkpointCh:
			if r.checkpointHead == 0 {
				r.checkpointHead = args.index
			}

			if args.index-r.checkpointHead >= uint64(r.checkpointInterval) {
				r.cacheLock.Lock()
				r.hotCache, r.coldCache = r.coldCache, r.hotCache
				r.hotCache.sinceVersion = args.version + 1
				r.cacheLock.Unlock()
				err := r.Checkpoint(args.index, args.version, true)
				if err != nil {
					return err
				}
			}
		}
	}
}

func (r *Wal) Checkpoint(index uint64, version int64, async bool) error {
	var wcache *walCache
	if async {
		wcache = r.coldCache
	} else {
		wcache = r.hotCache
	}

	start := time.Now()
	setCount := 0
	deleteCount := 0
	fmt.Printf("wal: checkpointing now. [%d - %d) will be flushed to state commitment\n",
		r.checkpointHead, index)
	buf := new(bytes.Buffer)
	for k, dn := range wcache.puts {
		err := dn.node.writeBytes(buf)
		if err != nil {
			return err
		}

		err = r.commitment.Set(k[:], buf.Bytes())
		if err != nil {
			return err
		}
		buf.Reset()
		setCount++
	}
	for _, dn := range wcache.deletes {
		err := r.commitment.Delete(dn.nodeKey[:])
		if err != nil {
			return err
		}
		deleteCount++
	}
	if err := r.wal.TruncateFront(index); err != nil {
		return err
	}

	if async {
		r.cacheLock.Lock()
		r.coldCache = &walCache{
			puts:    make(map[nodeCacheKey]*deferredNode),
			deletes: []*deferredNode{},
		}
		r.cacheLock.Unlock()
	} else {
		r.hotCache = &walCache{
			puts:         make(map[nodeCacheKey]*deferredNode),
			deletes:      []*deferredNode{},
			sinceVersion: version + 1,
		}
	}

	//if r.MetricWalSize != nil {
	//	r.MetricWalSize.Sub(checkpointBz)
	//}
	if r.MetricCacheSize != nil {
		r.MetricCacheSize.Set(0)
	}
	r.checkpointHead = index
	//checkpointBz = 0

	if r.CheckpointSignal != nil {
		r.CheckpointSignal <- struct{}{}
	}

	fmt.Printf("wal: checkpoint completed in %.3fs; %d sets, %d deletes\n",
		time.Since(start).Seconds(), setCount, deleteCount)
	return nil
}

func (r *Wal) MaybeCheckpoint(index uint64, version int64) error {
	if r.checkpointHead == 0 {
		r.checkpointHead = index
	}

	if index-r.checkpointHead >= uint64(r.checkpointInterval) {
		return r.Checkpoint(index, version, false)
	}

	return nil
}
