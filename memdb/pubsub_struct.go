package memdb

import (
	"github.com/google/uuid"
	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
	"sync"
)

type ChanMap struct {
	item *ConcurrentMap
	rw   *sync.RWMutex
}

// Chan is the shard used in PUB/SUB... commands
type Chan struct {
	in      chan *ChanMsg
	out     map[string]chan *ChanMsg
	numSubs int
	rw      *sync.RWMutex
}

type ChanMsg struct {
	info string
	key  string
	val  any
}

func NewChanMap(shardNum int) *ChanMap {
	return &ChanMap{
		item: NewConcurrentMap(shardNum), // Chans are not initialized here
		rw:   &sync.RWMutex{},
	}
}

// Send sends a message to a channel and return the number of subscribers
func (m *ChanMap) Send(key string, val string) int {
	channelTmp, ok := m.item.Get(key)
	// if channel does not exist. do nothing
	if !ok {
		return 0
	}
	// send message and schedule broadcast event
	channel, isChannel := channelTmp.(*Chan)
	if !isChannel {
		logger.Error("Fatal error: the element in channel map is not a channel shard!")
		return 0
	} else {
		msg := &ChanMsg{
			info: "message",
			key:  key,
			val:  val,
		}
		channel.rw.Lock()
		channel.in <- msg
		channel.rw.Unlock()
		go channel.Broadcast()
		return channel.numSubs
	}
}

// Subscribe return a receiving chan and the ID of that chan based on the given key.
func (m *ChanMap) Subscribe(key string) (<-chan *ChanMsg, string) {
	channelTmp, ok := m.item.Get(key)
	var channel *Chan
	// channel does not exist => create one
	if !ok {
		channel = m.Create(key)
	} else {
		channel = channelTmp.(*Chan)
	}
	out := make(chan *ChanMsg)
	// lock a single channel shard since modifying attribute
	channel.rw.Lock()
	defer channel.rw.Unlock()
	// register subscriber
	channel.numSubs++
	outID := uuid.NewString()
	channel.out[outID] = out
	return out, outID
}

func (m *ChanMap) UnSubscribe(key string, ID string) {
	// get the channel
	channelTmp, ok := m.item.Get(key)
	var channel *Chan
	if !ok {
		logger.Error("Unsubscribing with non-existing channel ")
		return
	} else {
		channel = channelTmp.(*Chan)
	}
	// lock channel shard
	channel.rw.Lock()
	defer channel.rw.Unlock()
	// delete out record
	channel.numSubs--
	delete(channel.out, ID)
	// destroy channel with no subscribers to free memory
	if channel.numSubs == 0 {
		m.item.Delete(key)
	}
}

// Create make a shard and add it to the map if the key does not exist and return that channel shard.
func (m *ChanMap) Create(key string) *Chan {
	var newShard *Chan
	// create if not exist
	if _, ok := m.item.Get(key); !ok {
		newShard = &Chan{
			in:  make(chan *ChanMsg, config.Configures.ChanBufferSize),
			out: make(map[string]chan *ChanMsg, config.Configures.ChanBufferSize),
			rw:  &sync.RWMutex{},
		}
		m.item.Set(key, newShard)
	}
	return newShard
}

// Broadcast publish all the messages in {in} channel to all {out} channels
func (c *Chan) Broadcast() {
	// lock the shard
	c.rw.RLock()
	defer c.rw.RUnlock()
	// do broadcast
	for elem := range c.in {
		for _, outChan := range c.out {
			// "message", channel name, val
			outChan <- elem
		}
		break
	}
}
