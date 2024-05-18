package mbm

import (
	"context"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/global"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbm/sampling"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/types"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
	utilmetric "github.com/kubewharf/katalyst-core/pkg/util/metric"
)

type MBMetricsProvisioner struct {
	machineInfo *machine.KatalystMachineInfo
	metricStore *utilmetric.MetricStore
	emitter     metrics.MetricEmitter

	sampler sampling.MBSampler
}

func (m MBMetricsProvisioner) Run(ctx context.Context) {
	m.sampler.Sample(ctx)
}

func NewMBMetricsProvisioner(_ *global.BaseConfiguration, metricConf *metaserver.MetricConfiguration,
	emitter metrics.MetricEmitter, _ pod.PodFetcher, metricStore *utilmetric.MetricStore) types.MetricsProvisioner {
	m := MBMetricsProvisioner{
		machineInfo: metricConf.MachineInfo,
		metricStore: metricStore,
		emitter:     emitter,
		sampler: metricConf.SamplerFactory(metricConf.MachineInfo,
			sampling.SamplerWriteFunc(metricStore.SetMBMPacketMetrics),
			sampling.SamplerWriteFunc(metricStore.SetMBMNUMAMetrics)),
	}

	m.sampler.Init()

	return &m
}

var _ types.MetricsProvisioner = &MBMetricsProvisioner{}
