# impl

special handling for node with dirty bit? since they will never be evicted perhaps don't need to check 
for a fault?

track bytes in pool instead of node count. this is a more realistic metric for memory usage.


## checkpoint

also called snapshot, flushing.
periodic checkpoint when pool working set is in overflow state (dirty nodes > soft ceiling), or a 
specified number of blocks have elapsed since the last checkpoint.  briefly lock the pool so that a `lock` 
bit can be set on each node pending checkpoint write.  `node.mutate` should refuse to mutate a node with 
`lock = true`.  test atomic bool performance so that `lock = false` may be set as nodes are written, 
otherwise a second global lock after the checkpoint is done can update `lock = false`.  

generally this mimics the behavior of a double buffer so long as there is space in the pool.
 
### clean up

aesthetics: migrate node to active record pattern with a pool handle so that `.left(tree)` -> `.left()`