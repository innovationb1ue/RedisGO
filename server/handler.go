package server

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/innovationb1ue/RedisGO/logger"
	"github.com/innovationb1ue/RedisGO/memdb"
	"github.com/innovationb1ue/RedisGO/resp"
)

// Handler handles all client requests to the server
// It holds a MemDb instance to exchange data with clients
type Handler struct {
	memDb *memdb.MemDb
}

// NewHandler creates a default Handler
func NewHandler() *Handler {
	return &Handler{
		memDb: memdb.NewMemDb(),
	}
}

// Handle distributes all the client command to execute
func (h *Handler) Handle(conn net.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	// gracefully close the tcp connection to client
	defer func() {
		cancel()
		err := conn.Close()
		if err != nil {
			logger.Error(err)
		}
		log.Println("stop handle one")
	}()
	// create a goroutine that reads from the client and pump data into ch
	ch := resp.ParseStream(conn)
	// parsedRes is a complete command read from client
	for parsedRes := range ch {
		// handle errors
		if parsedRes.Err != nil {
			if parsedRes.Err == io.EOF {
				logger.Info("Close connection ", conn.RemoteAddr().String())
			} else {
				logger.Panic("Handle connection ", conn.RemoteAddr().String(), " panic: ", parsedRes.Err.Error())
			}
			return
		}
		log.Println(string(parsedRes.Data.ByteData()))
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
		// run the string command
		// also pass connection as an argument since the command may block and return continuous messages
		res := h.memDb.ExecCommand(ctx, cmd, conn)
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

	}

}
