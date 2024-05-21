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

package dynamicpolicy

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	metric_consts "github.com/kubewharf/katalyst-core/pkg/consts"
	metrictypes "github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/types"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/external"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

const (
	MBM_Controller = "mbm_controller"
)

type StoppableComponent interface {
	agent.Component
	Stop()
}

type topologySummary struct {
	numPackages  int
	numNUMAs     int
	PackageNUMAs map[int][]int
}

func getTopologySummary(info *machine.KatalystMachineInfo) topologySummary {
	siblings := info.SiblingNumaMap
	numasPerPackage := len(siblings[0]) + 1
	numPackages := info.NumNUMANodes / numasPerPackage
	package_numa := make(map[int][]int)
	for i := 0; i < numPackages; i++ {
		numas := []int{}
		for j := 0; j < numasPerPackage; j++ {
			numas = append(numas, i*numasPerPackage+j)
		}
		package_numa[i] = numas
	}

	return topologySummary{
		numPackages:  numPackages,
		numNUMAs:     info.NumNUMANodes,
		PackageNUMAs: package_numa,
	}
}

// core controller of MBM
// it only interacts with MetricsReader to get current reading of momory bandwidth usage (todo)
// also, it enforces the MBM via ExternalManager (todo)
type mbmController struct {
	emitter                metrics.MetricEmitter
	externalManager        external.ExternalManager
	mbmThresholdPercentage int
	mbmScanInterval        time.Duration
	mbmControllerCancel    context.CancelFunc
	metricsReader          metrictypes.MetricsReader

	// todo: how to determine memory bandwidth spec?
	specMemBandwidth int64

	topologySummary

	// todo: add method to get numa-node x pods mapping
}

func (m *mbmController) Run(ctx context.Context) {
	general.Infof("mbm controller is enabled and in effect")

	ctx, m.mbmControllerCancel = context.WithCancel(ctx)
	go m.run(ctx)
}

func (m *mbmController) run(ctx context.Context) {
	wait.Until(m.process, m.mbmScanInterval, ctx.Done())
}

func (m *mbmController) process() {
	for packageID := 0; packageID < m.numPackages; packageID++ {
		mbPackage, err := m.metricsReader.GetPackageMetric(packageID, metric_consts.MetricMemBandwidthTotalPackage)
		if err != nil {
			// todo: log error
			continue
		}

		if int64(mbPackage.Value) >= m.specMemBandwidth/100*int64(m.mbmThresholdPercentage) {
			// exceeding the threshold; mb throttling kicks in
			m.throttlePackage(packageID)
		}
	}
}

func (m *mbmController) throttlePackage(packageID int) {
	for _, numaID := range m.PackageNUMAs[packageID] {
		mbNUMA, err := m.metricsReader.GetNumaMetric(numaID, metric_consts.MetricMemBandwidthTotalNuma)
		if err != nil {
			// todo: log error
			return
		}

		// todo: locate the combination of numa nodes associated with active (dynamic?) pods
		//       identify the noisy neighbours based on such node combination
		//       then, decide the mb quota for each nodes of the package
		//       lastly, apply quotas to throttle noisy neighbours (via their cpu cores)
		_ = mbNUMA
	}
}

func (m *mbmController) Stop() {
	if m.mbmControllerCancel != nil {
		m.mbmControllerCancel()
		m.mbmControllerCancel = nil
	}
}

func NewMBMController(emitter metrics.MetricEmitter,
	externalManager external.ExternalManager,
	metricsReader metrictypes.MetricsReader,
	threshold int,
	scanInterval time.Duration,
	topology topologySummary) StoppableComponent {
	return &mbmController{
		emitter:                emitter.WithTags(MBM_Controller),
		externalManager:        externalManager,
		metricsReader:          metricsReader,
		mbmThresholdPercentage: threshold,
		mbmScanInterval:        scanInterval,
		topologySummary:        topology,
	}
}

var _ StoppableComponent = &mbmController{}
