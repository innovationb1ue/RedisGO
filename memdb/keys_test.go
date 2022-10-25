package memdb

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/innovationb1ue/RedisGO/config"
)

func init() {
	config.Configures = &config.Config{ShardNum: 100}
}

func TestDelKey(t *testing.T) {
	memdb := NewMemDb()
	memdb.db.Set("a", "a")
	memdb.db.Set("b", "b")
	memdb.ttlKeys.Set("b", time.Now().Unix()+10)
	ctx := context.Background()
	del_a := delKey(ctx, memdb, [][]byte{[]byte("del"), []byte("a"), []byte("b")}, nil)

	if !bytes.Equal(del_a.ToBytes(), []byte(":2\r\n")) {
		t.Error("del reply is not correct")
	}

	_, ok1 := memdb.db.Get("a")
	_, ok2 := memdb.db.Get("b")
	_, ok3 := memdb.ttlKeys.Get("b")
	if ok1 || ok2 || ok3 {
		t.Error("del failed")
	}
}

func TestExpireKey(t *testing.T) {
	memdb := NewMemDb()
	memdb.db.Set("a", "a")
	memdb.db.Set("b", "b")
	ctx := context.Background()
	expire_a := expireKey(ctx, memdb, [][]byte{[]byte("expire"), []byte("a"), []byte("100"), []byte("nx")}, nil)
	if !bytes.Equal(expire_a.ToBytes(), []byte(":1\r\n")) {
		t.Error("expire reply is not correct")
	}
	attl, _ := memdb.ttlKeys.Get("a")
	if attl.(*TTLInfo).value-time.Now().Unix() > 100 || attl.(*TTLInfo).value-time.Now().Unix() < 99 {
		t.Error("ttl set incorrect")
	}
	expire_a1 := expireKey(ctx, memdb, [][]byte{[]byte("expire"), []byte("a"), []byte("1000"), []byte("xx")}, nil)
	if !bytes.Equal(expire_a1.ToBytes(), []byte(":1\r\n")) {
		t.Error("expire reply is not correct")
	}
	a1ttl, _ := memdb.ttlKeys.Get("a")
	if a1ttl.(*TTLInfo).value-time.Now().Unix() > 1000 || a1ttl.(*TTLInfo).value-time.Now().Unix() < 999 {
		t.Error("ttl set incorrect")
	}

	expire_b := expireKey(ctx, memdb, [][]byte{[]byte("expire"), []byte("b"), []byte("100")}, nil)
	if !bytes.Equal(expire_b.ToBytes(), []byte(":1\r\n")) {
		t.Error("expire reply is not correct")
	}
	bttl, _ := memdb.ttlKeys.Get("b")
	if bttl.(*TTLInfo).value-time.Now().Unix() > 100 || bttl.(*TTLInfo).value-time.Now().Unix() < 99 {
		t.Error("ttl set incorrect")
	}

	expire_b1 := expireKey(ctx, memdb, [][]byte{[]byte("expire"), []byte("b"), []byte("1000"), []byte("gt")}, nil)
	if !bytes.Equal(expire_b1.ToBytes(), []byte(":1\r\n")) {
		t.Error("expire reply is not correct")
	}
	b1ttl, _ := memdb.ttlKeys.Get("b")
	if b1ttl.(*TTLInfo).value-time.Now().Unix() > 1000 || b1ttl.(*TTLInfo).value-time.Now().Unix() < 999 {
		t.Error("ttl set incorrect")
	}
}
