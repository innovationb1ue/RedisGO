package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"strconv"
	"strings"
)

func zadd(m *MemDb, cmd cmdBytes, conn net.Conn) resp.RedisData {
	if len(cmd) < 4 {
		return resp.MakeWrongNumberArgs("zadd")
	}
	key := string(cmd[1])
	elems := make([]string, 0, len(cmd))
	for _, b := range cmd {
		elems = append(elems, string(b))
	}
	// phrase options
	optionsStr := ""
	// skip "ZADD" & "{KEY}"
	for _, s := range elems[2:] {
		optionsStr += s + "/"
	}
	// idx points to the start index of score:member pairs
	idx := 2
	// get options
	optionDict := map[string]bool{"nx": false, "xx": false, "gt": false, "lt": false, "ch": false, "incr": false}
	for k, _ := range optionDict {
		if strings.Contains(optionsStr, k) {
			optionDict[k] = true
			idx++
		}
	}
	// check mutual excluded options
	if (optionDict["gt"] && optionDict["lt"]) || (optionDict["nx"] && optionDict["gt"]) || (optionDict["nx"] && optionDict["lt"]) {
		return resp.MakeErrorData("ERR GT, LT, and/or NX options at the same time are not compatible")
	}
	// get key, create a sorted list if key does not exist
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	var sortedSet *SortedSet[SortedSetNode]
	SortedsetTmp, ok := m.db.Get(key)
	if !ok {
		sortedSet = NewSortedSet()
		m.db.Set(key, sortedSet)
	} else {
		sortedSet, ok = SortedsetTmp.(*SortedSet[SortedSetNode])
		if !ok {
			return resp.MakeWrongType()
		}
	}
	var addedCount int64
	// set value:member pairs
	for i := idx; i < len(cmd); i += 2 {
		// check overflow
		if i+1 >= len(cmd) {
			return resp.MakeErrorData("ERR syntax error")
		}
		// phrase score
		score, err := strconv.ParseFloat(string(cmd[i]), 64)
		if err != nil {
			return resp.MakeErrorData("ERR value is not a valid float")
		}
		// get member name
		member := string(cmd[i+1])
		// insert value (copy here. ) todo: change this to pass by reference to avoid coping everywhere.
		item := SortedSetNode{
			name:  member,
			score: score,
		}
		isExist := sortedSet.GetByName(member)
		// if the node exists, delete it first. O(logn)
		if isExist != nil {
			sortedSet.DeleteByName(member)
			sortedSet.Insert(item)
		} else {
			sortedSet.Insert(item)
			addedCount++
		}
	}

	return resp.MakeIntData(addedCount)
}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
}
