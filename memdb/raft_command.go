package memdb

import (
	"github.com/innovationb1ue/RedisGO/resp"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"log"
	"net"
	"strconv"
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
	// todo: check args
	confChangeC, ok = confCTmp.(chan<- raftpb.ConfChangeI)
	if !ok {
		return resp.MakeErrorData("ERR can not resolve conf change channel.l Make sure server runs in cluster mode. ")
	}
	var actType raftpb.ConfChangeType
	switch strings.ToLower(string(cmd[1])) {
	case "add":
		actType = raftpb.ConfChangeAddNode
		log.Println(actType)
	default:
		return resp.MakeErrorData("unrecognized key")
	}
	nodeID, err := strconv.ParseUint(string(cmd[2]), 10, 64)
	if err != nil {
		return resp.MakeErrorData("ERR value is not integer or out of range")
	}
	if nodeID < 0 {
		return resp.MakeErrorData("ID should be a positive integer")
	}
	url := string(cmd[3])
	//cc := raftpb.ConfChangeSingle{
	//	Type:   actType,
	//	NodeID: nodeID,
	//}
	//changev2 := raftpb.ConfChangeV2{
	//	Transition: 2,
	//	Changes:    []raftpb.ConfChangeSingle{cc},
	//	Context:    []byte(url),
	//}

	change := raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  nodeID,
		Context: []byte(url),
	}

	confChangeC <- change
	//confChangeC <- changev2

	return resp.MakeStringData("send changev2 to confChange")
}

// Member is designed to be a single arg command
func Member(ctx context.Context, m *MemDb, cmd [][]byte, conn net.Conn) resp.RedisData {
	if strings.ToLower(string(cmd[1])) == "list" {
		return MemberList(ctx, m, cmd, conn)
	} else {
		return resp.MakeErrorData("command not found for member ", string(cmd[1]))
	}
}

// MemberList implementing 'list' option of Member command
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
