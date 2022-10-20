package memdb

import (
	"context"
	"testing"
)

func TestZADD(t *testing.T) {
	m := NewMemDb()
	ctx := context.Background()
	zadd(ctx, m, MakeCommandBytes("zadd a 555 hero"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 333 ggbob"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 444 jeff"), nil)

	zadd(ctx, m, MakeCommandBytes("zadd a 666 king"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a -999 loser"), nil)
	// test change a member value
	zadd(ctx, m, MakeCommandBytes("zadd a 999 ggbob"), nil)
	zadd(ctx, m, MakeCommandBytes("zadd a 888 a1"), nil)

}
