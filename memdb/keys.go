package memdb

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/resp"
	"github.com/innovationb1ue/RedisGO/util"
)

// keys.go file implements the keys commands of redis

func delKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "del" {
		logger.Error("delKey Function: cmdName is not del")
		return resp.MakeErrorData("Protocol error: cmdName is not del")
	}
	if !m.CheckTTL(string(cmd[1])) {
		return resp.MakeIntData(int64(0))
	}
	dKey := 0
	for _, key := range cmd[1:] {
		m.locks.Lock(string(key))
		dKey += m.db.Delete(string(key))
		m.ttlKeys.Delete(string(key))
		m.locks.UnLock(string(key))
	}
	return resp.MakeIntData(int64(dKey))
}

func existsKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "exists" || len(cmd) < 2 {
		logger.Error("existsKey Function: cmdName is not exists or command args number is invalid")
		return resp.MakeErrorData("Protocol error: cmdName is not exists")
	}
	eKey := 0
	var key string
	for _, keyByte := range cmd[1:] {
		key = string(keyByte)
		if m.CheckTTL(key) {
			m.locks.RLock(key)
			if _, ok := m.db.Get(key); ok {
				eKey++
			}
			m.locks.RUnLock(key)
		}
	}
	return resp.MakeIntData(int64(eKey))
}

func keysKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "keys" || len(cmd) != 2 {
		logger.Error("keysKey Function: cmdName is not keys or cmd length is not 2")
		return resp.MakeErrorData(fmt.Sprintf("error: keys function get invalid command %s %s", string(cmd[0]), string(cmd[1])))
	}
	res := make([]resp.RedisData, 0)
	allKeys := m.db.Keys()
	pattern := string(cmd[1])
	for _, key := range allKeys {
		if m.CheckTTL(key) {
			if util.PattenMatch(pattern, key) {
				res = append(res, resp.MakeBulkData([]byte(key)))
			}
		}
	}
	return resp.MakeArrayData(res)
}

func expireKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "expire" || len(cmd) < 3 || len(cmd) > 4 {
		logger.Error("expireKey Function: cmdName is not expire or command args number is invalid")
		return resp.MakeErrorData("error: cmdName is not expire or command args number is invalid")
	}

	v, err := strconv.ParseInt(string(cmd[2]), 10, 64)
	if err != nil {
		logger.Error("expireKey Function: cmd[2] %s is not int", string(cmd[2]))
		return resp.MakeErrorData(fmt.Sprintf("error: %s is not int", string(cmd[2])))
	}
	ttl := time.Now().Unix() + v
	var opt string
	if len(cmd) == 4 {
		opt = strings.ToLower(string(cmd[3]))
	}
	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeIntData(int64(0))
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	var res int
	switch opt {
	case "nx":
		if _, ok := m.ttlKeys.Get(key); !ok {
			res = m.SetTTL(key, ttl)
		}
	case "xx":
		if _, ok := m.ttlKeys.Get(key); ok {
			res = m.SetTTL(key, ttl)
		}
	case "gt":
		if v, ok := m.ttlKeys.Get(key); ok && ttl > v.(int64) {
			res = m.SetTTL(key, ttl)
		}
	case "lt":
		if v, ok := m.ttlKeys.Get(key); ok && ttl < v.(int64) {
			res = m.SetTTL(key, ttl)
		}
	default:
		if opt != "" {
			logger.Error("expireKey Function: opt %s is not nx, xx, gt or lt", opt)
			return resp.MakeErrorData(fmt.Sprintf("ERR Unsupported option %s, except nx, xx, gt, lt", opt))
		}
		res = m.SetTTL(key, ttl)
	}
	return resp.MakeIntData(int64(res))
}

func persistKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "persist" || len(cmd) != 2 {
		logger.Error("persistKey Function: cmdName is not persist or command args number is invalid")
		return resp.MakeErrorData("error: cmdName is not persist or command args number is invalid")
	}
	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeIntData(int64(0))
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	res := m.DelTTL(key)
	return resp.MakeIntData(int64(res))
}

func ttlKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "ttl" || len(cmd) != 2 {
		logger.Error("ttlKey Function: cmdName is not ttl or command args number is invalid")
		return resp.MakeErrorData("error: cmdName is not ttl or command args number is invalid")
	}
	key := string(cmd[1])

	if !m.CheckTTL(key) {
		return resp.MakeIntData(int64(-2))
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)
	if _, ok := m.db.Get(key); !ok {
		return resp.MakeIntData(int64(-2))
	}
	now := time.Now().Unix()
	ttl, ok := m.ttlKeys.Get(key)
	if !ok {
		return resp.MakeIntData(int64(-1))
	}
	return resp.MakeIntData(ttl.(int64) - now)
}

func typeKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "type" || len(cmd) != 2 {
		logger.Error("typeKey Function: cmdName is not type or command args number is invalid")
		return resp.MakeErrorData("error: cmdName is not type or command args number is invalid")
	}
	key := string(cmd[1])

	if !m.CheckTTL(key) {
		return resp.MakeBulkData([]byte("none"))
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)
	v, ok := m.db.Get(key)
	if !ok {
		return resp.MakeStringData("none")
	}
	switch v.(type) {
	case []byte:
		return resp.MakeStringData("string")
	case *List:
		return resp.MakeStringData("list")
	case *Set:
		return resp.MakeStringData("set")
	case *Hash:
		return resp.MakeStringData("hash")
	default:
		logger.Error("typeKey Function: type func error, not in string|list|set|hash")
	}
	return resp.MakeErrorData("unknown error: server error")
}

func renameKey(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	cmdName := string(cmd[0])
	if strings.ToLower(cmdName) != "rename" || len(cmd) != 3 {
		logger.Error("renameKey Function: cmdName is not rename or command args number is invalid")
		return resp.MakeErrorData("error: cmdName is not rename or command args number is invalid")
	}
	oldName, newName := string(cmd[1]), string(cmd[2])

	if !m.CheckTTL(oldName) {
		return resp.MakeErrorData(fmt.Sprintf("error: %s not exist", oldName))
	}

	m.locks.LockMulti([]string{oldName, newName})
	defer m.locks.UnLockMulti([]string{oldName, newName})

	oldValue, ok := m.db.Get(oldName)
	if !ok {
		return resp.MakeErrorData(fmt.Sprintf("error: %s not exist", oldName))
	}
	m.db.Delete(oldName)
	m.ttlKeys.Delete(oldName)
	m.db.Delete(newName)
	m.ttlKeys.Delete(newName)
	m.db.Set(newName, oldValue)
	return resp.MakeStringData("OK")
}

func pingKeys(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) > 2 {
		return resp.MakeErrorData("error: wrong number of arguments for 'ping' command")
	}
	// default reply
	if len(cmd) == 1 {
		return resp.MakeStringData("PONG")
	}
	// echo replay
	return resp.MakeBulkData(cmd[1])
}

func RegisterKeyCommands() {
	RegisterCommand("ping", pingKeys)
	RegisterCommand("del", delKey)
	RegisterCommand("exists", existsKey)
	RegisterCommand("keys", keysKey)
	RegisterCommand("expire", expireKey)
	RegisterCommand("persist", persistKey)
	RegisterCommand("ttl", ttlKey)
	RegisterCommand("type", typeKey)
	RegisterCommand("rename", renameKey)
}
