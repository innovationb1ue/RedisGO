
i="GET,SET,INCR,DECR,LPUSH,RPUSH,LPOP,RPOP,SADD,HSET,SPOP,MSET"
redis-benchmark -c 50 -n 200000 -t "$i" -q -p 6380
redis-benchmark -c 50 -n 200000 -t "$i" -q -p 6379






