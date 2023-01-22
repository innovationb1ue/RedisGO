package server

import (
	"errors"
	"reflect"
	"strings"
)

type middleware struct {
	filters []func([][]byte) ([][]byte, error)
}

// filter function alias
type filterFunc = func([][]byte) ([][]byte, error)

// filter functions will get passed through before each command get executed
// return the modified command and possible failure with an error
func newMiddleware() *middleware {
	return &middleware{filters: make([]func([][]byte) ([][]byte, error), 0)}
}

func (m *middleware) Add(f func([][]byte) ([][]byte, error)) {
	m.filters = append(m.filters, f)
}

func (m *middleware) Delete(f filterFunc) bool {
	for i, candidate := range m.filters {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(candidate).Pointer() {
			m.filters = append(m.filters[:i], m.filters[i+1:]...)
			return true
		}
	}
	return false
}

func (m *middleware) Filter(cmd [][]byte) ([][]byte, error) {
	var err error = nil
	for _, f := range m.filters {
		cmd, err = f(cmd)
		if err != nil {
			return nil, err
		}
	}
	return cmd, err
}

// ***********************************
// possibly write global filters below

var Filters = []func(cmd [][]byte) ([][]byte, error){ClusterCmdFilter}

func ClusterCmdFilter(cmd [][]byte) ([][]byte, error) {
	command := strings.ToLower(string(cmd[0]))
	if command == "publish" || command == "subscribe" {
		return nil, errors.New("does not support pub/sub in cluster mode yet. ")
	}
	return cmd, nil
}
