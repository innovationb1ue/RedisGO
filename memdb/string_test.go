package memdb

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/innovationb1ue/RedisGO/config"
)

func init() {
	config.Configures = &config.Config{
		ShardNum: 100,
	}
}

func TestSetString(t *testing.T) {
	mem := NewMemDb()
	ctx := context.Background()
	// test set
	res := setString(ctx, mem, [][]byte{[]byte("set"), []byte("a"), []byte("a")}, nil)
	if !bytes.Equal(res.ToBytes(), []byte("+OK\r\n")) {
		t.Error("set reply error")
	}
	val, ok := mem.db.Get("a")
	if !ok || !bytes.Equal(val.([]byte), []byte("a")) {
		t.Error("set value error")
	}

	// test opt xx and ex
	res = setString(ctx, mem, [][]byte{[]byte("set"), []byte("a"), []byte("b"), []byte("xx"), []byte("ex"), []byte("100")}, nil)
	if !bytes.Equal(res.ToBytes(), []byte("+OK\r\n")) {
		t.Error("set reply error")
	}
	val, ok = mem.db.Get("a")
	if !ok || !bytes.Equal(val.([]byte), []byte("b")) {
		t.Error("set value error")
	}
	ttl, ok := mem.ttlKeys.Get("a")
	if !ok || ttl.(*TTLInfo).value-time.Now().Unix() > 100 || ttl.(*TTLInfo).value-time.Now().Unix() < 99 {
		t.Error("set ttl error")
	}

	// test opt keepttl
	res = setString(ctx, mem, [][]byte{[]byte("set"), []byte("a"), []byte("c"), []byte("get"), []byte("keepttl")}, nil)
	if !bytes.Equal(res.ToBytes(), []byte("$1\r\nb\r\n")) {
		t.Error("set reply error")
	}
	_, ok = mem.ttlKeys.Get("a")
	if !ok {
		t.Error("set keepttl error")
	}
}
