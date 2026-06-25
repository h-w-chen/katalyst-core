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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	capper2 "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper"
	powermetric "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/metric"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// ServiceName also is the unix socket name of the server is listening on
	ServiceName = "gpu_power_cap"

	pollingTimeout = time.Second * 60

	// MetadataApplyPreviousReset is the custom information from gpu power cap client to gpu power cap advisor that
	// allows client to catch up with the reset it just missed by specifying x-apply-previous-reset:yes.
	MetadataApplyPreviousReset = "x-apply-previous-reset"

	MetadataValueYes = "yes"
)

type longPoller struct {
	timeout       time.Duration
	dataUpdatedCh chan struct{}
}

func (l *longPoller) setDataUpdated() {
	close(l.dataUpdatedCh)
	l.dataUpdatedCh = make(chan struct{})
}

type powerCapService struct {
	sync.RWMutex
	started          bool
	capInstruction   *capper2.Instruction
	activeGetAdvices int32
	emitter          metrics.MetricEmitter
	grpcServer       *grpcServer

	longPoller *longPoller
}

func (g *powerCapService) Cap(ctx context.Context, targetWatts, currWatt int) {
	panic("not to use")
}

func (g *powerCapService) hadReset() bool {
	g.RLock()
	defer g.RUnlock()

	return g.capInstruction != nil && g.capInstruction.OpCode == capper.OpReset
}

func (g *powerCapService) getCapInstruction() *capper2.Instruction {
	g.RLock()
	defer g.RUnlock()
	return g.capInstruction
}

func (g *powerCapService) IsCapperReady() bool {
	return atomic.LoadInt32(&g.activeGetAdvices) > 0
}

func (g *powerCapService) Stop() error {
	g.Lock()
	defer g.Unlock()
	if !g.started {
		return nil
	}

	g.started = false
	g.grpcServer.server.Stop()
	return nil
}

func (g *powerCapService) Start() error {
	g.Lock()
	defer g.Unlock()

	if g.started {
		general.Infof("pap-gpu: svc: powerCapService.Start() already started, skipping")
		return nil
	}

	general.Infof("pap-gpu: svc: powerCapService.Start() called, starting grpc server...")
	g.started = true
	g.grpcServer.Run()
	general.Infof("pap-gpu: svc: grpcServer.Run() called")

	// reset gpu power capping to prevent accumulative effect
	g.requestReset()
	general.Infof("pap-gpu: svc: powerCapService.Start() done")

	return nil
}

func (g *powerCapService) Init() error {
	return nil
}

func (g *powerCapService) Name() string {
	return ServiceName
}

func (g *powerCapService) AddContainer(ctx context.Context, metadata *advisorsvc.ContainerMetadata) (*advisorsvc.AddContainerResponse, error) {
	return nil, errors.New("not implemented")
}

func (g *powerCapService) RemovePod(ctx context.Context, request *advisorsvc.RemovePodRequest) (*advisorsvc.RemovePodResponse, error) {
	return nil, errors.New("not implemented")
}

func (g *powerCapService) GetAdvice(ctx context.Context, request *advisorsvc.GetAdviceRequest) (*advisorsvc.GetAdviceResponse, error) {
	return g.getAdviceWithClientReadySignal(ctx, request, nil)
}

func (g *powerCapService) getAdviceWithClientReadySignal(ctx context.Context, request *advisorsvc.GetAdviceRequest, clientReadyCh chan<- struct{}) (*advisorsvc.GetAdviceResponse, error) {
	atomic.AddInt32(&g.activeGetAdvices, 1)
	defer atomic.AddInt32(&g.activeGetAdvices, -1)

	powermetric.EmitGetAdviceCalled(g.emitter)
	general.InfofV(6, "gpu-pap: svc: get advice request: %v", general.ToString(request))

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		toApplyPreviousReset := md.Get(MetadataApplyPreviousReset)
		if len(toApplyPreviousReset) > 0 && toApplyPreviousReset[0] == MetadataValueYes {
			if g.hadReset() {
				return g.getAdvice(ctx, request)
			}
		}
	}

	// when no ready instruction ready yet, fall back to long polling
	serverCtx, cancel := context.WithTimeout(ctx, g.longPoller.timeout)
	defer cancel()

	g.RLock()
	dataUpdatedCh := g.longPoller.dataUpdatedCh
	g.RUnlock()

	if clientReadyCh != nil {
		clientReadyCh <- struct{}{}
	}

	select {
	case <-dataUpdatedCh:
		return g.getAdvice(ctx, request)
	case <-serverCtx.Done():
		return &advisorsvc.GetAdviceResponse{}, nil
	case <-ctx.Done():
		general.Warningf("pap-gpu: svc: get advice aborted by either client disconnection or timeout")
		return nil, errors.New("client disconnected or canceled")
	}
}

func (g *powerCapService) getAdvice(_ context.Context, _ *advisorsvc.GetAdviceRequest) (*advisorsvc.GetAdviceResponse, error) {
	capInst := g.getCapInstruction()
	if capInst == nil {
		return &advisorsvc.GetAdviceResponse{}, nil
	}

	general.InfofV(6, "gpu-gpu: svc: cap service reply %v", *capInst)
	resp := capInst.ToAdviceResponse()
	return resp, nil
}

