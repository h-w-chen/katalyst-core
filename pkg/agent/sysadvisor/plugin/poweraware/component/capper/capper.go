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

	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/server"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	"github.com/kubewharf/katalyst-core/pkg/util/external/power"
)

type PowerCapper = server.NodePowerCapper

type powerCapper struct {
	limiter power.PowerLimiter
}

func (p powerCapper) Init() error {
	return p.limiter.Init()
}

func (p powerCapper) Reset() {
	p.limiter.Reset()
}

func (p powerCapper) Cap(_ context.Context, targetWatts, currWatt int) {
	if err := p.limiter.SetLimitOnBasis(targetWatts, currWatt); err != nil {
		klog.Errorf("pap: failed to power cap, current watt %d, target watt %d", currWatt, targetWatts)
	}
}

// todo: remove all local power capping code to qrm plugin (in katalyst-adapter repo)
func NewLocalPowerCapper(limiter power.PowerLimiter) PowerCapper {
	return &powerCapper{limiter: limiter}
}

func NewRemotePowerCapper(conf *config.Configuration, emitter metrics.MetricEmitter) PowerCapper {
	powerCapAdvisor, grpcServer, err := server.NewPowerCapAdvisorPluginServer(conf, emitter)
	if err != nil {
		klog.Errorf("pap: failed to create power cap advisor service: %v", err)
		return nil
	}

	grpcServer.Run()

	return powerCapAdvisor
}
