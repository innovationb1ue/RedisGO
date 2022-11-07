package server

import (
	"bytes"
	"context"
	"fmt"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
)

// Start starts a simple redis server
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

	// cluster logic here *******************
	// cluster channels. we initialize them anyway since they won't take much memory
	proposeC := make(chan string, 5)
	defer close(proposeC)
	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)
	storage := raft.NewMemoryStorage()
	// node config
	c := &raft.Config{
		ID:              0x01,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         storage,
		MaxSizePerMsg:   4096,
		MaxInflightMsgs: 256,
	}
	// single node cluster
	peers := []raft.Peer{{ID: 0x01}}
	node := raft.StartNode(c, peers)
	// timer used in event loop
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// send proposals over raft
	go func() {
		confChangeCount := uint64(0)
		for proposeC != nil && confChangeC != nil {
			select {
			case prop, ok := <-proposeC:
				if !ok {
					proposeC = nil
				} else {
					// blocks until accepted by raft state machine
					node.Propose(context.TODO(), []byte(prop))
				}

			case cc, ok := <-confChangeC:
				if !ok {
					confChangeC = nil
				} else {
					confChangeCount++
					cc.ID = confChangeCount
					node.ProposeConfChange(context.TODO(), cc)
				}
			}
		}
	}()
	// event loop on raft state machine updates
	go func() {
		for {
			select {
			case <-ticker.C:
				node.Tick()

			// store raft entries to wal, then publish over commit channel
			case rd := <-node.Ready():
				// Must save the snapshot file and WAL snapshot entry before saving any other entries
				// or hardstate to ensure that recovery after a snapshot restore is possible.
				//if !raft.IsEmptySnap(rd.Snapshot) {
				//	saveSnap(rd.Snapshot)
				//}
				//rc.wal.Save(rd.HardState, rd.Entries)
				//if !raft.IsEmptySnap(rd.Snapshot) {
				//	rc.raftStorage.ApplySnapshot(rd.Snapshot)
				//	rc.publishSnapshot(rd.Snapshot)
				//}
				storage.Append(rd.Entries)

				for _, entry := range rd.CommittedEntries {
					// apply commited commands to state machine
					data := mgr.ExecCommand(ctx, bytes.Split(entry.Data, []byte{' '}), nil)
					log.Println(string(data.ByteData()))
					if entry.Type == raftpb.EntryConfChange {
						var cc raftpb.ConfChange
						cc.Unmarshal(entry.Data)
						node.ApplyConfChange(cc)
					}
				}

				//rc.transport.Send(rc.processMessages(rd.Messages))
				//applyDoneC, ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries))
				//if !ok {
				//	rc.stop()
				//	return
				//}
				//rc.maybeTriggerSnapshot(applyDoneC)
				node.Advance()
				//case err := <-rc.transport.ErrorC:
				//	rc.writeError(err)
				//	return
				//
				//case <-rc.stopc:
				//	rc.stop()
				//	return
			}
		}
	}()
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
				// decide the right handler to process command
				if cfg.IsCluster {
					mgr.HandleCluster(ctx, conn, proposeC)
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
		}
	}
}
