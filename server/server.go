package server

import (
	"context"
	"fmt"
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
				mgr.Handle(ctx, conn)
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

func StartCluster(cfg *config.Config) error {
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
}
