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

package mbw

import (
	"context"

	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/global"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/mbw/monitor"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbw/sampling"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/types"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	utilmetric "github.com/kubewharf/katalyst-core/pkg/util/metric"
)

var mbwMonitor *monitor.MBMonitor

type MBWMetricsProvisioner struct {
	metricStore *utilmetric.MetricStore
	emitter     metrics.MetricEmitter

	// todo: make incompatible provisioner not run at all - this may require code change elsewhere in framework
	// shouldNotRun is flag that mbw metrics provisioner is NOT compatible on local machine
	shouldNotRun bool
	initialized  bool
	sampler      sampling.Sampler
}

func (m *MBWMetricsProvisioner) Run(ctx context.Context) {
	if m.shouldNotRun {
		klog.Infof("mbw: noop")
		return
	}

	if !m.initialized {
		m.initialized = true
		if err := m.sampler.Startup(ctx); err != nil {
			klog.Errorf("mbm: failed to initialize and start mbw monitor: %v", err)
			m.shouldNotRun = true
			return
		}
	}

	m.sampler.Sample(ctx)
}

func NewMBWMetricsProvisioner(config *global.BaseConfiguration, metricConf *metaserver.MetricConfiguration,
	emitter metrics.MetricEmitter, _ pod.PodFetcher, metricStore *utilmetric.MetricStore,
) types.MetricsProvisioner {
	klog.Infof("mbw: creating mbw metric provisioner")

	m := MBWMetricsProvisioner{
		metricStore: metricStore,
		emitter:     emitter,
	}

	// todo: remove temp code
	m.shouldNotRun = true

	//var err error
	//mbwMonitor, err = monitor.NewMonitor(config.MachineInfoConfiguration)
	//if err != nil {
	//	klog.Errorf("mbw: create provisioner failed")
	//	m.shouldNotRun = true
	//} else {
	//	m.sampler = sampling.New(mbwMonitor, metricStore, emitter)
	//}

	return &m
}

// GetMBWMonitor assumes its invocation AFTER NewMBWMetricsProvisioner (if any when enabled)
func GetMBWMonitor() *monitor.MBMonitor {
	// ok to return nil mbw monitor, which indicates that mbw metrics not enabled
	return mbwMonitor
}
