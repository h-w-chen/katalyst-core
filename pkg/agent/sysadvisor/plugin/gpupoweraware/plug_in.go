package gpupoweraware

import (
	"context"
	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/advisor"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper/server"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
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
	general.Infof("pap-gpu: gpuPowerPlugin.Run() calling advisor.Run()...")
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
	general.Infof("pap-gpu: NewGPUPowerAwarePlugin called, pluginName=%s", pluginName)
	general.Infof("pap-gpu: socket path config: %s", conf.GPUPowerAwarePluginConfiguration.GPUPowerCappingAdvisorSocketAbsPath)

	emitter := emitterPool.GetDefaultMetricsEmitter().WithTags(metricName)

	specPrefix := conf.PowerAwarePluginConfiguration.AnnotationKeyPrefix
	nodeFetcher := metaServer.NodeFetcher
	specFetcher := spec.NewFetcher(nodeFetcher, specPrefix)

	general.Infof("pap-gpu: capper service: creating capper via server.NewCapper...")
	capper, err := server.NewCapper(conf, emitter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create advisor")
	}
	general.Infof("pap-gpu: capper service: capper server created")

	gpuAdvisor, err := advisor.New(specFetcher, capper)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gpu advisor")
	}
	general.Infof("pap-gpu: advisor: advisor created")

	plugin, err := newPluginWithAdvisor(pluginName, conf, emitter, gpuAdvisor)
	if err != nil {
		general.Errorf("pap-gpu: plugin: newPluginWithAdvisor failed: %v", err)
		return nil, err
	}
	general.Infof("pap-gpu: plugin: NewGPUPowerAwarePlugin done, plugin=%v", plugin)
	return plugin, nil
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
