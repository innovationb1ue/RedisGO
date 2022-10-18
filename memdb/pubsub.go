package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"log"
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
	// aggregation channel. All subscribe channels will be forwarded to this channel
	aggregate := make(chan *ChanMsg)

	// subscribe channels & start forwarding msg to aggregate
	for _, key := range keys {
		out, ID := m.SubChans.Subscribe(key)
		// forwards all out channel to a single aggregate chan
		go func(key string) {
			defer log.Println("quit forwaring goroutine")
			for {
				select {
				case msg := <-out:
					// this will not block forever since we will drain the aggregate channel before closing this context
					aggregate <- msg
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
	for _, key := range keys {
		_, err := conn.Write(resp.MakeArrayData([]resp.RedisData{resp.MakeBulkData([]byte("subscribe")),
			resp.MakeBulkData([]byte(key)), resp.MakeIntData(int64(1))}).ToBytes())
		if err != nil {
			return nil
		}
	}
	// infinite loop: receive PUB messages and write to Conn
	for {
		select {
		// receive from channel
		case msg := <-aggregate:
			_, err := conn.Write(resp.MakeArrayData([]resp.RedisData{resp.MakeBulkData([]byte(msg.info)),
				resp.MakeBulkData([]byte(msg.key)),
				resp.MakeBulkData([]byte(msg.val.(string))),
			}).ToBytes())
			// clients leaving. break the event loop
			// todo: bug here. when client is not listening this wont immediately trigger.
			if err != nil {
				// drain the messages in aggregate. This prevents the forwarding goroutine from blocking forever
				for range aggregate {
				}
				return nil
			}
		}
	}
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
