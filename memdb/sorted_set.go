package memdb

import (
	"fmt"
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
	// skip "ZADD" and "{KEY}"
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
	if optionDict["incr"] && len(cmd) != 5 {
		return resp.MakeErrorData("ERR INCR option supports a single increment-element pair")
	}

	// lock the key
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	// declare sortedSet data structure
	var sortedSet *SortedSet[*SortedSetNode]
	// get key, create a new sorted list if the key does not exist
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
	var retInt int64
	var targetScore float64 // used when incr option
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
		// try to get the old member if exist
		node := sortedSet.GetByName(member)
		var r *SortedSetNode
		// check incr option or normally set the score
		if optionDict["incr"] && node != nil {
			targetScore = node.Value.GetScore() + score
			// create a virtual node to hold member names
			r = &SortedSetNode{
				Names: map[string]struct{}{member: {}},
				Score: targetScore,
			}
		} else {
			r = &SortedSetNode{
				Names: map[string]struct{}{member: {}},
				Score: score,
			}
		}
		// decide the return int64 value
		// ch means return the number of value changed	(added + changed)
		// normally we only count the member added 		(added)
		if optionDict["ch"] && node.Value.GetScore() != score {
			retInt++
		} else
		// member non-exist
		if _, ok := sortedSet.dict[member]; !ok {
			retInt++
		}
		// if the member exists, delete it first
		if node != nil {
			sortedSet.Delete(member)
		}
		// insert new member
		sortedSet.Insert(r)
	}
	log.Println(sortedSet.Values())
	if optionDict["incr"] {
		return resp.MakeBulkData(resp.MakePlainData(fmt.Sprintf("%f", targetScore)).ByteData())
	}
	return resp.MakeIntData(retInt)
}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
}
