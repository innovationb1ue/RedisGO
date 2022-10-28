package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"log"
	"net"
	"strings"
)

func xadd(ctx context.Context, m *MemDb, cmd [][]byte, _ net.Conn) resp.RedisData {
	if len(cmd) < 5 {
		return resp.MakeWrongNumberArgs("xadd")
	}
	var key = string(cmd[1])
	var idx = 2
	var stream any
	var ok bool
	switch strings.ToLower(string(cmd[idx])) {
	case "nomkstream":
		stream, ok = m.db.Get(key)
		if !ok {
			return nil
		}
		log.Println(stream)
	}
	return nil

}

func RegisterStreamCommands() {
	RegisterCommand("xadd", xadd)

}
