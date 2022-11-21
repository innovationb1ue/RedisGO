package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/memdb"
	"github.com/innovationb1ue/RedisGO/raftexample"
	"github.com/innovationb1ue/RedisGO/resp"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"io"
	"net"
	"strconv"
	"strings"
)

// Manager handles all client requests to the server
// It holds multiple MemDb instances
type Manager struct {
	CurrentDB *memdb.MemDb
	DBs       []*memdb.MemDb
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
		CurrentDB: DBs[0],
		DBs:       DBs,
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
		res = command.Executor(ctx, m.CurrentDB, cmd, conn)
	}
	return res
}

func (m *Manager) ExecStrCommand(ctx context.Context, cmdStr string, conn net.Conn) resp.RedisData {
	cmd := strings.Split(cmdStr, " ")
	if len(cmd) == 0 {
		return nil
	}
	var res resp.RedisData
	cmdName := strings.ToLower(cmd[0])
	// global commands
	byteCmd := make([][]byte, 0)
	for _, s := range cmd {
		byteCmd = append(byteCmd, []byte(s))
	}
	switch cmdName {
	case "select":
		return m.Select(byteCmd)
	}
	// get the command from hash table and execute it.
	command, ok := memdb.CmdTable[cmdName]
	if !ok {
		res = resp.MakeErrorData("ERR unknown command ", cmdName)
	} else {
		res = command.Executor(ctx, m.CurrentDB, byteCmd, conn)
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
	m.CurrentDB = m.DBs[dbIdx]
	return resp.MakeStringData("OK")
}

func (m *Manager) HandleCluster(ctx context.Context, conn net.Conn, proposeC chan<- *raftexample.RaftProposal, confChangeC chan<- raftpb.ConfChangeI, callback map[string]chan resp.RedisData, filter *middleware) {
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
			// get command passed through filters
			var err error
			cmd, err = filter.Filter(cmd)
			if err != nil {
				logger.Error("filter error ", err)
				// return error
				errData := resp.MakeErrorData("command does not pass checks")
				_, err := conn.Write(errData.ToBytes())
				if err != nil {
					logger.Error("write response to ", conn.RemoteAddr().String(), " error: ", err.Error())
				}
				continue
			}
			cmdStrings := arrayData.ToStringCommand()

			// confChange command
			// todo: temporary workaround for confChange propose
			if cmdStrings[0] == "rconf" {
				ctx = context.WithValue(ctx, "confChangeC", confChangeC)
				res := m.ExecCommand(ctx, cmd, conn)
				_, err := conn.Write(res.ToBytes())
				if err != nil {
					logger.Error("write response to ", conn.RemoteAddr().String(), " error: ", err.Error())
				}
				continue
			}

			// proposeC command to raft cluster
			cmdID := uuid.NewString()
			callback[cmdID] = make(chan resp.RedisData)

			proposal := &raftexample.RaftProposal{
				Data: strings.Join(cmdStrings, " "),
				ID:   cmdID,
			}
			proposeC <- proposal
			// todo: this never return for newly joined node
			res := <-callback[cmdID]
			delete(callback, cmdID)
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
