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

package podadmit

import (
	"net"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"
)

type service struct {
	sync.Locker
	started bool

	manager  pluginapi.ResourcePluginServer
	server   *grpc.Server
	sockPath string
}

func (s *service) Name() string {
	return "mb-pod-admit"
}

func (s *service) Start() error {
	s.Lock()
	defer s.Unlock()
	if s.started {
		return nil
	}

	s.started = true
	socket, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return errors.Wrap(err, "failed to start grpc server")
	}

	go func() {
		_ = s.server.Serve(socket)
	}()

	return nil
}

func (s *service) Stop() error {
	s.Lock()
	defer s.Unlock()

	if s.started {
		s.started = false
		s.server.Stop()
	}
	return nil
}

func NewPodAdmitService(manager pluginapi.ResourcePluginServer, sockPath string) (skeleton.GenericPlugin, error) {
	server := grpc.NewServer()
	pluginapi.RegisterResourcePluginServer(server, manager)

	return &service{
		manager:  manager,
		sockPath: sockPath,
	}, nil
}
