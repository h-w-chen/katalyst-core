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

package poweraware

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor/action/strategy"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor/action/strategy/assess"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper"
	capserver "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper/server"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/evictor"
	evictserver "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/evictor/server"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/reader"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic/crd"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/sysadvisor/poweraware"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const metricName = "poweraware-advisor-plugin"

var _conf *config.Configuration

type powerAwarePlugin struct {
	name    string
	dryRun  bool
	advisor advisor.PowerAwareAdvisor
}

func (p powerAwarePlugin) Name() string {
	return p.name
}

func (p powerAwarePlugin) Init() error {
	general.Infof("pap initialized")

	if err := p.advisor.Init(); err != nil {
		klog.Errorf("pap: failed to initialize power advisor: %v", err)
		return errors.Wrap(err, "failed to initialize power advisor")
	}

	return nil
}

func (p powerAwarePlugin) Run(ctx context.Context) {
	general.Infof("pap running")

	dynamicConfig := _conf.GetDynamicConfiguration()
	powerAwareConfig := dynamicConfig.PowerAwareConfiguration
	general.InfofV(6, "pap: NewPowerAwarePlugin: powerAwareConfig=%v", powerAwareConfig)

	p.advisor.Run(ctx)
	general.Infof("pap ran and finished")
}

func NewPowerAwarePlugin(
	pluginName string,
	conf *config.Configuration,
	_ interface{},
	emitterPool metricspool.MetricsEmitterPool,
	metaServer *metaserver.MetaServer,
	_ metacache.MetaCache,
) (plugin.SysAdvisorPlugin, error) {
	emitter := emitterPool.GetDefaultMetricsEmitter().WithTags(metricName)

	// Add PowerAware dynamic config watcher
	if err := metaServer.ConfigurationManager.AddConfigWatcher(crd.PowerAwareConfigurationGVR); err != nil {
		return nil, err
	}

	// Get dynamic configuration
	// todo: replace this var to other means
	_conf = conf
	dynamicConfig := _conf.GetDynamicConfiguration()
	powerAwareConfig := dynamicConfig.PowerAwareConfiguration
	general.InfofV(6, "pap: NewPowerAwarePlugin: powerAwareConfig=%v", powerAwareConfig)

	// Use dynamic config for runtime-adjustable fields
	dryRun := powerAwareConfig.DryRun
	disablePowerPressureEvict := powerAwareConfig.DisablePowerPressureEvict
	disablePowerCapping := powerAwareConfig.DisablePowerCapping

	// Fall back to static config if dynamic config fields are not set
	if !dryRun && conf.PowerAwarePluginConfiguration.DryRun {
		dryRun = conf.PowerAwarePluginConfiguration.DryRun
	}
	if !disablePowerPressureEvict && conf.DisablePowerPressureEvict {
		disablePowerPressureEvict = conf.DisablePowerPressureEvict
	}
	if !disablePowerCapping && conf.DisablePowerCapping {
		disablePowerCapping = conf.DisablePowerCapping
	}

	// Static config fields (not dynamically configurable)
	annotationKeyPrefix := conf.PowerAwarePluginConfiguration.AnnotationKeyPrefix
	dvfsIndication := conf.PowerAwarePluginConfiguration.DVFSIndication

	var err error
	var podEvictor evictor.PodEvictor
	if disablePowerPressureEvict {
		podEvictor = evictor.NewNoopPodEvictor()
	} else {
		if podEvictor, err = evictserver.NewPowerPressureEvictionServer(conf, emitter); err != nil {
			return nil, errors.Wrap(err, "pap: failed to create power aware plugin")
		}
	}

	var powerCapper capper.PowerCapper
	if disablePowerCapping {
		powerCapper = capper.NewNoopCapper()
	} else {
		if powerCapper, err = capserver.NewPowerCapPlugin(conf, emitter); err != nil {
			return nil, errors.Wrap(err, "pap: failed to create power aware plugin")
		}
	}

	var assessor assess.Assessor
	if dvfsIndication == poweraware.DVFSIndicationPower {
		general.Infof("pap: power as dvfs indication")
		assessor = assess.NewPowerChangeAssessor(0, 0)
	} else {
		general.Infof("pap: cpufreq as dvfs indication")
		assessor = assess.NewCPUFreqChangeAssessor(0, metaServer)
	}

	powerReader := reader.NewMetricStorePowerReader(metaServer)
	percentageEvictor := evictor.NewPowerLoadEvict(conf.QoSConfiguration, emitter, metaServer.PodFetcher, podEvictor)
	powerStrategy := strategy.NewEvictFirstStrategy(emitter, percentageEvictor, metaServer, powerCapper, assessor)
	reconciler := advisor.NewReconciler(dryRun, emitter,
		percentageEvictor, powerCapper, powerStrategy)
	powerAdvisor := advisor.NewAdvisor(dryRun,
		annotationKeyPrefix,
		podEvictor,
		emitter,
		metaServer.NodeFetcher,
		powerReader,
		powerCapper,
		reconciler,
	)

	return newPluginWithAdvisor(pluginName, conf, powerAdvisor)
}

func newPluginWithAdvisor(pluginName string, conf *config.Configuration, advisor advisor.PowerAwareAdvisor) (plugin.SysAdvisorPlugin, error) {
	return &powerAwarePlugin{
		name:    pluginName,
		dryRun:  conf.PowerAwarePluginConfiguration.DryRun,
		advisor: advisor,
	}, nil
}
