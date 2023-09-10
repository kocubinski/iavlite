# impl

- ensure all set of node.leftNode and node.right is passing through setLeftNode and setRightNode which 
  syncs node.leftNodeKey and node.rightNodeKey. 
- special handling for node with dirty bit? since they will never be evicted perhaps don't need to check 
  for a fault?
- too many changes. need to reset, rewind, and start over. map out incremental changes now that I have a 
  sense of what is needed.