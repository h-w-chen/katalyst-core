/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package capper

import (
	"context"
	"fmt"
	evictionv1apha1 "github.com/kubewharf/katalyst-api/pkg/protocol/evictionplugin/v1alpha1"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
)

// TestRemotePowerCapper is an integration test case; should NOT be run in unit test suite
func TestRemotePowerCapper(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	var lis *bufconn.Listener
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials())
	if err != nil {
		t.Fatalf("failed to set up buffconn for test: %v", err)
	}
	defer conn.Close()

	client := advisorsvc.NewAdvisorServiceClient(conn)
	lwClient, err := client.ListAndWatch(ctx, &advisorsvc.Empty{})
	if err != nil {
		t.Fatalf("test client failed to connect to test server: %v", err)
	}
	lwResp, err := lwClient.Recv()
	if err != nil {
		t.Fatalf("test client failed to call Recv: %v", err)
	}
	_ = lwResp
}
