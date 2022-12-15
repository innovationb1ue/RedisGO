package memdb

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func Test_xrange(t *testing.T) {
	type args struct {
		ctx context.Context
		m   *MemDb
		cmd [][]byte
		in3 net.Conn
	}
	ctx := context.Background()
	m := NewMemDb()

	xadd(ctx, m, MakeCommandBytes("xadd a * k1 v1"), nil)

	time.Sleep(time.Second * 1)
	xadd(ctx, m, MakeCommandBytes("xadd a * k2 v2"), nil)
	start := xadd(ctx, m, MakeCommandBytes("xadd a * k3 v3"), nil)
	startTime := start.String()
	xadd(ctx, m, MakeCommandBytes("xadd a * k4 v4"), nil)
	time.Sleep(time.Second * 1)
	xadd(ctx, m, MakeCommandBytes("xadd a * k5 v5"), nil)
	end := xadd(ctx, m, MakeCommandBytes("xadd a * k6 v6"), nil)
	endTime := end.String()
	xadd(ctx, m, MakeCommandBytes("xadd a * k7 v7"), nil)
	xadd(ctx, m, MakeCommandBytes("xadd a * fEnd vEnd"), nil)
	t.Logf("getting entries from %s to %s", startTime, endTime)
	tests := []struct {
		name string
		args args
	}{
		{
			name: "xrange",
			args: args{
				ctx: ctx,
				m:   m,
				cmd: MakeCommandBytes(fmt.Sprintf("xrange a %s %s", startTime, endTime)),
				in3: nil,
			},
		},
	} // TODO: Add test cases.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := make([]string, 0)
			if got := xrange(tt.args.ctx, tt.args.m, tt.args.cmd, tt.args.in3); got == nil {
				t.Errorf("xrange() = %v", got)
			} else {
				res = append(res, got.String())
				t.Log(got.String(), "\n")
			}
			t.Log(res)
		})
	}
}
