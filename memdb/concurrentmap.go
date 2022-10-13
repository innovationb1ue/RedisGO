package memdb

import (
	"sync"

	"github.com/innovationb1ue/RedisGO/util"
)

const MaxConSize = int(1<<31 - 1)

// ConcurrentMap manage a table slice with multiple hashmap shards to avoid lock bottleneck.
// It is threads safe by using rwLock.
// it supports maximum table size = MaxConSize
type ConcurrentMap struct {
	table []*shard
	size  int // table size
	count int // total number of keys
}

// shard is the minimal object that represents a k:v pair in redis
type shard struct {
	item map[string]any
	rwMu *sync.RWMutex
}

// NewConcurrentMap create a new ConcurrentMap with given size. If size <=0, it will be set to MaxConSize.
func NewConcurrentMap(size int) *ConcurrentMap {
	if size <= 0 || size > MaxConSize {
		size = MaxConSize
	}
	m := &ConcurrentMap{
		table: make([]*shard, size),
		size:  size,
		count: 0,
	}
	// fill shards
	for i := 0; i < size; i++ {
		m.table[i] = &shard{item: make(map[string]any), rwMu: &sync.RWMutex{}}
	}
	return m
}

func (m *ConcurrentMap) getKeyPos(key string) int {
	return util.HashKey(key) % m.size
}

func (m *ConcurrentMap) getShard(key string) *shard {
	return m.table[m.getKeyPos(key)]
}

func (m *ConcurrentMap) Set(key string, value any) int {
	added := 0
	shard := m.getShard(key)
	shard.rwMu.Lock()
	defer shard.rwMu.Unlock()

	if _, ok := shard.item[key]; !ok {
		m.count++
		added = 1
	}
	shard.item[key] = value
	return added
}

func (m *ConcurrentMap) SetIfExist(key string, value any) int {
	pos := m.getKeyPos(key)
	shard := m.table[pos]
	shard.rwMu.Lock()
	defer shard.rwMu.Unlock()

	if _, ok := shard.item[key]; ok {
		shard.item[key] = value
		return 1
	}
	return 0
}

func (m *ConcurrentMap) SetIfNotExist(key string, value any) int {
	pos := m.getKeyPos(key)
	shard := m.table[pos]
	shard.rwMu.Lock()
	defer shard.rwMu.Unlock()

	if _, ok := shard.item[key]; !ok {
		m.count++
		shard.item[key] = value
		return 1
	}
	return 0
}

func (m *ConcurrentMap) Get(key string) (any, bool) {
	pos := m.getKeyPos(key)
	// lock shard for reading.
	// this ensures no write will be synchronized before this read finished.
	shard := m.table[pos]
	shard.rwMu.RLock()
	defer shard.rwMu.RUnlock()

	value, ok := shard.item[key]
	return value, ok
}

func (m *ConcurrentMap) Delete(key string) int {
	pos := m.getKeyPos(key)
	shard := m.table[pos]
	shard.rwMu.Lock()
	defer shard.rwMu.Unlock()

	if _, ok := shard.item[key]; ok {
		delete(shard.item, key)
		m.count--
		return 1
	}
	return 0
}

func (m *ConcurrentMap) Len() int {
	return m.count
}

func (m *ConcurrentMap) Clear() {
	*m = *NewConcurrentMap(m.size)
}

func (m *ConcurrentMap) Keys() []string {
	keys := make([]string, m.count)
	i := 0
	for _, shard := range m.table {
		shard.rwMu.RLock()
		for key := range shard.item {
			keys[i] = key
			i++
		}
		shard.rwMu.RUnlock()
	}
	return keys
}
