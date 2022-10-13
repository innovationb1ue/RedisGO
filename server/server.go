package server

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
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
	// shutting down everything
	defer func() {
		log.Println("shutting down gracefully")
		err := listener.Close()
		if err != nil {
			logger.Error(err)
		}
	}()

	logger.Info("Server Listen at ", cfg.Host, ":", cfg.Port)
	// handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	// client chan
	clients := make(chan net.Conn)
	handler := NewHandler()
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

	for {
		select {
		// spawn a goroutine to handle client commands
		case conn := <-clients:
			logger.Info(conn.RemoteAddr().String(), " connected")
			// start the worker goroutine
			go func() {
				handler.Handle(conn)
			}()
		// exit server
		case sig := <-sigs:
			if sig == syscall.SIGTERM || sig == syscall.SIGINT {
				return nil
			}
		}
	}
}
