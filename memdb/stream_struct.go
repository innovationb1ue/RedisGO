package memdb

import (
	"errors"
	"fmt"
	"sync"
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
func (s *Stream) AddEntry(timeStamp int64, seqNum int64, val []string) error {
	// first entry
	if len(s.timeStamps) == 0 {
		idStr := fmt.Sprintf("%d-%d", timeStamp, seqNum)
		s.timeStamps = append(s.timeStamps, &StreamID{
			time:   timeStamp,
			seqNum: seqNum,
		})
		s.entry[idStr] = val
		return nil
	}
	// check top
	top := s.timeStamps[len(s.timeStamps)-1]
	if timeStamp < top.time || (timeStamp == top.time && top.seqNum >= seqNum) {
		return errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}
	// same time stamp but larger sequence number or larger time stamp
	idStr := fmt.Sprintf("%d-%d", timeStamp, seqNum)
	s.timeStamps = append(s.timeStamps, &StreamID{
		time:   timeStamp,
		seqNum: seqNum,
	})
	s.entry[idStr] = val
	return nil
}

func (s *Stream) Range(start *StreamID, end *StreamID) ([]*StreamID, [][]string) {
	if len(s.timeStamps) == 0 {
		return nil, nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	var msgs = make([][]string, 0, len(s.entry))
	if start.time == -1 && start.seqNum == -1 && end.seqNum == -1 && end.time == -1 {
		for _, k := range s.timeStamps {
			msgs = append(msgs, s.entry[fmt.Sprintf("%d-%d", k.time, k.seqNum)])
		}
		return s.timeStamps, msgs
	}
	return nil, nil
}

func (i *StreamID) Format() string {
	return fmt.Sprintf("%d-%d", i.time, i.seqNum)
}
