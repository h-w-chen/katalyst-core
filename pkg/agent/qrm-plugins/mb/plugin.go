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
	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type plugin struct {
	dieTopology *machine.DieTopology

	mbController *controller.Controller
}

func (c *plugin) Name() string {
	return "mb_plugin"
}

func (c *plugin) Start() error {
	general.InfofV(6, "mbm: plugin component starting ....")
	general.InfofV(6, "mbm: numa-CCD-cpu topology: \n%s", c.dieTopology)

	var err error
	podMBMonitor, err := monitor.New()
	if err != nil {
		return err
	}
	c.mbController, err = controller.New(podMBMonitor)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				general.Errorf("mbm: background run exited, due to error: %v", err)
			}
		}()

		c.mbController.Run()
	}()

	return nil
}

func (c *plugin) Stop() error {
	return c.mbController.Stop()
}

func NewComponent(agentCtx *agent.GenericContext, conf *config.Configuration,
	_ interface{}, agentName string,
) (bool, agent.Component, error) {
	mbController := &plugin{
		dieTopology: agentCtx.DieTopology,
	}

	return true, &agent.PluginWrapper{GenericPlugin: mbController}, nil
}
