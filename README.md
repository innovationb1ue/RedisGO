# RedisGO

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/innovationb1ue/RedisGO/blob/main/LICENSE)
![Go Report Card](https://goreportcard.com/badge/github.com/innovationb1ue/RedisGO)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationb1ue/RedisGO#section-readme.svg)](https://pkg.go.dev/github.com/innovationb1ue/RedisGO#section-readme)


(https://goreportcard.com/report/github.com/innovationb1ue/RedisGO)  
**RedisGO** is a high-performance distributed cache server written by GO. 
It use [Raft](https://raft.github.io/) for consensus and fault tolerance. 
It also reaches peak performance for the original C-Redis  
It implemented full [RESP](https://redis.io/docs/reference/protocol-spec/)(Redis Serialization Protocol), so it supports
all standard Redis clients.

## Base Works
The codebase is adapted from this version of
[thinredis](https://github.com/VincentFF/thinredis/tree/86fa648426da7e9c3ff4c04aef1e43f1fdc7b1ac)


## Features

* Support all Clients based on RESP protocol
* Support String, List, Set, SortedSet, and Hash data types
* Use AVL tree on Sorted Set (not skip list)
* Support TTL(Key-Value pair will be deleted after TTL)
* Dedicate memory usage. (C-Redis is not able to release any allocated memory, but we can!)
* Full in-memory storage
* Concurrent safe. 
* Distributed cluster and high consistency. 

## TODO
+ [x] Stream support
+ [ ] RDB persistence
+ [ ] AOF persistence
+ [x] Cluster Mode(with [Raft Algorithm](https://raft.github.io/) and higher level of write safety than C-Redis)
+ [ ] Master-Slave

## Contribute Guide

It's highly recommanded to post an issue first and follow it by pull request. Feel free to post any valuable suggestions. 

To start contributing, try to implement a simple command thats not on board already or help parsing some additional arguments for an existing command. The code style should be consistent with existing codes. Reference to other commands when writing up your commits. 

You might want to find all redis commands [here](https://redis.io/commands/).  

## Usage
Build RedisGO from source code:
```bash
$ go build -o RedisGO main.go
```
Start RedisGO server:
```bash
$ ./RedisGO
[info][server.go:26] 2022/09/08 13:23:50 [Server Listen at  127.0.0.1 : 6380]
```
Use start option commands or config file to change default settings:
```bash 
$ ./RedisGO -h
Usage of ./RedisGO:
  -config string
        Appoint a config file: such as /etc/redis.conf
  -host string
        Bind host ip: default is 127.0.0.1 (default "127.0.0.1")
  -logdir string
        Set log directory: default is /tmp (default "./")
  -loglevel string
        Set log level: default is info (default "info")
  -port int
        Bind a listening port: default is 6379 (default 6379)
```
## Communication with RedisGO server
Any redis client can communicate with RedisGO server.  
For example, use redis-cli to communicate with RedisGO server:

```bash
# start a RedisGO server listening at 12345 port
$ ./RedisGO -port 12345
[info][server.go:26] 2022/09/08 13:31:47 [Server Listen at  127.0.0.1 : 12345]
                      ...

# start a redis-cli and connect to RedisGO server
$ redis-cli -p 12345
127.0.0.1:12345> PING
PONG
127.0.0.1:12345> MSET key1 a key2 b
OK
127.0.0.1:12345> MGET key1 key2 nonekey
1) "a"
2) "b"
3) (nil)
127.0.0.1:12345> RPUSH list1 1 2 3 4 5
(integer) 5
127.0.0.1:12345> LRANGE list1 0 -1
1) "1"
2) "2"
3) "3"
4) "4"
5) "5"
127.0.0.1:12345> TYPE list1
list
127.0.0.1:12345> EXPIRE list1 100
(integer) 1
# wait for a few seconds
127.0.0.1:12345> TTL list1
(integer) 93
127.0.0.1:12345> PERSIST list1
(integer) 1
127.0.0.1:12345> TTL list1
(integer) -1
```

## Cluster Mode

Take a look at the files in `cluster_test` directory. 

You only need to provide peer addresses(IP & Port) and a few other settings through json file. 

To start a predefined 3 nodes cluster, run `cluster_test.sh` in `cluster_test` directory 
and connect to one of the ports in (6380, 6381, 6382) or many of them with your redis-cli. 
Then try out you favorite redis commands on different nodes. 

## Benchmark

Benchmark result is based on [redis-benchmark](https://redis.io/topics/benchmarks) tool.  
Testing on MacBook Pro 2021 with M1 pro, 16.0 GB RAM, and on macOS Monterey.

The first one is the RedisGO result and the second is from Redis.  
Note that this result could vary tremendously. Generally, we say we reach 80-90% of the original C Redis performance. 
`benchmark -c 50 -n 200000 -t [get|set|...] -q`

```text
SET: 176678.45   requests per second, p50=0.143 msec                    
GET: 187969.92 requests per second, p50=0.151 msec                    
INCR: 186741.36 requests per second, p50=0.135 msec                    
LPUSH: 173611.12 requests per second, p50=0.143 msec                    
RPUSH: 161943.31 requests per second, p50=0.143 msec                    
LPOP: 187265.92 requests per second, p50=0.135 msec                    
RPOP: 186915.88 requests per second, p50=0.135 msec                    
SADD: 186915.88 requests per second, p50=0.135 msec                    
HSET: 185873.61 requests per second, p50=0.143 msec                    
SPOP: 188501.42 requests per second, p50=0.135 msec                    
MSET (10 keys): 139275.77 requests per second, p50=0.199 msec    
zrange a 0 100 withscores: rps=161032.0 (overall: 152203.6) avg_msec=0.193
```

```text
SET: 185873.61 requests per second, p50=0.135 msec                    
GET: 185528.77 requests per second, p50=0.127 msec                    
INCR: 183992.64 requests per second, p50=0.135 msec                    
LPUSH: 205338.81 requests per second, p50=0.135 msec                    
RPUSH: 208116.55 requests per second, p50=0.135 msec                    
LPOP: 197238.64 requests per second, p50=0.135 msec                    
RPOP: 197628.47 requests per second, p50=0.135 msec                    
SADD: 201409.88 requests per second, p50=0.135 msec                    
HSET: 201409.88 requests per second, p50=0.135 msec                    
SPOP: 212765.95 requests per second, p50=0.127 msec                    
MSET (10 keys): 181323.66 requests per second, p50=0.199 msec 
zrange a 0 100 withscores: 165016.50 requests per second, p50=0.175 msec
```

## Support Commands
All commands are used as [redis commands](https://redis.io/commands/). You can use any Redis client to communicate with RedisGO.



| key     | string      | list   | set         | hash         | channels  | sorted set | Stream  |  
|---------|-------------|--------|-------------|--------------|-----------|------------|---------|
| del     | set         | llen   | sadd        | hdel         | subscribe | zadd       | xadd*   |
| exists  | get         | lindex | scard       | hexists      | publish   | zrange     | xrange* |
| keys    | getrange    | lpos   | sdiff       | hget         |           |            |         |
| expire  | setrange    | lpop   | sdirrstore  | hgetall      |           |            |         |
| persist | mget        | rpop   | sinter      | hincrby      |           |            |         |
| ttl     | mset        | lpush  | sinterstore | hincrbyfloat |           |            |         |
| type    | setex       | lpushx | sismember   | hkeys        |           |            |         |
| rename  | setnx       | rpush  | smembers    | hlen         |           |            |         |
|         | strlen      | rpushx | smove       | hmget        |           |            |         |
|         | incr        | lset   | spop        | hset         |           |            |         |
|         | incrby      | lrem   | srandmember | hsetnx       |           |            |         |
|         | decr        | ltrim  | srem        | hvals        |           |            |         |
|         | decrby      | lrange | sunion      | hstrlen      |           |            |         |
|         | incrbyfloat | lmove  | sunionstore | hrandfield   |           |            |         |
|         | append      | blpop  |             |              |           |            |         |
|         |             | brpop  |             |

*means partially implemented or is being worked on.

#### PS:
* **Lexicographical ranges**(BYLEX) in ZRANGE should never be used since it is barely useful and too complex to implement.
you can easily sort your result in lexicographical order in your code and please don't let Redis do it for you. This option also implicitly assumes all members have the same score, which might be misleading and confusing. 
If members have different scores the reply will be unspecified. 
