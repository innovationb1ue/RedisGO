package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/innovationb1ue/RedisGO/logger"
)

// resp package for parsing redis serialization protocol.
// Check https://redis.io/docs/reference/protocol-spec/ for the protocol details.

type ParsedRes struct {
	Data RedisData
	Err  error
}

type readState struct {
	bulkLen   int64
	arrayLen  int
	multiLine bool
	arrayData *ArrayData
	inArray   bool
}

func ParseStream(reader io.Reader) <-chan *ParsedRes {
	ch := make(chan *ParsedRes)
	go parse(reader, ch)
	return ch
}

// ReadAll is copied from io and modified to suit the context
func ReadAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 1024)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			// need to handle the EOF
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

func newParse(reader io.Reader, ch chan<- *ParsedRes) {
	for {
		// read all bytes
		msg, err := ReadAll(reader)
		if err != nil {
			logger.Warning("ReadAll err", err)
		}
		log.Println("parse received message ", string(msg))

		ch <- &ParsedRes{
			Data: MakeBulkData(msg),
			Err:  nil,
		}
	}
}

func parse(reader io.Reader, ch chan<- *ParsedRes) {
	bufReader := bufio.NewReaderSize(reader, 4096) // 4096 is the default bufio buffer size, might change to a smaller number by configuration
	state := new(readState)
	for {
		// continuously read until client sent EOF to disconnect
		var res RedisData
		var err error
		var msg []byte
		// read until "\r\n"
		msg, err = readLine(bufReader, state)
		// handle errors
		if err != nil {
			// read ended, stop reading.
			if err == io.EOF {
				ch <- &ParsedRes{
					Err: io.EOF,
				}
				close(ch)
				return
			} else {
				// Protocol error
				logger.Error(err)
				ch <- &ParsedRes{
					Err: err,
				}
				*state = readState{}
			}
			continue
		}
		// parse the read messages
		// if msg is an array or a bulk string, then parse their header first.
		// if msg is a normal line, parse it directly.
		if !state.multiLine {
			switch msg[0] {
			// Arrays
			case '*':
				// get array length. typically this is the number of command segments
				err := parseArrayHeader(msg, state)
				if err != nil {
					logger.Error(err)
					ch <- &ParsedRes{
						Err: err,
					}
					*state = readState{}
				} else {
					if state.arrayLen == -1 {
						// null array
						ch <- &ParsedRes{
							Data: MakeArrayData(nil),
						}
						*state = readState{}
					} else if state.arrayLen == 0 {
						// empty array
						ch <- &ParsedRes{
							Data: MakeArrayData([]RedisData{}),
						}
						*state = readState{}
					}
				}
				// continue to read the array elements
				continue
			// Bulk Strings
			case '$':
				// modify state.bulkLen and state.multiline
				err := parseBulkHeader(msg, state)
				if err != nil {
					logger.Error(err)
					ch <- &ParsedRes{
						Err: err,
					}
					*state = readState{}
				} else {
					if state.bulkLen == -1 {
						// null bulk string (nil)
						state.multiLine = false
						state.bulkLen = 0
						res = MakeBulkData(nil)
						// nil as array element
						if state.inArray {
							state.arrayData.data = append(state.arrayData.data, res)
							if len(state.arrayData.data) == state.arrayLen {
								ch <- &ParsedRes{
									Data: state.arrayData,
									Err:  nil,
								}
								*state = readState{}
							}
						} else {
							ch <- &ParsedRes{
								Data: res,
							}
						}
					}
				}
				continue
			default:
				// text (not been used)
				res, err = parseSingleLine(msg)
			}
		} else {
			// parse multiple lines: bulk string (binary safe)
			state.multiLine = false
			state.bulkLen = 0
			// read the actual string after control headers.
			res, err = parseMultiLine(msg)
		}
		// check for read error
		if err != nil {
			logger.Error(err)
			ch <- &ParsedRes{
				Err: err,
			}
			*state = readState{}
			continue
		}

		// Struct parsed data as an array or a single data, and put it into channel.
		if state.inArray {
			// append actual command string to command array and read next command string if any.
			state.arrayData.data = append(state.arrayData.data, res)
			// if array is finished, send to channel
			if len(state.arrayData.data) == state.arrayLen {
				ch <- &ParsedRes{
					Data: state.arrayData,
					Err:  nil,
				}
				*state = readState{}
				continue
			}
		} else {
			// not receiving array command, send read result to channel directly.
			ch <- &ParsedRes{
				Data: res,
				Err:  err,
			}
		}

	}
}

