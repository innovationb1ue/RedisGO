package server

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/innovationb1ue/RedisGO/config"
	"github.com/innovationb1ue/RedisGO/logger"
)

// Start starts a simple redis server
func Start(cfg *config.Config) error {
	clientClose := make([]chan struct{}, 0)
	// open tcp port
	listener, err := net.Listen("tcp", cfg.Host+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		logger.Panic(err)
		return err
	}
	// create client disconnect wait group
	wg := sync.WaitGroup{}
	// shutting down everything
	defer func() {
		log.Println("shutting down gracefully")
		// 1. close tcp port
		err := listener.Close()
		if err != nil {
			logger.Error(err)
		}
		// 2. send close to clients
		for _, c := range clientClose {
			close(c)
		}
		// 3. wait for all clients to disconnect
		wg.Wait()
	}()

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
			if err != nil {
				logger.Error(err)
			}
			clients <- conn
		}
	}()
	// server event loop
	for {
		select {
		// spawn a goroutine to handle client commands
		case conn := <-clients:
			logger.Info(conn.RemoteAddr().String(), " connected")
			closeChan := make(chan struct{})
			clientClose = append(clientClose, closeChan)
			// start the worker goroutine
			go func() {
				defer wg.Done()
				mgr.Handle(conn, closeChan)
			}()
			wg.Add(1)
		// exit server
		case sig := <-sigs:
			if sig == syscall.SIGTERM || sig == syscall.SIGINT {
				return nil
			}
		}
	}
}
