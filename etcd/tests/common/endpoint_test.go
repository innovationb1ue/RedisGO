// Copyright 2022 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"testing"
	"time"

	"go.etcd.io/etcd/tests/v3/framework/testutils"
)

func TestEndpointStatus(t *testing.T) {
	testRunner.BeforeTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clus := testRunner.NewCluster(ctx, t)
	defer clus.Close()
	cc := testutils.MustClient(clus.Client())
	testutils.ExecuteUntil(ctx, t, func() {
		_, err := cc.Status(ctx)
		if err != nil {
			t.Fatalf("get endpoint status error: %v", err)
		}
	})
}

func TestEndpointHashKV(t *testing.T) {
	testRunner.BeforeTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clus := testRunner.NewCluster(ctx, t)
	defer clus.Close()
	cc := testutils.MustClient(clus.Client())
	testutils.ExecuteUntil(ctx, t, func() {
		_, err := cc.HashKV(ctx, 0)
		if err != nil {
			t.Fatalf("get endpoint hashkv error: %v", err)
		}
	})
}

func TestEndpointHealth(t *testing.T) {
	testRunner.BeforeTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clus := testRunner.NewCluster(ctx, t)
	defer clus.Close()
	cc := testutils.MustClient(clus.Client())
	testutils.ExecuteUntil(ctx, t, func() {
		if err := cc.Health(ctx); err != nil {
			t.Fatalf("get endpoint health error: %v", err)
		}
	})
}
