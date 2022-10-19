package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
)

func RegisterPubSubCommands() {
	RegisterCommand("subscribe", subscribe)
	RegisterCommand("publish", publish)
}

func subscribe(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) < 2 {
		return resp.MakeWrongNumberArgs("subscribe")
	}
	// get all subscribe keys
	keys := make([]string, 0, len(cmd)-1)
	for _, b := range cmd[1:] {
		keys = append(keys, string(b))
	}

	// subscribe channels & start forwarding msg to aggregate
	for _, key := range keys {
		// add TCP connection to publish pool
		ID := m.SubChans.Subscribe(key, conn)
		// register the self-unsubscribe service
		go func(key string) {
			for {
				select {
				case <-ctx.Done():
					// Unsubscribe if the client is leaving
					m.SubChans.UnSubscribe(key, ID)
					return
				}
			}
		}(key)
	}
	// return initial subscribe success message
	// this implicitly assume all subscription are successful since no way they should fail.
	res := make([]resp.RedisData, 0, len(keys))
	for _, key := range keys {
		res = append(res, resp.MakeBulkData([]byte("subscribe")),
			resp.MakeBulkData([]byte(key)), resp.MakeIntData(int64(1)))
	}
	return resp.MakeArrayData(res)
}

func publish(ctx context.Context, m *MemDb, cmd [][]byte, _ net.Conn) resp.RedisData {
	if len(cmd) != 3 {
		return resp.MakeWrongNumberArgs("publish")
	}
	key := string(cmd[1])
	val := string(cmd[2])
	numSubs := m.SubChans.Send(key, val)
	return resp.MakeIntData(int64(numSubs))
}
