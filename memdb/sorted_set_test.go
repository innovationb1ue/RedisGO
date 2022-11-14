package memdb

import (
	"context"
	"fmt"
	"github.com/innovationb1ue/RedisGO/resp"
	"strconv"
	"strings"
	"testing"
)

var ctx context.Context

func init() {
	ctx = context.Background()
}

func MakeTestDB() *MemDb {
	m := NewMemDb()
	zadd(ctx, m, MakeCommandBytes("zadd a 555 hero"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 333 ggbob"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 444 jeff"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 666 king"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a -999 loser"), nil)
	// test change a member value
	zadd(ctx, m, MakeCommandBytes("zadd a 999 ggbob"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 888 a1"), nil)
	// -999 444 555 666 888 999 with a total of 6
	return m
}

func TestZADD(t *testing.T) {
	MakeTestDB()
}

func TestZRange(t *testing.T) {
	// need to make sure that we have put the correct values in it
	m := MakeTestDB()
	// test zrange
	res := zrange(ctx, m, MakeCommandBytes("zrange a 0 100 withscores"), nil)
	resArr := res.(*resp.ArrayData)
	resStrings := resArr.ToStringCommand()
	fmt.Println(resStrings)
	if resStrings[0] != "loser" {
		t.Error("zrange order is wrong")
	}
	if score, err := strconv.ParseFloat(resStrings[1], 64); err != nil || score != -999 {
		t.Error("zrange score is wrong, expect ", -999, "got", score)
	}
	if resStrings[2] != "jeff" {
		t.Error("zrange order is wrong")
	}
	if score, err := strconv.ParseFloat(resStrings[3], 64); err != nil || score != 444 {
		t.Error("zrange score is wrong")
	}
	if resStrings[len(resStrings)-2] != "ggbob" {
		t.Error("zrange order is wrong")
	}
	if score, err := strconv.ParseFloat(resStrings[len(resStrings)-1], 64); err != nil || score != 999 {
		t.Error("zrange score is wrong")
	}
}

func TestZREM(t *testing.T) {
	m := MakeTestDB()
	zrem(ctx, m, MakeCommandBytes("zrem a ggbob"), nil)
	res := zrange(ctx, m, MakeCommandBytes("zrange a 0 100 withscores"), nil)
	if strings.Contains(string(res.ByteData()), "ggbob") {
		t.Error("ZREM does not delete the key")
		t.FailNow()
	}
	zrem(ctx, m, MakeCommandBytes("zrem a jeff"), nil)
	res = zrange(ctx, m, MakeCommandBytes("zrange a 0 100 withscores"), nil)
	if strings.Contains(string(res.ByteData()), "jeff") {
		t.Error("ZREM does not delete the key")
		t.FailNow()
	}
}
