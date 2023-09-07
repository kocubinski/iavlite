# impl

- implement a configuration where each version is flushed to `memDB`. In this configuration constrain the 
  node pool size to something pretty small like 100,000 nodes. This will force the page cache to evict 
  nodes and test page fault behavior.