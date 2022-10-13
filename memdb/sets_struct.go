package memdb

import "math"

type void struct{}

type Set struct {
	table map[string]void
}

func NewSet() *Set {
	return &Set{make(map[string]void)}
}

func (s *Set) Add(key string) int {
	if s.Has(key) {
		return 0
	}
	s.table[key] = void{}
	return 1
}

func (s *Set) Remove(key string) int {
	if s.Has(key) {
		delete(s.table, key)
		return 1
	}
	return 0
}

func (s *Set) Len() int {
	return len(s.table)
}

func (s *Set) Has(key string) bool {
	_, ok := s.table[key]
	return ok
}

func (s *Set) Pop() string {
	for key := range s.table {
		s.Remove(key)
		return key
	}
	return ""
}

func (s *Set) Clear() {
	s.table = make(map[string]void)
}

func (s *Set) Members() []string {
	res := make([]string, 0, len(s.table))
	for key := range s.table {
		res = append(res, key)
	}
	return res
}

func (s *Set) Union(sets ...*Set) *Set {
	res := NewSet()
	for key := range s.table {
		res.Add(key)
	}
	for _, set := range sets {
		for key := range set.table {
			res.Add(key)
		}
	}
	return res
}

func (s *Set) Intersect(sets ...*Set) *Set {
	res := NewSet()
	for key := range s.table {
		res.Add(key)
	}
	for _, set := range sets {
		for key := range res.table {
			if !set.Has(key) {
				res.Remove(key)
			}
		}
	}
	return res
}

func (s *Set) Difference(sets ...*Set) *Set {
	res := NewSet()
	for key := range s.table {
		res.Add(key)
	}
	for _, set := range sets {
		for key := range set.table {
			res.Remove(key)
		}
	}
	return res
}

func (s *Set) IsSubset(set *Set) bool {
	for key := range s.table {
		if !set.Has(key) {
			return false
		}
	}
	return true
}

// Random returns a random member of the set.
// if count > 0, return max(len(set), count) number random members
// if count < 0, return exactly count number random members
func (s *Set) Random(count int) []string {
	absCount := int(math.Abs(float64(count)))
	res := make([]string, 0, absCount)
	// if empty
	if count == 0 || s.Len() == 0 {
		return res
	} else if count > 0 {
		// return exactly {count} elements
		if count > s.Len() {
			count = s.Len()
		}
		for key := range s.table {
			res = append(res, key)
			if len(res) == count {
				return res
			}
		}
	} else {
		// count < 0, return repeated elems until the array is filled
		for len(res) < -count {
			for key := range s.table {
				res = append(res, key)
				if len(res) == -count {
					break
				}
			}
		}
		return res
	}
	return res
}
