clicmd1="redis-cli -p 6379"
clicmd2="redis-cli -p 6380"

echo flushall | eval "$clicmd1"
echo flushall | eval "$clicmd2"
echo zadd a 1 a 2 b 3 c 40 d 50 f |eval  "$clicmd1"
echo zadd a 1 a 2 b 3 c 40 d 50 f |eval  "$clicmd2"

redis-benchmark -q -p 6379 zrange a 0 100 withscores
redis-benchmark -q -p 6380 zrange a 0 100 withscores

