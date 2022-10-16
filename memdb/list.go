package memdb

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/resp"
)

// list.go file implements the list commands of redis

func lLenList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "llen" {
		logger.Error("lLenList Function: cmdName is not llen")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) != 2 {
		return resp.MakeErrorData("wrong number of arguments for 'llen' command")
	}

	key := string(cmd[1])

	if !m.CheckTTL(key) {
		return resp.MakeIntData(0)
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)

	v, ok := m.db.Get(key)
	if !ok {
		return resp.MakeIntData(0)
	}

	typeV, ok := v.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return resp.MakeIntData(int64(typeV.Len))
}

func lIndexList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lindex" {
		logger.Error("lIndexList Function: cmdName is not lindex")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) != 3 {
		return resp.MakeErrorData("wrong number of arguments for 'lindex' command")
	}

	key := string(cmd[1])
	index, err := strconv.Atoi(string(cmd[2]))
	if err != nil {
		return resp.MakeErrorData("index is not an integer")
	}

	if !m.CheckTTL(key) {
		return resp.MakeBulkData(nil)
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)

	v, ok := m.db.Get(key)
	if !ok {
		return resp.MakeBulkData(nil)
	}

	typeV, ok := v.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	resNode := typeV.Index(index)
	if resNode == nil {
		return resp.MakeBulkData(nil)
	}
	return resp.MakeBulkData(resNode.Val)
}

func lPosList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lpos" {
		logger.Error("lPosList Function: cmdName is not lpos")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) < 3 || len(cmd)&1 != 1 {
		return resp.MakeErrorData("wrong number of arguments for 'lpos' command")
	}

	var rank, count, maxLen, reverse bool
	var rankVal, countVal, maxLenVal int
	var key string
	var elem []byte
	var err error

	var pos int

	key = string(cmd[1])
	elem = cmd[2]

	// handle params
	for i := 3; i < len(cmd); i += 2 {
		switch strings.ToLower(string(cmd[i])) {
		case "rank":
			rank = true
			rankVal, err = strconv.Atoi(string(cmd[i+1]))
			if err != nil || rankVal == 0 {
				return resp.MakeErrorData("rank value should 1,2,3... or -1,-2,-3...")
			}
		case "count":
			count = true
			countVal, err = strconv.Atoi(string(cmd[i+1]))
			if err != nil || countVal < 0 {
				return resp.MakeErrorData("count value is not an positive integer")
			}
		case "maxlen":
			maxLen = true
			maxLenVal, err = strconv.Atoi(string(cmd[i+1]))
			if err != nil || maxLenVal < 0 {
				return resp.MakeErrorData("maxlen value is not an positive integer")
			}
		default:
			return resp.MakeErrorData(fmt.Sprintf("unsupported option %s", string(cmd[i])))
		}
	}

	if !m.CheckTTL(key) {
		return resp.MakeBulkData(nil)
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)

	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeBulkData(nil)
	}

	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if list.Len == 0 {
		return resp.MakeBulkData(nil)
	}

	if count && countVal == 0 {
		countVal = list.Len
	}

	if maxLen && maxLenVal == 0 {
		maxLenVal = list.Len
	}

	// normally pos without options
	if !rank && !count && !maxLen {
		pos := list.Pos(elem)
		if pos == -1 {
			return resp.MakeBulkData(nil)
		} else {
			return resp.MakeIntData(int64(pos))
		}
	}

	// handle options
	var now *ListNode
	if rank {
		if rankVal > 0 {
			pos = -1
			for now = list.Head.Next; now != list.Tail; now = now.Next {
				pos++
				if bytes.Equal(now.Val, elem) {
					rankVal--
				}
				if maxLen {
					maxLenVal--
					if maxLenVal == 0 {
						break
					}
				}
				if rankVal == 0 {
					break
				}
			}
		} else {
			reverse = true
			pos = list.Len
			for now = list.Tail.Prev; now != list.Head; now = now.Prev {
				pos--
				if bytes.Equal(now.Val, elem) {
					rankVal++
				}
				if maxLen {
					maxLenVal--
					if maxLenVal == 0 {
						break
					}
				}
				if rankVal == 0 {
					break
				}
			}
		}
	} else {
		now = list.Head.Next
		pos = 0
		if maxLen {
			maxLenVal--
		}
	}

	// when rank is out of range, return nil
	if (rank && rankVal != 0) || now == list.Tail || now == list.Head {
		return resp.MakeBulkData(nil)
	}

	res := make([]resp.RedisData, 0)
	if !count {
		// if count is not set, return first find pos inside maxLen range
		for ; now != list.Tail; now = now.Next {
			if bytes.Equal(now.Val, elem) {
				return resp.MakeIntData(int64(pos))
			}
			pos++
			if maxLen {
				if maxLenVal <= 0 {
					break
				}
				maxLenVal--
			}
		}
		return resp.MakeBulkData(nil)
	} else {
		if !reverse {
			for ; now != list.Tail && countVal != 0; now = now.Next {
				if bytes.Equal(now.Val, elem) {
					res = append(res, resp.MakeIntData(int64(pos)))
					countVal--
				}
				pos++
				if maxLen {
					if maxLenVal <= 0 {
						break
					}
					maxLenVal--
				}
			}
		} else {
			for ; now != list.Head && countVal != 0; now = now.Prev {
				if bytes.Equal(now.Val, elem) {
					res = append(res, resp.MakeIntData(int64(pos)))
					countVal--
				}
				pos--
				if maxLen {
					if maxLenVal <= 0 {
						break
					}
					maxLenVal--
				}
			}
		}
	}
	if len(res) == 0 {
		return resp.MakeBulkData(nil)
	}
	return resp.MakeArrayData(res)
}

