package memdb

import (
	"context"
	"fmt"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"strconv"
)

type SortedSetMember struct {
	name  string
	score float64
}

func reverse[S ~[]E, E any](s S) S {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func zadd(ctx context.Context, m *MemDb, cmd cmdBytes, _ net.Conn) resp.RedisData {
	if len(cmd) < 4 {
		return resp.MakeWrongNumberArgs("zadd")
	}
	// convert bytes to strings
	elems := make([]string, 0, len(cmd))
	for _, b := range cmd {
		elems = append(elems, string(b))
	}
	key := elems[1]
	// idx points to the start index of Score:member pairs
	idx := 2
	// get options
	var nx, xx, gt, lt, ch, incr bool
	var endArgsFlag bool
	for i := idx; i < len(cmd)-2; i++ {
		switch elems[i] {
		case "nx":
			nx = true
		case "xx":
			xx = true
		case "gt":
			gt = true
		case "lt":
			lt = true
		case "ch":
			ch = true
		case "incr":
			incr = true
		default:
			endArgsFlag = true
			break
		}
		if endArgsFlag {
			break
		} else {
			idx++
		}
	}
	// check mutual excluded options
	if (gt && lt) || (nx && gt) || (nx && lt) {
		return resp.MakeErrorData("ERR GT, LT, and/or NX options at the same time are not compatible")
	}
	if incr && len(cmd) != 5 {
		return resp.MakeErrorData("ERR INCR option supports a single increment-element pair")
	}
	// declare sortedSet data structure
	var sortedSet *SortedSet[*SortedSetNode]
	// lock the key
	m.locks.Lock(key)
	defer m.locks.UnLock(key)
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
	retInt := int64(0)
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
		// try to get the old member if exist (node can be nil)
		node := sortedSet.GetByName(member)
		// check update-only option
		if node == nil && xx {
			continue
		}
		// check add-only option
		if node != nil && nx {
			continue
		}
		// check less-than option
		if node != nil && lt && score >= node.Value.GetScore() {
			continue
		}
		// check greater-than option
		if node != nil && gt && score <= node.Value.GetScore() {
			continue
		}
		// check for all same conditions. => do nothing
		if node != nil && !incr && score == node.Value.GetScore() {
			continue
		}

		// check incr option
		var r *SortedSetNode
		if incr && node != nil {
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
		if ch && node.Value.GetScore() != score {
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
	//log.Println(sortedSet.Values())
	if incr {
		return resp.MakeBulkData(resp.MakePlainData(fmt.Sprintf("%f", targetScore)).ByteData())
	}
	return resp.MakeIntData(retInt)
}

func zrange(ctx context.Context, m *MemDb, cmd cmdBytes, _ net.Conn) resp.RedisData {
	if len(cmd) < 4 {
		return resp.MakeWrongNumberArgs("zrange")
	}
	key := string(cmd[1])
	start, err := strconv.Atoi(string(cmd[2]))
	if err != nil {
		return resp.MakeErrorData("ERR value is not an integer or out of range")
	}
	end, err := strconv.Atoi(string(cmd[3]))
	if err != nil {
		return resp.MakeErrorData("ERR value is not an integer or out of range")
	}
	var withscore, rev bool

	// parse options
	for _, optionStr := range cmd[4:] {
		switch string(optionStr) {
		case "withscores":
			withscore = true
		case "rev":
			rev = true
		}
	}

	m.locks.Lock(key)
	defer m.locks.UnLock(key)
	var sortedSet *SortedSet[*SortedSetNode]
	sortedSetTmp, ok := m.db.Get(key)
	if !ok {
		return resp.MakeArrayData([]resp.RedisData{})
	} else {
		sortedSet, ok = sortedSetTmp.(*SortedSet[*SortedSetNode])
		if !ok {
			return resp.MakeWrongType()
		}
	}
	res := make([]resp.RedisData, 0)
	sortedSet.Ascend(func(node *Node[*SortedSetNode], int2 int) bool {
		names := node.Value.GetNames()
		score := node.Value.GetScore()
		for n := range names {
			res = append(res, resp.MakeBulkData([]byte(n)))
			if withscore {
				res = append(res, resp.MakeBulkData([]byte(fmt.Sprintf("%f", score))))
			}
		}
		return true
	})
	var maxLen int
	if end >= len(res) {
		maxLen = len(res)
	} else {
		maxLen = end + 1
	}

	res = res[start:maxLen]
	if rev {
		res = reverse[[]resp.RedisData](res)
	}
	return resp.MakeArrayData(res)
}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
	RegisterCommand("zrange", zrange)
}
