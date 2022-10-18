package memdb

import "github.com/innovationb1ue/RedisGO/logger"

type SortedSet[T Val] struct {
	*Btree[T]
}

func NewSortedSet() *SortedSet[*SortedSetNode] {
	return &SortedSet[*SortedSetNode]{NewBtree[*SortedSetNode]()}
}

type SortedSetNode struct {
	Names map[string]struct{} // allow multiple member with the same value
	Score float64
}

func (n *SortedSetNode) Empty() {
	n.Names = nil
}

func (n *SortedSetNode) SetScore(score float64) {
	n.Score = score
}

func (n *SortedSetNode) GetScore() float64 {
	return n.Score
}

func (n *SortedSetNode) IsNameExist(name string) bool {
	_, ok := n.Names[name]
	return ok
}

func (n *SortedSetNode) AddName(name string) {
	n.Names[name] = struct{}{}
}

func (n *SortedSetNode) DeleteName(name string) {
	delete(n.Names, name)
}

func (n *SortedSetNode) Comp(val Val) int64 {
	n2 := val.(*SortedSetNode)
	if n.Score > n2.Score {
		return 1
	} else if n.Score < n2.Score {
		return -1
	} else if n.Score == n2.Score {
		return 0
	} else {
		logger.Error("cant compare values in sorted set Comp")
		return 0
	}
}

func (n *SortedSetNode) GetNames() map[string]struct{} {
	return n.Names
}
