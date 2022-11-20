package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"net"
	"strings"
)

import (
	"context"
)

// rconf stands for raft configuration.
func rconf(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	confCTmp := ctx.Value("confChangeC")
	var confChangeC chan<- raftpb.ConfChangeI
	var ok bool
	confChangeC, ok = confCTmp.(chan<- raftpb.ConfChangeI)
	if !ok {
		return resp.MakeErrorData("can not resolve conf change channel.l Make sure server runs in cluster mode. ")
	}
	cc := raftpb.ConfChangeSingle{
		Type:   raftpb.ConfChangeAddNode,
		NodeID: 0,
	}
	changes := raftpb.ConfChangeV2{
		Transition: 0,
		Changes:    []raftpb.ConfChangeSingle{cc},
		Context:    nil,
	}
	confChangeC <- changes
	return nil
}

func Member(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[1])) == "list" {
		return MemberList(ctx, m, cmd, conn)
	} else {
		return resp.MakeErrorData("command not found for member ", string(cmd[1]))
	}
}

func MemberList(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	node := m.Raft
	if node == nil {
		return resp.MakeErrorData("raft node is not initialized. ")
	}
	peers := m.Raft.Peers
	res := make([]resp.RedisData, 0, len(peers))
	for _, k := range peers {
		res = append(res, resp.MakeStringData(k))
	}
	return resp.MakeArrayData(res)
}

func RegisterRaftCommand() {
	RegisterCommand("rconf", rconf)
	RegisterCommand("member", Member)
}
