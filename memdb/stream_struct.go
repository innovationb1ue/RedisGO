package memdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StreamID struct {
	time   int64
	seqNum int64
}

type Stream struct {
	entry      map[string][]string // message is bulk of strings
	timeStamps []*StreamID         // ordered IDs
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
	// lock map
	s.lock.Lock()
	defer s.lock.Unlock()
	// auto determine timestamp if necessary
	if ID.time == -1 {
		ID.time = time.Now().UnixMilli()
	}
	// first entry
	if len(s.timeStamps) == 0 {
		ID.seqNum = 0
		s.timeStamps = append(s.timeStamps, ID)
		s.entry[ID.Format()] = val
		return nil
	}
	// check top first part. lower timestamp => return error
	top := s.timeStamps[len(s.timeStamps)-1]
	if ID.time < top.time {
		return errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}
	// auto determine sequence number with same timestamp
	if ID.time == top.time && ID.seqNum == -1 {
		ID.seqNum = top.seqNum + 1
	} else if ID.time == top.time && ID.seqNum < top.seqNum {
		return errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
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
	// normal case (linear search for the start and the end)
	// complexity: O(n)
	// can be optimized to O(nlogn) but further imprvement require data structure refactor.
	flag := false
	for _, k := range s.timeStamps {
		// abort when exceeding maximum timestamp and not inf range
		if k.time > end.time && end.time != -1 || (k.time == end.time && k.seqNum >= end.seqNum) {
			break
		}
		// find the start
		if k.time >= start.time && (k.seqNum >= start.seqNum || start.seqNum == -1) {
			flag = true
		}
		if flag {
			msgs = append(msgs, s.entry[fmt.Sprintf("%d-%d", k.time, k.seqNum)])
			IDs = append(IDs, k)
		}
	}
	return IDs, msgs
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

func (s *Stream) DropFirstN(n int) int {
	s.lock.Lock()
	defer s.lock.Unlock()
	length := len(s.timeStamps)
	if n == 0 {
		return len(s.timeStamps)
	}
	// drop all
	if n >= length {
		s.entry = make(map[string][]string)
		s.timeStamps = make([]*StreamID, 0)
		return 0
	}
	// drop partially
	for _, t := range s.timeStamps[:n] {
		delete(s.entry, t.Format())
	}
	s.timeStamps = s.timeStamps[n:]
	return len(s.timeStamps)

}

// StreamID methods
// *****************

// Format return the string representation of a standard stream entry ID like "1667271690022-1"
func (i *StreamID) Format() string {
	return fmt.Sprintf("%d-%d", i.time, i.seqNum)
}

func (i *StreamID) GreaterEqual(ID *StreamID) bool {
	return i.time >= ID.time && i.seqNum >= ID.seqNum
}

// Parse read the string as a timestamp string like "1671073232746-0" into StreamID struct.
func (i *StreamID) Parse(text string) error {
	trunks := strings.Split(text, "-")
	var err error
	if len(trunks) > 2 {
		return errors.New("ERR Invalid stream ID specified as stream command argument")
	}
	// first part
	i.time, err = strconv.ParseInt(trunks[0], 10, 64)
	if err != nil {
		return errors.New("ERR Invalid stream ID specified as stream command argument")
	}
	// sequence number part
	if len(trunks) == 2 {
		i.seqNum, err = strconv.ParseInt(trunks[1], 10, 64)
		if err != nil {
			return errors.New("ERR Invalid stream ID specified as stream command argument")
		}
	} else {
		i.seqNum = 0
	}
	return nil
}