func (g *powerCapService) deliverPendingReset(ch chan<- struct{}) {
	g.RLock()
	defer g.RUnlock()

	if g.capInstruction != nil && g.capInstruction.OpCode == capper.OpReset {
		ch <- struct{}{}
	}
}

func (g *powerCapService) ListAndWatch(empty *advisorsvc.Empty, server advisorsvc.AdvisorService_ListAndWatchServer) error {
	return fmt.Errorf("LW not supported")
}

func (g *powerCapService) Reset() {
	g.Lock()
	defer g.Unlock()

	if !g.started {
		general.Warningf("pap-gpu: svc: gpu power capping service is unavailable")
		g.emitErrorCode(powermetric.ErrorCodePowerCapperUnavailable)
		return
	}

	g.emitGPUPowerCapReset()
	g.requestReset()
}

func (g *powerCapService) requestReset() {
	g.capInstruction = capper2.PowerCapReset
	g.longPoller.setDataUpdated()
}

// Cap sends a GPU power capping instruction with level "all" (default).
// It sends a GPU power capping instruction with the specified level.
// level can be "infer", "train", or "all".
func (g *powerCapService) CapWithLevel(ctx context.Context, oplevel capper2.Level, targetWatts, currWatt int) {
	capInst, err := capper2.NewInstruction(targetWatts, currWatt, oplevel)
	if err != nil {
		general.Warningf("pap-gpu: svc: invalid gpu cap request: %v", err)
		g.emitErrorCode(powermetric.ErrorCodeOther)
		return
	}

	g.Lock()
	defer g.Unlock()

	if !g.started {
		general.Warningf("pap-gpu: svc: gpu power capping service is unavailable")
		g.emitErrorCode(powermetric.ErrorCodePowerCapperUnavailable)
		return
	}

	g.emitGPUPowerCapInstruction(capInst)
	g.capInstruction = capInst
	g.longPoller.setDataUpdated()
}

func newGPUPowerCapService(emitter metrics.MetricEmitter) *powerCapService {
	return &powerCapService{
		emitter: emitter,
		longPoller: &longPoller{
			timeout:       pollingTimeout,
			dataUpdatedCh: make(chan struct{}),
		},
	}
}

func newGPUPowerCapServiceSuite(conf *config.Configuration, emitter metrics.MetricEmitter) (*powerCapService, *grpcServer, error) {
	general.Infof("pap-gpu: svc: newGPUPowerCapServiceSuite called")
	gpuPowerCapSvc := newGPUPowerCapService(emitter)

	socketPath := conf.GPUPowerAwarePluginConfiguration.GPUPowerCappingAdvisorSocketAbsPath
	general.Infof("pap-gpu: svc: socketPath=%s", socketPath)

	general.Infof("pap-gpu: svc: removing old socket file if exists...")
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		general.Errorf("pap-gpu: svc: failed to remove old socket file: %v", err)
		return nil, nil, errors.Wrap(err, "failed to clean up the residue file")
	}
	general.Infof("pap-gpu: svc: old socket file cleaned up")

	socketDir := filepath.Dir(socketPath)
	general.Infof("pap-gpu: svc: creating socket dir: %s", socketDir)
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		general.Errorf("pap-gpu: svc: failed to create socket dir: %v", err)
		return nil, nil, errors.Wrap(err, "failed to create folders to unix sock file")
	}
	general.Infof("pap-gpu: svc: socket dir created/verified")

	general.Infof("pap-gpu: svc: listening on unix socket: %s", socketPath)
	sock, err := net.Listen("unix", socketPath)
	if err != nil {
		general.Errorf("pap-gpu: svc: net.Listen failed: %v", err)
		return nil, nil, fmt.Errorf("%v listen %s failed: %v", gpuPowerCapSvc.Name(), socketPath, err)
	}
	general.Infof("pap-gpu: svc: net.Listen succeeded, socket created at %s", socketPath)

	server := grpc.NewServer()
	advisorsvc.RegisterAdvisorServiceServer(server, gpuPowerCapSvc)
	general.Infof("pap-gpu: avc: AdvisorServiceServer registered")

	return gpuPowerCapSvc, newGRPCServer(server, sock), nil
}

// NewCapper creates a GPU power capping plugin.
// It implements PowerCapper with GPU-specific CapWithLevel method.
func NewCapper(conf *config.Configuration, emitter metrics.MetricEmitter) (capper2.PowerCapper, error) {
	general.Infof("pap-gpu: NewCapper called")
	gpuPowerCapAdvisor, grpcServer, err := newGPUPowerCapServiceSuite(conf, emitter)
	if err != nil {
		general.Errorf("pap-gpu: newGPUPowerCapServiceSuite failed: %v", err)
		return nil, errors.Wrap(err, "failed to create gpu power capping server")
	}

	gpuPowerCapAdvisor.grpcServer = grpcServer
	general.Infof("pap-gpu: NewCapper done")
	return gpuPowerCapAdvisor, nil
}