func lPopList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) != 2 && len(cmd) != 3 {
		return resp.MakeErrorData("wrong number of arguments for 'lpop' command")
	}

	var cnt int
	var err error
	if len(cmd) == 3 {
		cnt, err = strconv.Atoi(string(cmd[2]))
		if err != nil || cnt <= 0 {
			return resp.MakeErrorData("count value must be a positive integer")
		}
	}

	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeBulkData(nil)
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeBulkData(nil)
	}
	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// remove the key when list is empty
	defer func() {
		if list.Len == 0 {
			m.db.Delete(key)
			m.DelTTL(key)
		}
	}()

	// if cnt is not set, return first element
	if cnt == 0 {
		e := list.LPop()
		if e == nil {
			return resp.MakeBulkData(nil)
		}
		return resp.MakeBulkData(e.Val)
	}

	// return cnt number elements as array
	res := make([]resp.RedisData, 0)
	for i := 0; i < cnt; i++ {
		e := list.LPop()
		if e == nil {
			break
		}
		res = append(res, resp.MakeBulkData(e.Val))
	}
	return resp.MakeArrayData(res)
}

func rPopList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "rpop" {
		logger.Error("rPopList: command is not rpop")
		return resp.MakeErrorData("server error")
	}
	if len(cmd) != 2 && len(cmd) != 3 {
		return resp.MakeErrorData("wrong number of arguments for 'rpop' command")
	}

	var cnt int
	var err error
	if len(cmd) == 3 {
		cnt, err = strconv.Atoi(string(cmd[2]))
		if err != nil || cnt <= 0 {
			return resp.MakeErrorData("count value must be a positive integer")
		}
	}

	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeBulkData(nil)
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeBulkData(nil)
	}
	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	defer func() {
		if list.Len == 0 {
			m.db.Delete(key)
			m.DelTTL(key)
		}
	}()

	// if cnt is not set, return last element
	if cnt == 0 {
		e := list.RPop()
		if e == nil {
			return resp.MakeBulkData(nil)
		}
		return resp.MakeBulkData(e.Val)
	}

	// return cnt number elements as array
	res := make([]resp.RedisData, 0)
	for i := 0; i < cnt; i++ {
		e := list.RPop()
		if e == nil {
			break
		}
		res = append(res, resp.MakeBulkData(e.Val))
	}
	return resp.MakeArrayData(res)
}

func lPushList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lpush" {
		logger.Error("lPushList Function : cmdName is not lpush")
		return resp.MakeErrorData("Server Error")
	}
	if len(cmd) < 3 {
		return resp.MakeErrorData("wrong number of arguments for 'lpush' command")
	}

	key := string(cmd[1])
	m.CheckTTL(key)

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	var list *List
	tem, ok := m.db.Get(key)
	// no such key, create a list
	if !ok {
		list = NewList()
		m.db.Set(key, list)
	} else {
		// try to assert it as a List.
		list, ok = tem.(*List)
		if !ok {
			return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}
	// push args to the list
	for i := 2; i < len(cmd); i++ {
		list.LPush(cmd[i])
	}
	// return the length of the list
	return resp.MakeIntData(int64(list.Len))
}

func lPushXList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lpushx" {
		logger.Error("lPushXList Function : cmdName is not lpushx")
		return resp.MakeErrorData("Server Error")
	}
	if len(cmd) < 3 {
		return resp.MakeErrorData("wrong number of arguments for 'lpushx' command")
	}

	key := string(cmd[1])
	m.CheckTTL(key)

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	var list *List
	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeIntData(0)
	} else {
		list, ok = tem.(*List)
		if !ok {
			return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}
	for i := 2; i < len(cmd); i++ {
		list.LPush(cmd[i])
	}
	return resp.MakeIntData(int64(list.Len))
}

