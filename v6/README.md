# iavl-v2 (v6) design

## State Commitment

This is a prune-free state commitment (SC) design similar to MemIAVL featuring periodic checkpoints 
(snapshots) to disk.  Instead of mmap a page cache is used, giving us more control over memory management. 
Only the most recent SC tree is kept in memory.  Instead of a custom snapshot format persisted nodes are 
written to an LSM tree.

SC maintains one large page cache.  The eviction policy is CLOCK with two bits on each node, `use` and 
`dirty`. `use` indicates that the node was recently used and `dirty` indicates the node is part of the 
current working  set.  CLOCK will unset a `use` bit when seen but only the checkpoint process will unset a 
`dirty` bit.

The cache maintains two slices of frame ids, `hotWorkingSet` and `coldWorkingSet`. `hotWorkingSet` 
accumulates working nodes as they created in the IAVL tree. Once the page cache reaches a memory 
threshold *or* a certain amount of time (blocks) has elapsed since the last flush, the `hotWorkingSet` is 
flushed (checkpoint) to disk. This involves swapping `hotWorkingSet` and `coldWorkingSet`, enumerating the 
frame ids, fetching the nodes from cache, writing to disk and clearing the `dirty` bit.

Instead of IAVL traversal fetching from a slice or other external data structure as a buffer pool,
the pointers `node.leftNode` and `node.rightNode` just fetch directly from the page cache which is a slice
of pre-allocated `Node` structs. This also has the effect of relieving GC pressure. Therefore, when fetching
left or right nodes the following tasks must be performed: 1) dereference the node pointer, 2) compare 
node keys, then 3a) if equal set the `use` bit, or 3b) if unequal (page fault) fetch the node from disk then 
evict and replace a node in cache.

Periodic snapshot cleanup can be performed by a background process using the IAVL tree diff 
algorithm introduced in [iavl#646](https://github.com/cosmos/iavl/pull/646) to identify and delete orphans.

## State Storage

On every commit leaves (application key-value pairs) are **always** flushed directly to state storage (SS).
SS therefore behaves as a write-ahead log (WAL).  If SS writes are fast enough a proper WAL is not needed. 
If however an append-only WAL has better write performance a similar strategy to state commitment (SC) can 
be used to increase throughput.  Namely, a in-memory working set to service reads which is periodically 
flushed at checkpoint intervals to disk and the WAL truncated.  The presumes that writing in batches to SS 
is more performant than every block, which may the case for certain SS backends, or due to the decreased 
read IO resulting from the in-memory working set.