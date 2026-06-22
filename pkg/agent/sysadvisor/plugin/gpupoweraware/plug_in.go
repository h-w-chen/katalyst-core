package gpupoweraware

import (
	"context"
	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/advisor"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
)

const metricName = "gpu-poweraware-advisor-plugin"

type gpuPowerPlugin struct {
	name   string
	dryRun bool

	emitter metrics.MetricEmitter
	advisor advisor.Advisor
}

func (g gpuPowerPlugin) Name() string {
	return g.name
}

func (g gpuPowerPlugin) Init() error {
	return g.advisor.Init()
}

func (g gpuPowerPlugin) Run(ctx context.Context) {
	g.advisor.Run(ctx)
}

func NewGPUPowerAwarePlugin(
	pluginName string,
	conf *config.Configuration,
	_ interface{},
	emitterPool metricspool.MetricsEmitterPool,
	metaServer *metaserver.MetaServer,
	_ metacache.MetaCache,
) (plugin.SysAdvisorPlugin, error) {
	emitter := emitterPool.GetDefaultMetricsEmitter().WithTags(metricName)

	// todo: build gpuAdvisor
	gpuAdvisor, err := advisor.New()
	if err != nil {
		return nil, errors.Wrap(err, "[gpu-pap] failed to create gpu advisor")
	}

	return newPluginWithAdvisor(pluginName, conf, emitter, gpuAdvisor)
}

func newPluginWithAdvisor(pluginName string, conf *config.Configuration, emitter metrics.MetricEmitter, advisor advisor.Advisor,
) (plugin.SysAdvisorPlugin, error) {
	return &gpuPowerPlugin{
		name:    pluginName,
		dryRun:  conf.PowerAwarePluginConfiguration.DryRun,
		emitter: emitter,
		advisor: advisor,
	}, nil
}
