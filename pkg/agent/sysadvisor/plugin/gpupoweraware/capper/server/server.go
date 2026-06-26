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
	"net"

	"google.golang.org/grpc"

	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type grpcServer struct {
	server   *grpc.Server
	listener net.Listener
}

func (gs *grpcServer) Run() {
	general.Infof("pap-gpu: grpc: grpcServer.Run() starting Serve in goroutine...")
	go func() {
		general.Infof("pap-gpu: grpc: grpcServer.Serve() listening on %v", gs.listener.Addr())
		_ = gs.server.Serve(gs.listener)
		general.Infof("pap-gpu: grpc: grpcServer.Serve() returned")
		defer func(lis net.Listener) {
			err := lis.Close()
			if err != nil {
				general.Warningf("pap-gpu: grpc: gpu power capping server: listener failed to close: %v", err)
			}
		}(gs.listener)
	}()
	general.Infof("pap-gpu: grpc: grpcServer.Run() goroutine launched")
}

func newGRPCServer(server *grpc.Server, lis net.Listener) *grpcServer {
	return &grpcServer{
		server:   server,
		listener: lis,
	}
}
