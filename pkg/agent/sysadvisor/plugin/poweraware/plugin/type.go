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
	"net"

	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type PodEvictor interface {
	Reset(ctx context.Context)
	Evict(ctx context.Context, pod *v1.Pod) error
}

type NodePowerCapper interface {
	Init() error
	Reset()
	Cap(ctx context.Context, targetWatts, currWatt int)
}

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

func (gs GRPCServer) Run() {
	go func() {
		_ = gs.server.Serve(gs.listener)
		defer func(lis net.Listener) {
			err := lis.Close()
			if err != nil {
				klog.Warningf("socket listener failed to close: %v", err)
			}
		}(gs.listener)
	}()
}

func NewGRPCServer(server *grpc.Server, lis net.Listener) *GRPCServer {
	return &GRPCServer{
		server:   server,
		listener: lis,
	}
}
