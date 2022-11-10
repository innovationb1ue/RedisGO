go build ..
./RedisGO --IsCluster=true --ClusterConfigPath="./cluster_config.json" --config="./redis.conf" &
./RedisGO --IsCluster=true --ClusterConfigPath="./cluster_config1.json" --config="./redis1.conf" &
./RedisGO --IsCluster=true --ClusterConfigPath="./cluster_config2.json" --config="./redis2.conf"