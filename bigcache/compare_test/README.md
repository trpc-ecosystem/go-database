## Performance test
Operation types: single-goroutine write, 80-goroutine read, one-time GC invocation, single-goroutine delete

The comparison data is as follows.

## bigcache
```
readRoutines:80, totalItems:1000000, kpre:k vpre:v
write bigCache cost = 908.050759ms
read BigCache cost = 6.122891338s
2019/12/19 16:17:01  bigCache nil Pause:10242 1
del BigCache cost = 477.629119ms
```
## localcache
```
write local cost = 1.028851498s
read local cost = 7.630089803s
2019/12/19 16:17:11  localcache nil Pause:27548 1
delete local cost = 1.069555972s
```
## sync.Map
```
write sync.Map cost = 1.35084257s
read sync.Map cost = 6.888130791s
2019/12/25 17:05:29  sync.Map nil Pause:41345 1
del sync.Map cost = 776.195203ms
```



