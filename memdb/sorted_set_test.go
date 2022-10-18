package memdb

import "testing"

func TestZADD(t *testing.T) {
	m := NewMemDb()
	zadd(m, MakeCommandBytes("zadd a 555 hero"), nil)
	zadd(m, MakeCommandBytes("zadd a 333 ggbob"), nil)
	zadd(m, MakeCommandBytes("zadd a 444 jeff"), nil)

	zadd(m, MakeCommandBytes("zadd a 666 king"), nil)
	zadd(m, MakeCommandBytes("zadd a -999 loser"), nil)
	// test change a member value
	zadd(m, MakeCommandBytes("zadd a 999 ggbob"), nil)
	zadd(m, MakeCommandBytes("zadd a 888 a1"), nil)

}
