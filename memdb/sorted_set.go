package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"log"
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
	// idx points to the start index of Score:member pairs
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
	// lock the key
	defer m.locks.UnLock(key)
	var sortedSet *SortedSet[*SortedSetNode]
	SortedsetTmp, ok := m.db.Get(key)
	if !ok {
		sortedSet = NewSortedSet()
		m.db.Set(key, sortedSet)
	} else {
		sortedSet, ok = SortedsetTmp.(*SortedSet[*SortedSetNode])
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
		// phrase Score
		score, err := strconv.ParseFloat(string(cmd[i]), 64)
		if err != nil {
			return resp.MakeErrorData("ERR value is not a valid float")
		}
		// get member Names
		member := string(cmd[i+1])
		// insert value (copied here. )
		r := &record{
			name:  member,
			score: score,
		}
		// if the node exists, append the member Names to the node.Names list
		added := sortedSet.Insert(r)
		if added {
			addedCount++
		}

	}
	log.Println(sortedSet.Values(), sortedSet.len)
	return resp.MakeIntData(addedCount)
}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
}