// Read a line or bulk line end of "\r\n" from a reader.
// Return:
//
//	[]byte: read bytes.
//	error: io.EOF or Protocol error
func readLine(reader *bufio.Reader, state *readState) ([]byte, error) {
	var msg []byte
	var err error
	if state.multiLine && state.bulkLen >= 0 {
		// read bulk line, binary safety. read exactly {bulkLen} bytes from the reader.
		msg = make([]byte, state.bulkLen+2) // +2 is for the space of "\r\n"
		_, err = io.ReadFull(reader, msg)
		if err != nil {
			return nil, err
		}
		state.bulkLen = 0
		if msg[len(msg)-1] != '\n' || msg[len(msg)-2] != '\r' {
			return nil, errors.New(fmt.Sprintf("Protocol error. Stream message %s is invalid.", string(msg)))
		}
	} else {
		// read normal line
		msg, err = reader.ReadBytes('\n')
		if err != nil {
			return msg, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' {
			return nil, errors.New(fmt.Sprintf("Protocol error. Stream message %s is invalid.", string(msg)))
		}
	}
	return msg, nil
}

func parseSingleLine(msg []byte) (RedisData, error) {
	// msg is like "*This is a string. \r\n"
	msgType := msg[0]
	// the actual string content without the first indicator and \r\n at the end
	msgData := string(msg[1 : len(msg)-2])
	var res RedisData

	switch msgType {
	case '+':
		// simple string
		res = MakeStringData(msgData)
	case '-':
		// error
		res = MakeErrorData(msgData)
	case ':':
		// integer
		data, err := strconv.ParseInt(msgData, 10, 64)
		if err != nil {
			logger.Error("Cant phrase int64 from " + msgData + " where error: " + string(msg))
			return nil, err
		}
		res = MakeIntData(data)
	default:
		// plain string
		res = MakePlainData(msgData)
	}
	// likely never be nil
	if res == nil {
		logger.Error("Protocol error: parseSingleLine get nil data")
		return nil, errors.New("Protocol error: " + string(msg))
	}
	return res, nil
}

// parseMultiLine parses the second part of a single command string. i.e. the "incr\r\n" in "$4\r\nincr\r\n"
func parseMultiLine(msg []byte) (RedisData, error) {
	if len(msg) < 2 {
		return nil, errors.New("protocol error: invalid bulk string")
	}
	// discard "\r\n" at the end
	msgData := msg[:len(msg)-2]
	res := MakeBulkData(msgData)
	return res, nil
}

func parseArrayHeader(msg []byte, state *readState) error {
	// get the array length
	arrayLen, err := strconv.Atoi(string(msg[1 : len(msg)-2]))
	if err != nil || arrayLen < 0 {
		return errors.New("Protocol error: " + string(msg))
	}
	state.arrayLen = arrayLen
	state.inArray = true
	state.arrayData = MakeArrayData([]RedisData{})
	return nil
}

func parseBulkHeader(msg []byte, state *readState) error {
	// typically read bulk string which might contains "\r\n"
	// get length
	bulkLen, err := strconv.ParseInt(string(msg[1:len(msg)-2]), 10, 64)
	if err != nil || bulkLen < -1 {
		return errors.New("Protocol error: " + string(msg))
	}
	// indicate length and indicate the read function not to stop when encountering "\r\n" till the length exhausted.
	state.bulkLen = bulkLen
	state.multiLine = true
	return nil
}
