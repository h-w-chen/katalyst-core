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

package plugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path"

	"google.golang.org/grpc"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

const (
	PowerCapAdvisorPlugin = "node_power_cap"
)

type powerCapAdvisorPluginServer struct {
	lis        net.Listener
	grpcServer *grpc.Server
}

func (p powerCapAdvisorPluginServer) Init() error {
	return nil
}

func (p powerCapAdvisorPluginServer) Name() string {
	return PowerCapAdvisorPlugin
}

func (p powerCapAdvisorPluginServer) AddContainer(ctx context.Context, metadata *advisorsvc.ContainerMetadata) (*advisorsvc.AddContainerResponse, error) {
	return nil, errors.New("not implemented")
}

func (p powerCapAdvisorPluginServer) RemovePod(ctx context.Context, request *advisorsvc.RemovePodRequest) (*advisorsvc.RemovePodResponse, error) {
	return nil, errors.New("not implemented")
}

func (p powerCapAdvisorPluginServer) ListAndWatch(empty *advisorsvc.Empty, server advisorsvc.AdvisorService_ListAndWatchServer) error {
	//TODO implement me
	panic("implement me")
}

func (p powerCapAdvisorPluginServer) Reset() {
	//TODO implement me
	panic("implement me")
}

func (p powerCapAdvisorPluginServer) Cap(ctx context.Context, targetWatts, currWatt int) {
	//TODO implement me
	panic("implement me")
}

var _ advisorsvc.AdvisorServiceServer = &powerCapAdvisorPluginServer{}

func NewPowerCapAdvisorPluginServer(conf *config.Configuration, emitter metrics.MetricEmitter) (NodePowerCapper, *GRPCServer, error) {
	// todo: emit significant metrics
	powerCapAdvisor := &powerCapAdvisorPluginServer{}

	// todo: extract dir out of conf
	socketPath := path.Join("/tmp/test", fmt.Sprintf("%s.sock", powerCapAdvisor.Name()))
	// todo: delete file if exists
	sock, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%v listen %s failed: %v", powerCapAdvisor.Name(), socketPath, err)
	}

	server := grpc.NewServer()
	advisorsvc.RegisterAdvisorServiceServer(server, powerCapAdvisor)

	return powerCapAdvisor, NewGRPCServer(server, sock), nil
}
