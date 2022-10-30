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
