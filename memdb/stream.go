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
	var key = string(cmd[1])
	var idx = 2
	var nomkstream, maxlen, minid, equal, wave, limit bool
	var threshold, count int
	var isDone bool
	var err error
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
			isDone = true
			break
		}
		// break out of infinite for loop
		if isDone {
			break
		}
	}

	return nil

}

func RegisterStreamCommands() {
	RegisterCommand("xadd", xadd)

}
