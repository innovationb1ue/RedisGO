package memdb

import "github.com/innovationb1ue/RedisGO/logger"

type SortedSet[T Val] struct {
	*Btree[T]
}

func NewSortedSet() *SortedSet[SortedSetNode] {
	return &SortedSet[SortedSetNode]{NewBtree[SortedSetNode]()}
}

type SortedSetNode struct {
	name  string
	score float64
}

func (n SortedSetNode) Comp(val Val) int64 {
	// only compare the same type of values
	n2 := val.(SortedSetNode)
	if n.score > n2.score {
		return 1
	} else if n.score < n2.score {
		return -1
	} else if n.score == n2.score {
		return 0
	} else {
		logger.Error("cant compare values in sorted set Comp")
		return 0
	}
}

func (n SortedSetNode) GetName() string {
	return n.name
}
