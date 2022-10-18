package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"log"
	"net"
	"strconv"
	"strings"
)

func zadd(m *MemDb, cmd cmdBytes, _ net.Conn) resp.RedisData {
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
	for k := range optionDict {
		if strings.Contains(optionsStr, k) {
			optionDict[k] = true
			idx++
		}
	}
	// check mutual excluded options
	if (optionDict["gt"] && optionDict["lt"]) || (optionDict["nx"] && optionDict["gt"]) || (optionDict["nx"] && optionDict["lt"]) {
		return resp.MakeErrorData("ERR GT, LT, and/or NX options at the same time are not compatible")
	}
	// lock the key
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	var sortedSet *SortedSet[*SortedSetNode]
	// get key, create a sorted list if key does not exist
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
	// todo: optimize performance here. dont create a single virtual node in each loop
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
		// insert value (temporal structure here)
		r := &SortedSetNode{
			Names: map[string]struct{}{member: {}},
			Score: score,
		}
		// decide member count
		if _, ok := sortedSet.dict[member]; !ok {
			addedCount++
		}
		// if the member exists, delete it first
		node := sortedSet.GetByName(member)
		if node != nil {
			sortedSet.Delete(member)
		}
		// insert member
		sortedSet.Insert(r)
	}
	log.Println(sortedSet.Values())
	return resp.MakeIntData(addedCount)
}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
}
