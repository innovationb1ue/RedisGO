package memdb

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type StreamID struct {
	time   int64
	seqNum int64
}

type Stream struct {
	entry      map[string][]string // message is bulk of strings
	timeStamps []*StreamID
	lock       sync.RWMutex
}

// NewStream create a new stream data structure
func NewStream() *Stream {
	return &Stream{
		entry:      make(map[string][]string),
		timeStamps: make([]*StreamID, 0),
		lock:       sync.RWMutex{},
	}
}

// AddEntry perform O(1) inserting
func (s *Stream) AddEntry(ID *StreamID, val []string) error {
	// auto determine timestamp if necessary
	if ID.time == -1 {
		ID.time = time.Now().UnixMilli()
	}
	// lock map
	s.lock.Lock()
	defer s.lock.Unlock()
	// first entry
	if len(s.timeStamps) == 0 {
		ID.seqNum = 0
		s.timeStamps = append(s.timeStamps, ID)
		s.entry[ID.Format()] = val
		return nil
	}
	// check top
	top := s.timeStamps[len(s.timeStamps)-1]
	if ID.time < top.time || (ID.time == top.time && top.seqNum >= ID.seqNum) {
		return errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}
	// larger ID
	if ID.time == top.time {
		ID.seqNum = top.seqNum + 1
	} else {
		ID.seqNum = 0
	}
	s.timeStamps = append(s.timeStamps, ID)
	s.entry[ID.Format()] = val
	return nil
}

// Range go over a specific interval of stream IDs and return all the entries with that range.
// use -1 for range to infinity
// Return: ID structures, slice of key-value pairs of each entry(2d slice)
func (s *Stream) Range(start *StreamID, end *StreamID) ([]*StreamID, [][]string) {
	if len(s.timeStamps) == 0 {
		return nil, nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	var msgs = make([][]string, 0)
	var IDs = make([]*StreamID, 0)
	// infinity case
	if start.time == -1 && end.time == -1 {
		for _, k := range s.timeStamps {
			msgs = append(msgs, s.entry[fmt.Sprintf("%d-%d", k.time, k.seqNum)])
			IDs = append(IDs, k)
		}
		return s.timeStamps, msgs
	}
	return nil, nil
}

func (s *Stream) DropFirst() int {
	if len(s.timeStamps) == 0 {
		return 0
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	stamp := s.timeStamps[0]
	s.timeStamps = s.timeStamps[1:]
	delete(s.entry, stamp.Format())
	return len(s.timeStamps)
}

// Format return the string representation of a standard stream entry ID like "1667271690022-1"
func (i *StreamID) Format() string {
	return fmt.Sprintf("%d-%d", i.time, i.seqNum)
}
