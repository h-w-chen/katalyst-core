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

package mb

import (
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/allocator"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/state"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

// todo: not to use global vars
var (
	TaskManager task.Manager
	MBDomainManager *mbdomain.MBDomainManager
	MBController *controller.Controller
)

type plugin struct {
	qosConfig              *generic.QoSConfiguration
	pluginRegistrationDirs []string

	dieTopology        *machine.DieTopology
	incubationInterval time.Duration

	mbController    *controller.Controller
	podAdmitService skeleton.GenericPlugin
}

func (p *plugin) Name() string {
	return "qrm_mb_plugin"
}

func (p *plugin) Start() error {
	general.InfofV(6, "mbm: plugin component starting ....")
	general.InfofV(6, "mbm: mb incubation interval %v", p.incubationInterval)
	general.InfofV(6, "mbm: numa-CCD-cpu topology: \n%s", p.dieTopology)

	// todo: NOT to return error (to crash explicitly); consider downgrade service
	if !p.dieTopology.FakeNUMAEnabled {
		return errors.New("mbm: not virtual numa; no need to dynamically manage the memory bandwidth")
	}

	//domainManager := mbdomain.NewMBDomainManager(p.dieTopology, p.incubationInterval)
	//
	//var err error
	//
	//dataKeeper, err := state.NewMBRawDataKeeper()
	//if err != nil {
	//	return errors.Wrap(err, "failed to create raw data state keeper")
	//}
	//
	//taskManager, err := task.New(p.dieTopology.DiesInNuma, p.dieTopology.CPUsInDie, dataKeeper, domainManager)
	//if err != nil {
	//	return errors.Wrap(err, "failed to create task manager")
	//}
	//
	//podMBMonitor, err := monitor.NewDefaultMBMonitor(p.dieTopology.CPUsInDie, dataKeeper, taskManager, domainManager)
	//if err != nil {
	//	return errors.Wrap(err, "mbm: failed to create default mb monitor")
	//}
	//
	//mbPlanAllocator, err := createMBPlanAllocator()
	//if err != nil {
	//	return errors.Wrap(err, "mbm: failed to create mb plan allocator")
	//}
	//
	//domainPolicy, err := policy.NewDefaultDomainMBPolicy(p.incubationInterval)
	//if err != nil {
	//	return errors.Wrap(err, "mbm: failed to create domain manager")
	//}
	//
	//p.mbController, err = controller.New(podMBMonitor, mbPlanAllocator, domainManager, domainPolicy)
	//if err != nil {
	//	return errors.Wrap(err, "mbm: failed to create mb controller")
	//}

	//// todo: disable the competing (shadowing) memory plugin; instead to cooperate into the built-in dynamic memory plugin:
	//p.podAdmitService, err = podadmit.NewPodAdmitService(p.qosConfig, domainManager, p.mbController, taskManager, p.pluginRegistrationDirs)
	//if err != nil {
	//	return errors.Wrap(err, "mbm: failed to create pod admit service")
	//}
	//if err := p.podAdmitService.Start(); err != nil {
	//	return errors.Wrap(err, "mbm: failed to start pod admit service")
	//}
	
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				general.Errorf("mbm: background run exited, due to error: %v", err)
			}
		}()

		p.mbController.Run()
	}()

	return nil
}

func createMBPlanAllocator() (allocator.PlanAllocator, error) {
	schemataUpdater, err := resctrl.NewSchemataUpdater()
	if err != nil {
		return nil, err
	}

	ctrlGroupMBSetter, err := resctrl.NewCtrlGroupSetter(schemataUpdater)
	if err != nil {
		return nil, err
	}

	return allocator.NewPlanAllocator(ctrlGroupMBSetter)
}

func (p *plugin) Stop() error {
	// todo: not sure why not being called on ctrl-C ???
	general.Infof("mbm: mb plugin is stopping...")
	if p.podAdmitService != nil {
		if err := p.podAdmitService.Stop(); err != nil {
			general.Errorf("mbm: plugin failed to stop pod admitter service: %v", err)
		}
	}

	return p.mbController.Stop()
}

func NewComponent(agentCtx *agent.GenericContext, conf *config.Configuration,
	_ interface{}, agentName string,
) (bool, agent.Component, error) {
	plugin := &plugin{
		qosConfig:              conf.QoSConfiguration,
		pluginRegistrationDirs: conf.QRMPluginSocketDirs,
		dieTopology:            agentCtx.DieTopology,
		incubationInterval:     conf.IncubationInterval,
	}

	domainManager := mbdomain.NewMBDomainManager(plugin.dieTopology, plugin.incubationInterval)

	var err error

	dataKeeper, err := state.NewMBRawDataKeeper()
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to create raw data state keeper")
	}

	taskManager, err := task.New(plugin.dieTopology.DiesInNuma, plugin.dieTopology.CPUsInDie, dataKeeper, domainManager)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to create task manager")
	}

	podMBMonitor, err := monitor.NewDefaultMBMonitor(plugin.dieTopology.CPUsInDie, dataKeeper, taskManager, domainManager)
	if err != nil {
		return false, nil, errors.Wrap(err, "mbm: failed to create default mb monitor")
	}

	mbPlanAllocator, err := createMBPlanAllocator()
	if err != nil {
		return false, nil, errors.Wrap(err, "mbm: failed to create mb plan allocator")
	}

	domainPolicy, err := policy.NewDefaultDomainMBPolicy(plugin.incubationInterval)
	if err != nil {
		return false, nil, errors.Wrap(err, "mbm: failed to create domain manager")
	}

	plugin.mbController, err = controller.New(podMBMonitor, mbPlanAllocator, domainManager, domainPolicy)
	if err != nil {
		return false, nil, errors.Wrap(err, "mbm: failed to create mb controller")
	}

	MBController = plugin.mbController
	TaskManager = taskManager
	MBDomainManager = domainManager

	return true, &agent.PluginWrapper{GenericPlugin: plugin}, nil
}
