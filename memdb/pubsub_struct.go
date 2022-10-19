package memdb

import (
	"github.com/google/uuid"
	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"sync"
)

type ChanMap struct {
	item *ConcurrentMap
	rw   *sync.RWMutex
}

// Chan is the shard used in PUB/SUB... commands
type Chan struct {
	in      chan *ChanMsg
	conns   map[string]net.Conn
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
	channel := channelTmp.(*Chan)
	msg := &ChanMsg{
		info: "message",
		key:  key,
		val:  val,
	}
	// block sending messages since we need to get the number of active clients
	for k, c := range channel.conns {
		_, err := c.Write(resp.MakeArrayData([]resp.RedisData{resp.MakeBulkData([]byte(msg.info)),
			resp.MakeBulkData([]byte(msg.key)),
			resp.MakeBulkData([]byte(msg.val.(string))),
		}).ToBytes())
		if err != nil {
			_ = c.Close()
			delete(channel.conns, k)
			channel.numSubs--
		}
	}
	return channel.numSubs

}

// Subscribe return a receiving chan and the ID of that chan based on the given key.
func (m *ChanMap) Subscribe(key string, conn net.Conn) string {
	channelTmp, ok := m.item.Get(key)
	var channel *Chan
	// channel does not exist => create one
	if !ok {
		channel = m.Create(key)
	} else {
		channel = channelTmp.(*Chan)
	}
	// lock a single channel shard since modifying attribute
	channel.rw.Lock()
	defer channel.rw.Unlock()
	connID := uuid.NewString()
	channel.conns[connID] = conn
	// register subscriber
	channel.numSubs++
	return connID
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
	// lock channel since modifying map
	channel.rw.Lock()
	defer channel.rw.Unlock()
	// delete out record
	channel.numSubs--
	delete(channel.conns, ID)
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
			in:      make(chan *ChanMsg, config.Configures.ChanBufferSize),
			conns:   make(map[string]net.Conn, 0),
			numSubs: 0,
			rw:      &sync.RWMutex{},
		}
		m.item.Set(key, newShard)
	}
	return newShard
}
