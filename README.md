# iavlite

iavl prototypes.

## versions

### legacy/

the iavl v1 implementation with certain key sections commented out, namely.

- saveNewNodes does not write to a backing database.
- saveNewNodes does not clear leftNode, rightNode = nil.
- node.getLeftNode, node.getRightNode fails if node.leftNode or node.rightNode is nil.

These changes have the effect of leaving the entire tree in memory and testing the overall efficiency of
tree traversal, rotations, and hashing algorithms in IAVL.  Other implementations follow the same
methodology to build a baseline of performance.

### v0

A failing a fresh start experiment based loosely on memiavl.  Currently very slow and failing hash validations.

### v1

basically the legacy implementation with additional (unused) code pasted in for pushing data into a backing database.

### v2

First try at incorporating a node pool, something like a buffer pool/cache. Uses the GhostNode concept.  Node look ups
proceed like: `node key -> frameId -> node`. Slow throughput and fails hash validation.

```
processed 19,200,000 leaves in 1.876538959s; 53,289 leaves/s
processed 19,300,000 leaves in 1.750037959s; 57,141 leaves/s 
final version: 1500000, hash: d80f4fa84b60ee4f943b0d75a7ae79de22b6c1923a1a6b915d543fb82ed4c454
    core.go:98:
        	Error Trace:	/Users/mattk/src/iavl/iavlite/testutil/core.go:98
        	Error:      	Not equal:
        	            	expected: "d80f4fa84b60ee4f943b0d75a7ae79de22b6c1923a1a6b915d543fb82ed4c454"
        	            	actual  : "ebc23d2e4e43075bae7ebc1e5db9d5e99acbafaa644b7c710213e109c8592099" 
```

### v3

Second try at a node pool.  Stores free list id as a node pointer for a direct look up in the buffer pool like `frameId -> node`. Faster and passes hash validation.

```
processed 19,200,000 leaves in 1.327169625s; 75,348 leaves/s
processed 19,300,000 leaves in 1.08690575s; 92,004 leaves/s 
```

### v4

legacy implementation with `node.reset()` instead of `node.clone()` to reset nodes to relieve GC pressure. 
Includes a parallel hash `writeHashBytes2` which uses the stdlib `binary` package instead of 
`internal/encoding` to write bytes.  Overall increased throughput mostly due to `reset()`.  Still behind 
`memiavl` on GC cycles but this is probably due to (de)allocation of nodekey byte arrays.  GC won't be the 
limiting factor on a live chain so probably not worth optimizing further.

## TODO

- Memory and buffer pool metrics
- Use unsafe.Pointer instead of frameId to process nodes in tree traversals.

