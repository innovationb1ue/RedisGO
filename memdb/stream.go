package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func xadd(ctx context.Context, m *MemDb, cmd [][]byte, _ net.Conn) resp.RedisData {
	if len(cmd) < 5 {
		return resp.MakeWrongNumberArgs("xadd")
	}
	// parse args
	var key = string(cmd[1])
	var idx = 2
	var nomkstream, maxlen, minid, equal, wave, limit, autoID bool
	var threshold, count int
	var isDone bool
	var err error
	var ID int64
	var seqNum int64
	for {
		switch strings.ToLower(string(cmd[idx])) {
		case "nomkstream":
			idx++
			nomkstream = true
		case "maxlen":
			maxlen = true
			idx++
			if string(cmd[idx]) == "~" {
				wave = true
				idx++
				threshold, err = strconv.Atoi(string(cmd[idx]))
				if err != nil {
					return resp.MakeErrorData("ERR value is not an integer or out of range")
				}
				idx++
			}
			if string(cmd[idx]) == "=" {
				equal = true
				idx++
				threshold, err = strconv.Atoi(string(cmd[idx]))
				if err != nil {
					return resp.MakeErrorData("ERR value is not an integer or out of range")
				}
				idx++
			}
			// default case, follow by a number
			threshold, err = strconv.Atoi(string(cmd[idx]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
			idx++
		case "minid":
			minid = true
			idx++
		case "limit":
			limit = true
			idx++
			count, err = strconv.Atoi(string(cmd[idx]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
		default:
			// parse ID or determine auto ID here
			if string(cmd[idx]) == "*" {
				ID = time.Now().UnixMilli()
				seqNum = -1
			} else {
				trunks := strings.Split(string(cmd[idx]), "-")
				if len(trunks) != 2 {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				ID, err = strconv.ParseInt(trunks[0], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				seqNum, err = strconv.ParseInt(trunks[1], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
			}
			isDone = true
			idx++
			break
		}
		// break out of infinite for loop
		if isDone {
			break
		}
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	// get value pairs
	kvPairsBytes := cmd[idx:]
	var kvPairs = make([]string, 0, len(kvPairsBytes))
	for _, i := range kvPairsBytes {
		kvPairs = append(kvPairs, string(i))
	}
	log.Println("Got value pairs ", kvPairs)

	var tmp any
	var ok bool
	// key doesn't exist
	if tmp, ok = m.db.Get(key); !ok {
		stream := NewStream()
		if autoID {
			ID = time.Now().UnixMilli()
			seqNum = -1
		}
		err := stream.AddEntry(ID, seqNum, kvPairs)
		if err != nil {
			return resp.MakeErrorData("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
		m.db.Set(key, stream)
	} else {
		// key exist
		stream, ok := tmp.(*Stream)
		if !ok {
			return resp.MakeWrongType()
		}
		err := stream.AddEntry(time.Now().UnixMilli(), 0, kvPairs)
		if err != nil {
			return resp.MakeErrorData("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
	}

	if nomkstream {
	}
	if maxlen {
	}
	if minid {
	}
	if equal {
	}
	if wave {
	}
	if limit {
	}
	println(threshold, count)
	return resp.MakeArrayData([]resp.RedisData{})
}

func xrange(ctx context.Context, m *MemDb, cmd [][]byte, _ net.Conn) resp.RedisData {
	if len(cmd) < 4 {
		return resp.MakeWrongNumberArgs("xrange")
	}
	key := string(cmd[1])
	var start, end *StreamID
	if string(cmd[2]) == "-" {
		start = &StreamID{
			time:   -1,
			seqNum: -1,
		}
	}
	if string(cmd[3]) == "+" {
		end = &StreamID{
			time:   -1,
			seqNum: -1,
		}
	}
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	var stream *Stream
	tmp, ok := m.db.Get(key)
	// key doesn't exist
	if !ok {
		stream = NewStream()
		m.db.Set(key, stream)
	} else {
		// key exist, assert it is a stream
		stream, ok = tmp.(*Stream)
		if !ok {
			return resp.MakeWrongType()
		}
	}
	ids, entries := stream.Range(start, end)
	if ids != nil && entries != nil && len(ids) == len(entries) {
		res := make([]resp.RedisData, 0, len(ids))
		for i := 1; i < len(ids); i++ {
			idData := resp.MakeStringData(ids[i].Format())
			entriesData := make([]resp.RedisData, 0, len(entries[i]))
			for _, val := range entries[i] {
				entriesData = append(entriesData, resp.MakeStringData(val))
			}
			entriesArr := resp.MakeArrayData(entriesData)
			res = append(res, resp.MakeArrayData([]resp.RedisData{idData, entriesArr}))
		}
		return resp.MakeArrayData(res)
	}
	return nil
}

func RegisterStreamCommands() {
	RegisterCommand("xadd", xadd)
	RegisterCommand("xrange", xrange)
}
