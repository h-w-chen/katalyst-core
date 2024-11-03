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
	"fmt"
	"net"
	"os"
	"path"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-api/pkg/plugins/registration"
	watcherapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
	qrmpluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const mbResourcePluginSocketBaseName = "qrm_mb_plugin.sock"

type service struct {
	sync.Mutex
	started bool

	admitter pluginapi.ResourcePluginServer
	server   *grpc.Server

	// there are various places that kubelet may probe at for historical reason
	sockPaths []string
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

	_ = cleanupFiles(s.sockPaths)

	s.started = true
	for _, sockPath := range s.sockPaths {
		socket, err := net.Listen("unix", sockPath)
		if err != nil {
			return errors.Wrap(err, "failed to start grpc server")
		}

		go func() {
			_ = s.server.Serve(socket)
		}()
	}

	return nil
}

func cleanupFiles(files []string) error {
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s failed with error: %v", file, err)
		}
	}

	return nil
}

func (s *service) Stop() error {
	s.Lock()
	defer s.Unlock()

	if s.started {
		general.Infof("mbm: pod admitter service is stopping...")
		s.started = false
		s.server.Stop()

		if err := cleanupFiles(s.sockPaths); err != nil {
			return errors.Wrap(err, "failed to stop pod admitter service")
		}
	}

	return nil
}

// todo: use skeleton.NewRegistrationPluginWrapper to create service in line with others
func NewPodAdmitService(qosConfig *generic.QoSConfiguration,
	domainManager *mbdomain.MBDomainManager, mbController *controller.Controller, taskManager task.Manager,
	sockDirs []string,
) (skeleton.GenericPlugin, error) {
	admissionManager := &admitter{
		UnimplementedResourcePluginServer: pluginapi.UnimplementedResourcePluginServer{},
		qosConfig:                         qosConfig,
		domainManager:                     domainManager,
		mbController:                      mbController,
		taskManager:                       taskManager,
	}

	server := grpc.NewServer()
	pluginapi.RegisterResourcePluginServer(server, admissionManager)

	// setup registration service for kubelet to probe
	regSvc := registration.NewRegistrationHandler(watcherapi.ResourcePlugin, "memory", []string{qrmpluginapi.Version})
	watcherapi.RegisterRegistrationServer(server, regSvc)

	sockPaths := make([]string, len(sockDirs))
	for i, dir := range sockDirs {
		sockPaths[i] = path.Join(dir, mbResourcePluginSocketBaseName)
	}

	return &service{
		admitter:  admissionManager,
		sockPaths: sockPaths,
		server:    server,
	}, nil
}
