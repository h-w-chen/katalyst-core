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

package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	capper2 "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

// setupTestServer creates an in-memory gRPC server with the powerCapService registered,
// starts it, and returns a connected client, the service, and a cleanup function.
func setupTestServer(t *testing.T) (advisorsvc.AdvisorServiceClient, *powerCapService, func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	svc := newGPUPowerCapService(&metrics.DummyMetrics{})
	svc.started = true

	baseServer := grpc.NewServer()
	advisorsvc.RegisterAdvisorServiceServer(baseServer, svc)

	go func() {
		_ = baseServer.Serve(lis)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "failed to dial bufconn")

	client := advisorsvc.NewAdvisorServiceClient(conn)

	cleanup := func() {
		conn.Close()
		baseServer.Stop()
	}

	return client, svc, cleanup
}

// doGetAdviceThenPush starts GetAdvice in a goroutine, waits for it to enter the
// long-poll select (capturing the dataUpdatedCh), then calls push to unblock it.
// Returns the response or any error.
func doGetAdviceThenPush(
	t *testing.T,
	client advisorsvc.AdvisorServiceClient,
	svc *powerCapService,
	push func(),
) (*advisorsvc.GetAdviceResponse, error) {
	t.Helper()

	type result struct {
		resp *advisorsvc.GetAdviceResponse
		err  error
	}
	done := make(chan result, 1)

	go func() {
		resp, err := client.GetAdvice(context.Background(), &advisorsvc.GetAdviceRequest{})
		done <- result{resp, err}
	}()

	// wait for GetAdvice to enter the handler and capture dataUpdatedCh
	time.Sleep(100 * time.Millisecond)

	push()

	select {
	case r := <-done:
		return r.resp, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("GetAdvice did not return in time")
		return nil, nil
	}
}

// Test_CapWithLevel_GetAdvice_E2E verifies the end-to-end flow:
// server-side CapWithLevel → client-side GetAdvice returns the instruction.
func Test_CapWithLevel_GetAdvice_E2E(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := doGetAdviceThenPush(t, client, svc, func() {
		svc.CapWithLevel(context.Background(), capper2.LevelDecode, 80, 100)
	})
	require.NoError(t, err)

	require.NotNil(t, resp)
	require.Len(t, resp.ExtraEntries, 1)

	calcRes := resp.ExtraEntries[0].CalculationResult
	assert.Equal(t, string(capper.OpCap), calcRes.Values["op-code"])
	assert.Equal(t, "100", calcRes.Values["current-value"])
	assert.Equal(t, "80", calcRes.Values["target-value"])
	assert.Equal(t, string(capper2.LevelDecode), calcRes.Values["op-level"])
}

// Test_Reset_GetAdvice_E2E verifies that Reset produces a reset instruction
// that the gRPC client receives via GetAdvice.
func Test_Reset_GetAdvice_E2E(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := doGetAdviceThenPush(t, client, svc, func() {
		svc.Reset()
	})
	require.NoError(t, err)

	require.NotNil(t, resp)
	require.Len(t, resp.ExtraEntries, 1)

	calcRes := resp.ExtraEntries[0].CalculationResult
	assert.Equal(t, string(capper.OpReset), calcRes.Values["op-code"])
}

// Test_GetAdvice_EmptyResponse_WhenNoInstruction verifies that GetAdvice
// returns an empty response (after long-poll timeout) when no instruction is set.
func Test_GetAdvice_EmptyResponse_WhenNoInstruction(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	svc.longPoller.timeout = 50 * time.Millisecond

	resp, err := client.GetAdvice(context.Background(), &advisorsvc.GetAdviceRequest{})
	require.NoError(t, err)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.ExtraEntries)
}

// Test_GetAdvice_ApplyPreviousReset verifies the metadata-based
// "apply previous reset" flow: when the client sends x-apply-previous-reset:yes
// and the server has a pending reset, it returns the reset immediately.
func Test_GetAdvice_ApplyPreviousReset(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	// set a pending reset instruction
	svc.requestReset()

	// client sends metadata requesting to apply previous reset
	md := metadata.Pairs(MetadataApplyPreviousReset, MetadataValueYes)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.GetAdvice(ctx, &advisorsvc.GetAdviceRequest{})
	require.NoError(t, err)

	require.NotNil(t, resp)
	require.Len(t, resp.ExtraEntries, 1)

	calcRes := resp.ExtraEntries[0].CalculationResult
	assert.Equal(t, string(capper.OpReset), calcRes.Values["op-code"])
}

// Test_GetAdvice_ApplyPreviousReset_NoResetPending verifies that
// when x-apply-previous-reset:yes is sent but no reset is pending,
// the request falls through to the normal long-poll path.
func Test_GetAdvice_ApplyPreviousReset_NoResetPending(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	svc.longPoller.timeout = 50 * time.Millisecond

	md := metadata.Pairs(MetadataApplyPreviousReset, MetadataValueYes)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.GetAdvice(ctx, &advisorsvc.GetAdviceRequest{})
	require.NoError(t, err)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.ExtraEntries)
}

// Test_CapWithLevel_MultipleLevels verifies that different GPU workload levels
// are correctly propagated through the gRPC layer.
func Test_CapWithLevel_MultipleLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level capper2.Level
	}{
		{name: "decode", level: capper2.LevelDecode},
		{name: "all", level: capper2.LevelAll},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, svc, cleanup := setupTestServer(t)
			defer cleanup()

			resp, err := doGetAdviceThenPush(t, client, svc, func() {
				svc.CapWithLevel(context.Background(), tt.level, 70, 110)
			})
			require.NoError(t, err)

			require.Len(t, resp.ExtraEntries, 1)
			calcRes := resp.ExtraEntries[0].CalculationResult
			assert.Equal(t, string(tt.level), calcRes.Values["op-level"])
		})
	}
}

// Test_IsCapperReady verifies that IsCapperReady returns true only when
// there is an active GetAdvice call in flight.
func Test_IsCapperReady(t *testing.T) {
	t.Parallel()

	client, svc, cleanup := setupTestServer(t)
	defer cleanup()

	assert.False(t, svc.IsCapperReady())

	svc.longPoller.timeout = 5 * time.Second
	errCh := make(chan error, 1)
	go func() {
		_, err := client.GetAdvice(context.Background(), &advisorsvc.GetAdviceRequest{})
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	assert.True(t, svc.IsCapperReady())

	svc.CapWithLevel(context.Background(), capper2.LevelAll, 50, 100)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("GetAdvice did not return in time")
	}
}

// Test_CapWithLevel_WhenStopped verifies that CapWithLevel is a no-op
// when the service is not started.
func Test_CapWithLevel_WhenStopped(t *testing.T) {
	t.Parallel()

	svc := newGPUPowerCapService(&metrics.DummyMetrics{})
	// not started

	svc.CapWithLevel(context.Background(), capper2.LevelAll, 50, 100)
	assert.Nil(t, svc.getCapInstruction())
}