func rPushList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "rpush" {
		logger.Error("rPushList Function : cmdName is not rpush")
		return resp.MakeErrorData("server error")
	}
	if len(cmd) < 3 {
		return resp.MakeErrorData("wrong number of arguments for 'rpush' command")
	}

	key := string(cmd[1])
	m.CheckTTL(key)

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	var list *List
	tem, ok := m.db.Get(key)
	if !ok {
		list = NewList()
		m.db.Set(key, list)
	} else {
		list, ok = tem.(*List)
		if !ok {
			return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}
	for i := 2; i < len(cmd); i++ {
		list.RPush(cmd[i])
	}
	return resp.MakeIntData(int64(list.Len))
}

func rPushXList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "rpushx" {
		logger.Error("rPushXList Function : cmdName is not rpushx")
		return resp.MakeErrorData("server error")
	}
	if len(cmd) < 3 {
		return resp.MakeErrorData("wrong number of arguments for 'rpushX' command")
	}

	key := string(cmd[1])
	m.CheckTTL(key)

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	var list *List
	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeIntData(0)
	} else {
		list, ok = tem.(*List)
		if !ok {
			return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}
	for i := 2; i < len(cmd); i++ {
		list.RPush(cmd[i])
	}
	return resp.MakeIntData(int64(list.Len))
}

func lSetList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lset" {
		logger.Error("lSetList Function : cmdName is not lset")
		return resp.MakeErrorData("server error")
	}
	if len(cmd) != 4 {
		return resp.MakeErrorData("wrong number of arguments for 'lset' command")
	}

	index, err := strconv.Atoi(string(cmd[2]))
	if err != nil {
		return resp.MakeErrorData("index must be an integer")
	}

	key := string(cmd[1])

	if !m.CheckTTL(key) {
		return resp.MakeErrorData("key not exist")
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeErrorData("key not exist")
	}

	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	success := list.Set(index, cmd[3])
	if !success {
		return resp.MakeErrorData("index out of range")
	}
	return resp.MakeStringData("OK")
}

func lRemList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lrem" {
		logger.Error("lRemList Function : cmdName is not lrem")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) != 4 {
		return resp.MakeErrorData("wrong number of arguments for 'lrem' command")
	}

	count, err := strconv.Atoi(string(cmd[2]))
	if err != nil {
		return resp.MakeErrorData("count must be an integer")
	}

	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeIntData(0)
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeIntData(0)
	}

	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	defer func() {
		if list.Len == 0 {
			m.db.Delete(key)
			m.DelTTL(key)
		}
	}()

	res := list.RemoveElement(cmd[3], count)

	return resp.MakeIntData(int64(res))
}

func lTrimList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "ltrim" {
		logger.Error("lTrimList Function : cmdName is not ltrim")
		return resp.MakeErrorData("server error")
	}
	if len(cmd) != 4 {
		return resp.MakeErrorData("wrong number of arguments for 'ltrim' command")
	}
	start, err1 := strconv.Atoi(string(cmd[2]))
	end, err2 := strconv.Atoi(string(cmd[3]))
	if err1 != nil || err2 != nil {
		return resp.MakeErrorData("start and end must be an integer")
	}

	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeStringData("OK")
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)

	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeStringData("OK")
	}
	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	defer func() {
		if list.Len == 0 {
			m.db.Delete(key)
			m.DelTTL(key)
		}
	}()

	list.Trim(start, end)
	return resp.MakeStringData("OK")
}

func lRangeList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lrange" {
		logger.Error("lRangeList Function : cmdName is not lrange")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) != 4 {
		return resp.MakeErrorData("wrong number of arguments for 'lrange' command")
	}

	start, err1 := strconv.Atoi(string(cmd[2]))
	end, err2 := strconv.Atoi(string(cmd[3]))
	if err1 != nil || err2 != nil {
		return resp.MakeErrorData("index must be an integer")
	}

	key := string(cmd[1])
	if !m.CheckTTL(key) {
		return resp.MakeEmptyArrayData()
	}

	m.locks.RLock(key)
	defer m.locks.RUnLock(key)

	tem, ok := m.db.Get(key)
	if !ok {
		return resp.MakeEmptyArrayData()
	}
	list, ok := tem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	temRes := list.Range(start, end)
	if temRes == nil {
		return resp.MakeEmptyArrayData()
	}
	res := make([]resp.RedisData, len(temRes))
	for i := 0; i < len(temRes); i++ {
		res[i] = resp.MakeBulkData(temRes[i])
	}

	return resp.MakeArrayData(res)
}

