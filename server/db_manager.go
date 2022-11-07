package server

import (
	"context"
	"fmt"
	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/memdb"
	"github.com/innovationb1ue/RedisGO/resp"
	"io"
	"net"
	"strconv"
	"strings"
)

// Manager handles all client requests to the server
// It holds multiple MemDb instances
type Manager struct {
	memDb *memdb.MemDb
	DBs   []*memdb.MemDb
}

type MemStorageStats struct {
	initialState, firstIndex, lastIndex, entries, term, snapshot int
}

// NewManager creates a default Manager
func NewManager(cfg *config.Config) *Manager {
	DBs := make([]*memdb.MemDb, cfg.Databases)
	for i := 0; i < cfg.Databases; i++ {
		DBs[i] = memdb.NewMemDb()
	}
	return &Manager{
		memDb: DBs[0],
		DBs:   DBs,
	}
}

// Handle distributes all the client command to execute
func (m *Manager) Handle(ctx context.Context, conn net.Conn) {
	// gracefully close the tcp connection to client
	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Error(err)
		}
	}()
	// create a goroutine that reads from the client and pump data into ch
	ch := resp.ParseStream(ctx, conn)
	// parsedRes is a complete command read from client
	for {
		select {
		case parsedRes := <-ch:
			// handle errors
			if parsedRes.Err != nil {
				if parsedRes.Err == io.EOF {
					logger.Info("Close connection ", conn.RemoteAddr().String())
				} else {
					logger.Panic("Handle connection ", conn.RemoteAddr().String(), " panic: ", parsedRes.Err.Error())
				}
				return
			}
			// empty msg
			if parsedRes.Data == nil {
				logger.Error("empty parsedRes.Data from ", conn.RemoteAddr().String())
				continue
			}
			// handling array command
			arrayData, ok := parsedRes.Data.(*resp.ArrayData)
			if !ok {
				logger.Error("parsedRes.Data is not ArrayData from ", conn.RemoteAddr().String())
				continue
			}
			// extract [][]bytes command
			cmd := arrayData.ToCommand()
			// run the string command when in standalone mode
			// also pass connection as an argument since the command may block and return continuous messages
			res := m.ExecCommand(ctx, cmd, conn)
			// return result
			if res != nil {
				_, err := conn.Write(res.ToBytes())
				if err != nil {
					logger.Error("write response to ", conn.RemoteAddr().String(), " error: ", err.Error())
				}
			} else {
				// return error
				errData := resp.MakeErrorData("unknown error")
				_, err := conn.Write(errData.ToBytes())
				if err != nil {
					logger.Error("write response to ", conn.RemoteAddr().String(), " error: ", err.Error())
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) ExecCommand(ctx context.Context, cmd [][]byte, conn net.Conn) resp.RedisData {
	if len(cmd) == 0 {
		return nil
	}
	var res resp.RedisData
	cmdName := strings.ToLower(string(cmd[0]))
	// global commands
	switch cmdName {
	case "select":
		return m.Select(cmd)
	}
	// get the command from hash table and execute it.
	command, ok := memdb.CmdTable[cmdName]
	if !ok {
		res = resp.MakeErrorData("ERR unknown command ", cmdName)
	} else {
		res = command.Executor(ctx, m.memDb, cmd, conn)
	}
	return res
}

func (m *Manager) Select(cmd [][]byte) resp.RedisData {
	if len(cmd) != 2 {
		return resp.MakeWrongNumberArgs("select")
	}
	dbIdx, err := strconv.Atoi(string(cmd[1]))
	if err != nil {
		return resp.MakeErrorData("ERR value is not an integer or out of range")
	}
	if dbIdx >= len(m.DBs) || dbIdx < 0 {
		return resp.MakeErrorData(fmt.Sprintf("ERR DB index is out of range with maximum %d", len(m.DBs)))
	}
	m.memDb = m.DBs[dbIdx]
	return resp.MakeStringData("OK")
}

func (m *Manager) HandleCluster(ctx context.Context, conn net.Conn, propose chan<- string) {
	// gracefully close the tcp connection to client
	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Error(err)
		}
	}()
	// create a goroutine that reads from the client and pump data into ch
	ch := resp.ParseStream(ctx, conn)
	// parsedRes is a complete command read from client
	for {
		select {
		case parsedRes := <-ch:
			// handle errors
			if parsedRes.Err != nil {
				if parsedRes.Err == io.EOF {
					logger.Info("Close connection ", conn.RemoteAddr().String())
				} else {
					logger.Panic("Handle connection ", conn.RemoteAddr().String(), " panic: ", parsedRes.Err.Error())
				}
				return
			}
			// empty msg
			if parsedRes.Data == nil {
				logger.Error("empty parsedRes.Data from ", conn.RemoteAddr().String())
				continue
			}
			// handling array command
			arrayData, ok := parsedRes.Data.(*resp.ArrayData)
			if !ok {
				logger.Error("parsedRes.Data is not ArrayData from ", conn.RemoteAddr().String())
				continue
			}
			// extract [][]bytes command
			cmdStrings := arrayData.ToStringCommand()
			// propose command to raft cluster
			propose <- strings.Join(cmdStrings, " ")
			conn.Write(resp.MakeStringData("OK").ToBytes())
			// todo: fetch command state from leader and return response
		case <-ctx.Done():
			return
		}
	}
}
