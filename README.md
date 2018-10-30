[![GoDoc](https://godoc.org/github.com/dhilipkumars/redis-sentinel-micro?status.svg)](https://godoc.org/github.com/dhilipkumars/redis-sentinel-micro)
[![Build Status](https://drone.io/github.com/dhilipkumars/redis-sentinel-micro/status.png)](https://drone.io/github.com/dhilipkumars/redis-sentinel-micro/latest)
[![Coverage Status](https://coveralls.io/repos/github/dhilipkumars/redis-sentinel-micro/badge.svg)](https://coveralls.io/github/dhilipkumars/redis-sentinel-micro)
[![Go Report Card](https://goreportcard.com/badge/github.com/dhilipkumars/redis-sentinel-micro)](https://goreportcard.com/report/github.com/dhilipkumars/redis-sentinel-micro)

### NOTE: This version of sentinel-micro is modified to work with Kubernetes Statefulset  

# redis-sentinel-micro
Minimalistic redis sentinel process only to do slave promoton

## Rationale 
If you are running redis-servers on infrastructues that are managed by cluster managers like `kubernetes` or `apache mesos`, these cluster managers automatically take care of `redis-server`'s life cycle, we dont need sentinel for that.  The only thing that would be missing is slave promotion incase the master crashed.  We want an utility only for that functionality. 

## Algorithm
This project follows the same approach that is followed by the redis-sentinel in function `compareSlavesForPromotion()`
```
int compareSlavesForPromotion(const void *a, const void *b) {
   
   /*Step 1: Lowest Priority. */

   /*Step 2: Maximum replication offset. */
   
   /* Step 3: Lexographic sort based on runid. */
   
   }
```
except `Step 3`, where instead of Lexographic `redis-sentinel-micro` selects the slave which completed replication most recently, based on `master_last_io_seconds_ago` attribute.


## Usage
Assume a Redis Instance with 1 Master and 5 Slaves, like below
```
role:master
connected_slaves:5
slave0:ip=172.31.10.90,port=6385,state=online,offset=197,lag=1
slave1:ip=172.31.10.90,port=6386,state=online,offset=197,lag=1
slave2:ip=172.31.10.90,port=6384,state=online,offset=197,lag=1
slave3:ip=172.31.10.90,port=6383,state=online,offset=197,lag=1
slave4:ip=172.31.10.90,port=6382,state=online,offset=197,lag=1
master_repl_offset:197
```
The master crashes mid-way leaving all the 5 slaves in an inconsistent stage like below
```
slave_repl_offset:98187702
slave_repl_offset:98187702
slave_repl_offset:98150330
slave_repl_offset:98134252
slave_repl_offset:98134252
```
All the above slaves have equal priority, but we have 3 replication values `98134252`, `98150330` and `98187702`. Ideally the slave with highest replication value should be promoted. 
When you run `redis-sentinel-micro` like this it automatically detects the next master and re-configures all the slaves to point to this new master
```
$./redis-sentinel-micro -logtostderr 0.0.0.0:6382 0.0.0.0:6383 0.0.0.0:6384 0.0.0.0:6385 0.0.0.0:6386        
I1213 19:19:58.530901   16261 redis_sentinel_micro.go:290] Supplied Arguments are [0.0.0.0:6382 0.0.0.0:6383 0.0.0.0:6384 0.0.0.0:6385 0.0.0.0:6386]
I1213 19:19:58.531001   16261 redis_sentinel_micro.go:161] Processing 0.0.0.0:6382
I1213 19:19:58.531042   16261 redis_sentinel_micro.go:161] Processing 0.0.0.0:6383
I1213 19:19:58.531073   16261 redis_sentinel_micro.go:161] Processing 0.0.0.0:6384
I1213 19:19:58.531102   16261 redis_sentinel_micro.go:161] Processing 0.0.0.0:6385
I1213 19:19:58.531135   16261 redis_sentinel_micro.go:161] Processing 0.0.0.0:6386
R=&{ slave -1 610 98150330 172.31.10.90 6381 100 false <nil>}
R=&{ slave -1 610 98134252 172.31.10.90 6381 100 false <nil>}
R=&{ slave -1 610 98187702 172.31.10.90 6381 100 false <nil>}
R=&{ slave -1 610 98134252 172.31.10.90 6381 100 false <nil>}
R=&{ slave -1 610 98187702 172.31.10.90 6381 100 false <nil>}
I1213 19:19:58.532407   16261 redis_sentinel_micro.go:275] PrintServers()
<<SNIPPED prints an elaborate json array>>
I1213 19:19:58.533968   16261 redis_sentinel_micro.go:303] New Master={0.0.0.0:6382 slave -1 610 98187702 172.31.10.90 6381 100 false 0xc82007e300}
I1213 19:19:58.534858   16261 redis_sentinel_micro.go:312] New Master is 0.0.0.0:6382, All the slaves are re-configured to replicate from this
I1213 19:19:58.534986   16261 redis_sentinel_micro.go:275] PrintServers()
<<SNIPPED prints an elaborate json array>>
I1213 19:19:58.536414   16261 redis_sentinel_micro.go:316] Redis-Sentinal-micro Finished
```
The slave `6382` becomes the new master
```
./redis-cli -h 0.0.0.0 -p 6382 info replication
# Replication
role:master
connected_slaves:4
slave0:ip=127.0.0.1,port=6385,state=online,offset=631,lag=1
slave1:ip=127.0.0.1,port=6384,state=online,offset=631,lag=0
slave2:ip=127.0.0.1,port=6383,state=online,offset=631,lag=1
slave3:ip=127.0.0.1,port=6386,state=online,offset=631,lag=0
master_repl_offset:631
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:2
repl_backlog_histlen:630
```

If you run it on a stable (a properly configured) cluster, it does nothing
```
./redis-sentinel-micro -logtostderr 0.0.0.0:6382 0.0.0.0:6383 0.0.0.0:6384 0.0.0.0:6385 0.0.0.0:6386
I1213 19:29:12.034424   16369 redis_sentinel_micro.go:290] Supplied Arguments are [0.0.0.0:6382 0.0.0.0:6383 0.0.0.0:6384 0.0.0.0:6385 0.0.0.0:6386]
I1213 19:29:12.034517   16369 redis_sentinel_micro.go:161] Processing 0.0.0.0:6382
I1213 19:29:12.034568   16369 redis_sentinel_micro.go:161] Processing 0.0.0.0:6383
I1213 19:29:12.034602   16369 redis_sentinel_micro.go:161] Processing 0.0.0.0:6384
I1213 19:29:12.034636   16369 redis_sentinel_micro.go:161] Processing 0.0.0.0:6385
I1213 19:29:12.034681   16369 redis_sentinel_micro.go:161] Processing 0.0.0.0:6386
R=&{ master 0 0 771   0 false <nil>}
R=&{ slave 7 0 771 0.0.0.0 6382 100 false <nil>}
R=&{ slave 7 0 771 0.0.0.0 6382 100 false <nil>}
R=&{ slave 7 0 771 0.0.0.0 6382 100 false <nil>}
R=&{ slave 7 0 771 0.0.0.0 6382 100 false <nil>}
I1213 19:29:12.036024   16369 redis_sentinel_micro.go:275] PrintServers()
<<SNIPPED prints an elaborate json array>>
I1213 19:29:12.037455   16369 redis_sentinel_micro.go:205] RSMaster_EP=0.0.0.0:6382 available MasterEP=0.0.0.0:6382
I1213 19:29:12.037498   16369 redis_sentinel_micro.go:205] RSMaster_EP=0.0.0.0:6382 available MasterEP=0.0.0.0:6382
I1213 19:29:12.037534   16369 redis_sentinel_micro.go:205] RSMaster_EP=0.0.0.0:6382 available MasterEP=0.0.0.0:6382
I1213 19:29:12.037558   16369 redis_sentinel_micro.go:205] RSMaster_EP=0.0.0.0:6382 available MasterEP=0.0.0.0:6382
W1213 19:29:12.037581   16369 redis_sentinel_micro.go:220] The redis master is already configured, dont do anything SyncBytes=771 availableMasterHits=4 len(Slaves)=4
E1213 19:29:12.037606   16369 redis_sentinel_micro.go:300] Redis Instance does'nt need a Slave Promotion
```

## Building docker images
> All building can happen in docker to keep from requiring build dependencies on local machine.

### Sentinel
To build a new image for the `sentinel` binary, run the following command:

```
docker build -t dhilipkumars/redis-sentinel-k8s:0.2.0 ./
```

### Make Slave
To build a new image for the `make_slave` binary, run the following command:

```
docker build -t dhilipkumars/mk-redis-slave:0.2.0 -f mk_redis_slave/Dockerfile ./mk_redis_slave
```