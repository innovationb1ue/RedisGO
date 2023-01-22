package server

import (
	"github.com/innovationb1ue/RedisGO/memdb"
	"github.com/stretchr/testify/assert"
	"testing"
)

var middlewareInstance = newMiddleware()

// test command bytes to get pass through a simple filter
func TestMiddleware_Add(t *testing.T) {
	// increase this counter in the filter function
	callCount := 0
	// simple fileter function
	simpleFilter := func(cmd [][]byte) ([][]byte, error) {
		callCount += 1
		return cmd, nil
	}
	middlewareInstance.Add(simpleFilter)

	cmd := memdb.MakeCommandBytes("should get passed")
	newCmd, err := middlewareInstance.Filter(cmd)
	if err != nil {
		t.Error("this should get pass the filter without any error. Possible filter logic is broken")
	}
	assert.Equal(t, 1, callCount)
	assert.Equal(t, newCmd, memdb.MakeCommandBytes("should get passed"))
}

func TestDeleteMiddleware(t *testing.T) {
	callCount := 0
	simpleFilter := func(cmd [][]byte) ([][]byte, error) {
		callCount += 1
		return cmd, nil
	}
	middlewareInstance.Add(simpleFilter)
	middlewareInstance.Delete(simpleFilter)
	cmd := memdb.MakeCommandBytes("should get passed")
	newCmd, err := middlewareInstance.Filter(cmd)
	if err != nil {
		t.Error("Delete logic is broken")
	}
	assert.Equal(t, 0, callCount, "filter function is not deleted after calling delete")
	assert.Equal(t, newCmd, memdb.MakeCommandBytes("should get passed"))
}
