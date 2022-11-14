package memdb

import (
	"context"
	"fmt"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"strconv"
	"strings"
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
	var withscore, rev, byscore, bylex, limit bool
	var offset, count int
	var err error

	// parse options
	for i, optionStr := range cmd[4:] {
		switch strings.ToLower(string(optionStr)) {
		case "withscores":
			withscore = true
		case "rev":
			rev = true
		case "limit":
			limit = true
			if i+2 >= len(cmd) {
				return resp.MakeErrorData("ERR syntax error")
			}
			// when met limit keyword, we need to process the following offset and count
			offset, err = strconv.Atoi(string(cmd[i+4+1]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
			count, err = strconv.Atoi(string(cmd[i+4+2]))
			if err != nil {
				return resp.MakeErrorData("ERR value is not an integer or out of range")
			}
		// never use this option. this is an awful option provided by Redis. we return empty here.
		case "byscore":
			byscore = true
			return resp.MakeArrayData([]resp.RedisData{})
		case "bylex":
			bylex = true
		}
	}
	// check invalid combination of options

	// limit is always used with byscore or bylex
	if limit && !(byscore || bylex) {
		return resp.MakeErrorData("ERR syntax error, LIMIT is only supported in combination with either BYSCORE or BYLEX")
	}
	if bylex && withscore {
		return resp.MakeErrorData("ERR syntax error, WITHSCORES not supported in combination with BYLEX")
	}
	if bylex && rev {
		return resp.MakeArrayData([]resp.RedisData{})
	}
	var start, end int
	if !byscore && !bylex {
		start, err = strconv.Atoi(string(cmd[2]))
		if err != nil {
			return resp.MakeErrorData("ERR value is not an integer or out of range")
		}
		end, err = strconv.Atoi(string(cmd[3]))
		if err != nil {
			return resp.MakeErrorData("ERR value is not an integer or out of range")
		}
	}

	// retrive the key
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
	// get set length
	NumOfMembers := sortedSet.Len()
	if end > NumOfMembers {
		end = NumOfMembers
	}
	// traverse btree for all elements , members are in ascend order
	members := make([]*SortedSetMember, 0, NumOfMembers)
	memberCount := -1 // current member index
	sortedSet.Ascend(func(node *Node[*SortedSetNode], int2 int) bool {
		memberCount++
		if memberCount < start {
			return true
		}
		if memberCount > end {
			return false
		}
		names := node.Value.GetNames()
		score := node.Value.GetScore()
		for n := range names {
			members = append(members, &SortedSetMember{name: n, score: score})
		}
		return true
	})
	// special handling for byscore. entering this condition will return the whole function inside
	if byscore {
		var leftopen, rightopen bool
		var min, max float64
		var err error
		// get range interval (default is closed interval)
		if cmd[2][0] == '(' {
			min, err = strconv.ParseFloat(string(cmd[2][1:]), 64)
			leftopen = true
		} else {
			min, err = strconv.ParseFloat(string(cmd[2]), 64)
		}
		if cmd[3][0] == '(' {
			max, err = strconv.ParseFloat(string(cmd[3][1:]), 64)
			rightopen = true
		} else {
			max, err = strconv.ParseFloat(string(cmd[3]), 64)
		}
		if err != nil {
			return resp.MakeErrorData("ERR min or max is not a float")
		}
		scoreIdxStart := -1
		scoreIdxEnd := -1
		// retrieve members based on the interval
		if !leftopen && !rightopen {
			for idx, member := range members {
				if member.score >= min && member.score <= max && scoreIdxStart == -1 {
					scoreIdxStart = idx
				} else if member.score > max && scoreIdxEnd == -1 {
					scoreIdxEnd = idx - 1 // by all means do not include this one
					break
				}
			}
		} else if leftopen && !rightopen {
			for idx, member := range members {
				if member.score > min && member.score <= max && scoreIdxStart == -1 {
					scoreIdxStart = idx
				} else if member.score > max && scoreIdxEnd == -1 {
					scoreIdxEnd = idx - 1 // by all means do not include this one
					break
				}
			}
		} else if !leftopen && rightopen {
			for idx, member := range members {
				if member.score >= min && member.score < max && scoreIdxStart == -1 {
					scoreIdxStart = idx
				} else if member.score >= max && scoreIdxEnd == -1 {
					scoreIdxEnd = idx - 1 // by all means do not include this one
					break
				}
			}
		}
		// middle of members till the end. (idxStart got but idxEnd is missing, we assign End to max length)
		if scoreIdxStart >= 0 && scoreIdxEnd == -1 {
			scoreIdxEnd = NumOfMembers
		}
		members = members[scoreIdxStart : scoreIdxEnd+1]
		// build response if there is any member fall into the interval
		if scoreIdxStart >= 0 {
			res := make([]resp.RedisData, 0)
			// handle offset and limit if specified
			if limit {
				if offset >= len(members) || offset < 0 {
					return resp.MakeArrayData([]resp.RedisData{})
				}
				// shift members rightward with offset
				members = members[offset:]
				// save only {count} members
				if count > 0 && count <= len(members) {
					members = members[:count]
				}
			}

			// make response
			for _, member := range members {
				name := member.name
				score := member.score
				res = append(res, resp.MakeBulkData([]byte(name)))
				// if specified withscores option, also add member scores in the response
				if withscore {
					res = append(res, resp.MakeBulkData([]byte(fmt.Sprintf("%f", score))))
				}
			}

			return resp.MakeArrayData(res)
		} else {
			// no member in the interval, just return empty array
			return resp.MakeArrayData([]resp.RedisData{})
		}
	}
	// normal ZRANGE without byscore|bylex|
	res := make([]resp.RedisData, 0)
	for _, member := range members {
		name := member.name
		score := member.score
		res = append(res, resp.MakeBulkData([]byte(name)))
		if withscore {
			res = append(res, resp.MakeBulkData([]byte(fmt.Sprintf("%f", score))))
		}
	}
	//res = res[start:NumOfMembers]
	if rev {
		res = reverse[[]resp.RedisData](res)
	}
	return resp.MakeArrayData(res)
}

func zrem(ctx context.Context, m *MemDb, cmd cmdBytes, _ net.Conn) resp.RedisData {
	if len(cmd) < 3 {
		return resp.MakeWrongNumberArgs("zrem")
	}
	key := strings.ToLower(string(cmd[1]))
	// retrieve the key
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
	keys := make([]string, 0, len(cmd)-2)
	for _, s := range cmd[2:] {
		keys = append(keys, string(s))
	}
	affectedCount := int64(0)
	for _, k := range keys {
		res := sortedSet.Delete(k)
		if res != nil {
			affectedCount++
		}
	}
	return resp.MakeIntData(affectedCount)

}

func RegisterSortedSetCommands() {
	RegisterCommand("zadd", zadd)
	RegisterCommand("zrange", zrange)
	RegisterCommand("zrem", zrem)
}
