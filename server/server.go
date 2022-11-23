package server

import (
	"context"
	"fmt"
	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/raftexample"
	"github.com/innovationb1ue/RedisGO/resp"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Start starts a redis server and raft layer if in cluster mode
func Start(cfg *config.Config) error {
	// open tcp port
	listener, err := net.Listen("tcp", cfg.Host+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		logger.Panic(err)
		return err
	}
	var isTerminating bool
	// create client disconnect wait group
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// shutting down everything
	defer func() {
		logger.Info("shutting down gracefully")
		// 1. close listening tcp port
		err := listener.Close()
		if err != nil {
			logger.Error(err)
		}
		// 2. shut down client goroutines (send disconnect msg)
		cancel()
		// 3. wait for all clients to disconnect
		wg.Wait()
		logger.Info("See you again. ")
	}()
	// welcome text
	fmt.Println("\n██████╗░███████╗██████╗░██╗░██████╗░██████╗░░█████╗░\n██╔══██╗██╔════╝██╔══██╗██║██╔════╝██╔════╝░██╔══██╗\n██████╔╝█████╗░░██║░░██║██║╚█████╗░██║░░██╗░██║░░██║\n██╔══██╗██╔══╝░░██║░░██║██║░╚═══██╗██║░░╚██╗██║░░██║\n██║░░██║███████╗██████╔╝██║██████╔╝╚██████╔╝╚█████╔╝\n╚═╝░░╚═╝╚══════╝╚═════╝░╚═╝╚═════╝░░╚═════╝░░╚════╝░")
	logger.Info("Server Listen at ", cfg.Host, ":", cfg.Port)
	// handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	// client chan
	clients := make(chan net.Conn)
	// create n db for SELECT cmd
	mgr := NewManager(cfg)

	// spawn a worker to accept tcp connections & create client objects
	go func() {
		for {
			conn, err := listener.Accept()
			if isTerminating {
				close(clients)
				return
			}
			if err != nil {
				logger.Error(err)
				return
			}
			clients <- conn
		}
	}()

	// cluster logic here *******************
	var proposeC chan *raftexample.RaftProposal
	var confChangeC chan raftpb.ConfChangeI
	var commitC <-chan *raftexample.RaftCommit
	var errorC <-chan error
	var snapshotterReady <-chan *snap.Snapshotter
	var resultCallback map[string]chan resp.RedisData
	var clusterFilter *middleware
	var RaftNode *raftexample.RaftNode
	if cfg.IsCluster {
		logger.Info("Initializing cluster node")
		// send to proposeC to send message to raft cluster
		proposeC = make(chan *raftexample.RaftProposal)
		confChangeC = make(chan raftpb.ConfChangeI)
		defer close(proposeC)
		defer close(confChangeC)
		resultCallback = make(map[string]chan resp.RedisData)
		// start raft node
		getSnapshot := func() ([]byte, error) { return mgr.CurrentDB.GetSnapshot() }
		// read from commitC to update state machine
		commitC, errorC, snapshotterReady, RaftNode = raftexample.NewRaftNode(cfg.NodeID, cfg.RaftAddr, strings.Split(cfg.PeerAddrs, ","), cfg.JoinCluster, getSnapshot, proposeC, confChangeC)
		<-snapshotterReady
		mgr.CurrentDB.Raft = RaftNode
		go handleClusterCommits(ctx, commitC, confChangeC, mgr, resultCallback, errorC)
		// build cluster command filter
		clusterFilter = newMiddleware()
		clusterFilter.Add(ClusterCmdFilter)
	}

	// server event loop
	for {
		select {
		// spawn a goroutine to handle client commands
		case conn, ok := <-clients:
			if !ok {
				return nil
			}
			logger.Info(conn.RemoteAddr().String(), " connected")
			// start the worker goroutine
			go func() {
				defer wg.Done()
				log.Println("start one worker")
				// decide the right handler to process command
				if cfg.IsCluster {
					mgr.HandleCluster(ctx, conn, proposeC, confChangeC, resultCallback, clusterFilter)
				} else {
					mgr.Handle(ctx, conn)
				}
			}()
			wg.Add(1)
		// exit server
		case sig := <-sigs:
			if sig == syscall.SIGTERM || sig == syscall.SIGINT {
				logger.Info("Terminate")
				isTerminating = true
				return nil
			}
		// error in raft cluster
		case <-errorC:
			return nil
		}

	}
}

func handleClusterCommits(ctx context.Context, commitC <-chan *raftexample.RaftCommit, confChangeC chan<- raftpb.ConfChangeI, dbMgr *Manager, resultCallback map[string]chan resp.RedisData, errorC <-chan error) {
	for msg := range commitC {
		log.Println("commitC receive ", msg)
		if msg == nil {
			log.Println("loaded empty snapshot")
			continue
		}
		for _, cmd := range msg.Data {
			ctx = context.WithValue(ctx, "confChangeC", confChangeC)
			res := dbMgr.ExecStrCommand(ctx, cmd.Data, nil)
			if callback, ok := resultCallback[cmd.ID]; ok {
				callback <- res
			}
			log.Println("cluster commitC: exec command ", msg.Data, "result = ", res)
		}
		close(msg.ApplyDoneC)
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}
