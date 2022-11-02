package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"strconv"
	"strings"
)

func xadd(ctx context.Context, m *MemDb, cmd [][]byte, _ net.Conn) resp.RedisData {
	if len(cmd) < 5 {
		return resp.MakeWrongNumberArgs("xadd")
	}
	// parse args
	var key = string(cmd[1])
	var idx = 2
	var nomkstream, maxlen, minid, wave, limit bool
	var threshold int
	var IDThreshold = &StreamID{
		time:   -1,
		seqNum: -1,
	}
	var isDone bool
	var err error
	var ID = &StreamID{
		time:   -1,
		seqNum: -1,
	}
	// parse args
	for {
		switch strings.ToLower(string(cmd[idx])) {
		case "nomkstream":
			idx++
			nomkstream = true
		case "maxlen":
			maxlen = true
			idx++
			// parse = (exact trim) or ~ (approximately trim)
			if string(cmd[idx]) == "~" {
				idx++
			} else if string(cmd[idx]) == "=" {
				idx++
			}
			// parse a following integer
			threshold, err = strconv.Atoi(string(cmd[idx]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
			if threshold < 0 {
				return resp.MakeErrorData("ERR The MAXLEN argument must be >= 0.")
			}
			idx++
		case "minid":
			minid = true
			idx++
			idStr := string(cmd[idx])
			// optional = or ~ following minid
			if idStr == "=" || idStr == "~" {
				idx++
				idStr = string(cmd[idx])
			}
			// complete ID
			if strings.Contains(idStr, "-") {
				trunks := strings.Split(idStr, "-")
				if len(trunks) != 2 {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				stamp, err := strconv.ParseInt(trunks[0], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				IDThreshold.time = stamp
				seqNum, err := strconv.ParseInt(trunks[1], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				IDThreshold.seqNum = seqNum
			} else {
				// incomplete ID
				stamp, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				IDThreshold.time = stamp
			}
		case "limit":
			limit = true
			idx++
			_, err = strconv.Atoi(string(cmd[idx]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
			idx++
		case "~":
			wave = true
		default:
			// parse ID or determine auto ID here
			if string(cmd[idx]) != "*" {
				trunks := strings.Split(string(cmd[idx]), "-")
				if len(trunks) != 2 {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				stamp, err := strconv.ParseInt(trunks[0], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				ID.time = stamp
				seqNum, err := strconv.ParseInt(trunks[1], 10, 64)
				if err != nil {
					return resp.MakeErrorData("ERR Invalid stream ID specified as stream command argument")
				}
				ID.seqNum = seqNum
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
	// check invalid combinations
	if maxlen && minid {
		return resp.MakeErrorData("ERR syntax error, MAXLEN and MINID options at the same time are not compatible")
	}
	if limit && !wave {
		return resp.MakeErrorData("ERR syntax error, LIMIT cannot be used without the special ~ option")
	}
	// lock the key
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	// get value pairs
	kvPairsBytes := cmd[idx:]
	var kvPairs = make([]string, 0, len(kvPairsBytes))
	for _, i := range kvPairsBytes {
		kvPairs = append(kvPairs, string(i))
	}
	// broken pairs
	if len(kvPairs)%2 != 0 {
		return resp.MakeWrongNumberArgs("xadd")
	}

	var tmp any
	var ok bool
	var stream *Stream
	// key doesn't exist
	if tmp, ok = m.db.Get(key); !ok {
		// option: don't create stream if key doesn't exist
		if nomkstream {
			return resp.MakeBulkData(nil)
		}
		stream = NewStream()
		err := stream.AddEntry(ID, kvPairs)
		if err != nil {
			return resp.MakeErrorData("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
		m.db.Set(key, stream)
	} else {
		// key exist
		stream, ok = tmp.(*Stream)
		if !ok {
			return resp.MakeWrongType()
		}
		err := stream.AddEntry(ID, kvPairs)
		if err != nil {
			return resp.MakeErrorData("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
	}
	// need to perform xtrim
	if maxlen && len(stream.timeStamps) > threshold {
		stream.DropFirstN(len(stream.timeStamps) - threshold)
	}
	if minid {
		// perform a trim. Drop first N earlier entries
		// O(n) linear search. Could be optimized later
		// todo: optimize this search
		count := 0
		for _, id := range stream.timeStamps {
			if id.GreaterEqual(ID) {
				stream.DropFirstN(count)
			} else {
				count++
			}
		}
	}
	if wave {
		// nearly exact trimming is not useful in our implement.
		// We will always perform a exact trim now.
	}
	return resp.MakeBulkData(resp.MakeStringData(ID.Format()).ByteData())
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
		for i := 0; i < len(ids); i++ {
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