func lMoveList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[0])) != "lmove" {
		logger.Error("lMoveList Function : cmdName is not lmove")
		return resp.MakeErrorData("server error")
	}

	if len(cmd) != 5 {
		return resp.MakeErrorData("wrong number of arguments for 'lmove' command")
	}

	src := string(cmd[1])
	des := string(cmd[2])
	srcDrc := strings.ToLower(string(cmd[3]))
	desDrc := strings.ToLower(string(cmd[4]))
	if (srcDrc != "left" && srcDrc != "right") || (desDrc != "left" && desDrc != "right") {
		return resp.MakeErrorData("options must be left or right")
	}

	if !m.CheckTTL(src) {
		return resp.MakeBulkData(nil)
	}

	m.CheckTTL(des)

	keys := []string{src, des}

	m.locks.LockMulti(keys)
	defer m.locks.UnLockMulti(keys)

	srcTem, ok := m.db.Get(src)
	if !ok {
		return resp.MakeBulkData(nil)
	}

	srcList, ok := srcTem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	defer func() {
		if srcList.Len == 0 {
			m.db.Delete(src)
			m.DelTTL(src)
		}
	}()

	if srcList.Len == 0 {
		return resp.MakeBulkData(nil)
	}

	desTem, ok := m.db.Get(des)
	if !ok {
		desTem = NewList()
		m.db.Set(des, desTem)
	}

	desList, ok := desTem.(*List)
	if !ok {
		return resp.MakeErrorData("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// pop from src
	var popElem *ListNode
	if srcDrc == "left" {
		popElem = srcList.LPop()
	} else {
		popElem = srcList.RPop()
	}

	//    insert to des
	if desDrc == "left" {
		desList.LPush(popElem.Val)
	} else {
		desList.RPush(popElem.Val)
	}
	return resp.MakeBulkData(popElem.Val)
}

func blPopList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	// at least 3 args like "BLPOP key timeout"
	if len(cmd) < 3 {
		return resp.MakeErrorData("ERR wrong number of arguments for 'blpop' command")
	}
	return bXPopList(m, cmd, "left")
}

func brPopList(m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	// at least 3 args like "BLPOP key timeout"
	if len(cmd) < 3 {
		return resp.MakeErrorData("ERR wrong number of arguments for 'blpop' command")
	}
	return bXPopList(m, cmd, "right")
}

func bXPopList(m *MemDb, cmd [][]byte, direction string) resp.RedisData {

	// last arg is block timeout
	timeout, err := strconv.Atoi(string(cmd[len(cmd)-1]))
	if err != nil {
		return resp.MakeErrorData("ERR timeout is not a float or out of range")
	}
	var timer *time.Timer
	if timeout == 0 {
		timer = time.NewTimer(math.MaxInt)
	} else {
		timer = time.NewTimer(time.Duration(timeout) * time.Second)
	}
	// query interval (this could be narrowed down to query more frequently but will use more CPU resource)
	ticker := time.NewTicker(100 * time.Millisecond) // 100ms per query = 10 times per second

	// retrieve all list names
	keyBytes := cmd[1 : len(cmd)-1]
	keyStrings := make([]string, 0, len(keyBytes))
	for _, bStr := range keyBytes {
		keyStrings = append(keyStrings, string(bStr))
	}
	left := "left"
	right := "right"

	// block (will return inside the infinite loop)
	for {
		select {
		// time to query keys
		case <-ticker.C:
			for _, key := range keyStrings {
				// lock db actions
				m.locks.Lock(key)
				// get key
				tmp, ok := m.db.Get(key)
				// key exist
				if ok {
					// assert is List
					list, isList := tmp.(*List)
					if isList {
						// Pop from list
						var node *ListNode
						if direction == left {
							node = list.LPop()
						} else if direction == right {
							node = list.RPop()
						}
						if node != nil {
							// find a value. need to manually release the lock since we are leaving this scope
							m.locks.UnLock(key)
							return resp.MakeArrayData([]resp.RedisData{resp.MakeStringData(key), resp.MakeBulkData(node.Val)})
						}
					}
				}
				// will finally release the lock here if nothing available
				m.locks.UnLock(key)
			}
		// timeout
		case <-timer.C:
			return resp.MakeBulkData(nil)
		}
	}
}

func RegisterListCommands() {
	RegisterCommand("llen", lLenList)
	RegisterCommand("lindex", lIndexList)
	RegisterCommand("lpos", lPosList)
	RegisterCommand("lpop", lPopList)
	RegisterCommand("rpop", rPopList)
	RegisterCommand("lpush", lPushList)
	RegisterCommand("lpushx", lPushXList)
	RegisterCommand("rpush", rPushList)
	RegisterCommand("rpushx", rPushXList)
	RegisterCommand("lset", lSetList)
	RegisterCommand("lrem", lRemList)
	RegisterCommand("ltrim", lTrimList)
	RegisterCommand("lrange", lRangeList)
	RegisterCommand("lmove", lMoveList)
	RegisterCommand("blpop", blPopList)
	RegisterCommand("brpop", brPopList)
}
