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
	general.Infof("gpu-pap: gpuPowerPlugin.Run() calling advisor.Run()...")
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
	general.Infof("gpu-pap: NewGPUPowerAwarePlugin called, pluginName=%s", pluginName)
	general.Infof("gpu-pap: socket path config: %s", conf.GPUPowerAwarePluginConfiguration.GPUPowerCappingAdvisorSocketAbsPath)

	emitter := emitterPool.GetDefaultMetricsEmitter().WithTags(metricName)

	// todo: build gpuAdvisor
	specPrefix := conf.PowerAwarePluginConfiguration.AnnotationKeyPrefix
	nodeFetcher := metaServer.NodeFetcher
	specFetcher := spec.NewFetcher(nodeFetcher, specPrefix)

	general.Infof("gpu-pap: creating capper via server.NewCapper...")
	capper, err := server.NewCapper(conf, emitter)
	if err != nil {
		general.Errorf("gpu-pap: server.NewCapper failed: %v", err)
		return nil, errors.Wrap(err, "failed to create advisor")
	}
	general.Infof("gpu-pap: server.NewCapper succeeded")

	gpuAdvisor, err := advisor.New(specFetcher, capper)
	if err != nil {
		general.Errorf("gpu-pap: advisor.New failed: %v", err)
		return nil, errors.Wrap(err, "[gpu-pap] failed to create gpu advisor")
	}
	general.Infof("gpu-pap: advisor.New succeeded")

	plugin, err := newPluginWithAdvisor(pluginName, conf, emitter, gpuAdvisor)
	if err != nil {
		general.Errorf("gpu-pap: newPluginWithAdvisor failed: %v", err)
		return nil, err
	}
	general.Infof("gpu-pap: NewGPUPowerAwarePlugin done, plugin=%v", plugin)
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
