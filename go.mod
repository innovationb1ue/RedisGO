module github.com/innovationb1ue/RedisGO

go 1.19

require (
	github.com/google/uuid v1.3.0
	go.etcd.io/etcd/raft/v3 v3.5.5
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	go.etcd.io/etcd/api/v3 v3.6.0-alpha.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

// Bad imports are sometimes causing attempts to pull that code.
// This makes the error more explicit.
replace go.etcd.io/etcd => ./FORBIDDEN_DEPENDENCY

replace go.etcd.io/etcd/v3 => ./FORBIDDEN_DEPENDENCY

replace go.etcd.io/etcd/raft/v3 v3.5.5 => ./raft
