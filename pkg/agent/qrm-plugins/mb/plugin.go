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
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mba"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type plugin struct {
	dieTopology *machine.DieTopology
}

func (c plugin) Name() string {
	return "mb_plugin"
}

func (c plugin) Start() error {
	general.InfofV(6, "mbm: plugin component starting ....")
	general.InfofV(6, "mbm: numa-CCD-cpu topology: \n%s", c.dieTopology)

	nodesByPackage := make(map[int]int)
	for packageID, numas := range c.dieTopology.NUMAsInPackage {
		for _, numa := range numas {
			nodesByPackage[numa] = packageID
		}
	}

	mbaManager, err := mba.New(nodesByPackage, c.dieTopology.DiesInNuma, c.dieTopology.CPUsInDie)
	if err != nil {
		return errors.Wrap(err, "failed to create mba manager")
	}

	if err := mbaManager.CreateResctrlLayout(afero.NewOsFs()); err != nil {
		return errors.Wrap(err, "failed to create resctrl mba layout")
	}

	panic("impl me")
}

func (c plugin) Stop() error {
	panic("impl me")
}

func NewComponent(agentCtx *agent.GenericContext, conf *config.Configuration,
	_ interface{}, agentName string,
) (bool, agent.Component, error) {
	mbPlugin := &plugin{
		dieTopology: agentCtx.DieTopology,
	}

	return true, &agent.PluginWrapper{GenericPlugin: mbPlugin}, nil
}
