package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
)

func RegisterPubSubCommands() {
	RegisterCommand("subscribe", subscribe)
	RegisterCommand("publish", publish)
}

func subscribe(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) < 2 {
		return resp.MakeErrorData("ERR wrong number of arguments for 'subscribe' command")
	}
	// all subscribe keys
	keys := make([]string, 0, len(cmd)-1)
	for _, b := range cmd[1:] {
		keys = append(keys, string(b))
	}
	agg := make(chan *ChanMsg)

	// block and receive
	for _, key := range keys {
		out, ID := m.SubChans.Subscribe(key)
		// forwards all out channel to a single agg chan
		go func(key string) {
			for msg := range out {
				agg <- msg
			}
			// Unsubscribe if the channel is closed
			m.SubChans.UnSubscribe(key, ID)
		}(key)
	}
	for _, key := range keys {
		conn.Write(resp.MakeArrayData([]resp.RedisData{resp.MakeStringData("subscribe"),
			resp.MakeStringData(key), resp.MakeIntData(int64(1))}).ToBytes())
	}
	for {
		select {
		// receive from channel
		case msg := <-agg:
			conn.Write(resp.MakeArrayData([]resp.RedisData{resp.MakeStringData(msg.info),
				resp.MakeStringData(msg.key),
				resp.MakeStringData(msg.val.(string)),
			}).ToBytes())
		}
	}
}

func publish(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) != 3 {
		return resp.MakeErrorData("ERR wrong number of arguments for 'publish' command")
	}
	key := string(cmd[1])
	val := string(cmd[2])
	numSubs := m.SubChans.Send(key, val)
	return resp.MakeIntData(int64(numSubs))
}
