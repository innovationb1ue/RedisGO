package memdb

import (
	"context"
	"github.com/innovationb1ue/RedisGO/resp"
	"net"
	"strings"
)

type cmdBytes = [][]byte

// CmdTable holds all registered commands
var CmdTable = make(map[string]*command)

// We allow executor to directly write message back to the tcp connection for some blocking commands.
// But it should never be spoilt. Normal commands should always return a data but not write
// into the pipe by themselves.
type cmdExecutor func(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData

type command struct {
	executor cmdExecutor
}

func RegisterCommand(cmdName string, executor cmdExecutor) {
	CmdTable[cmdName] = &command{
		executor: executor,
	}
}

func MakeCommandBytes(input string) [][]byte {
	cmdStrs := strings.Split(input, " ")
	cmds := make(cmdBytes, 0)
	for _, c := range cmdStrs {
		cmds = append(cmds, []byte(c))
	}
	return cmds
}
