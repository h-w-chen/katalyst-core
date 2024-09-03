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

package externalpolicy

import (
	"fmt"
	"path"

	// note: amd_utils is the package from internal repo code.byted.org/tce/amd-utils
	"amd_utils/utils"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/component/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/component/capper/amd"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/component/capper/intel"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/server"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/util/external/power"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	// PowerResourcePluginPolicyNameExternal indicates that the policy of power adjustment is from external component (sys-advisor)
	PowerResourcePluginPolicyNameExternal = "external-power-cap"
)

type externalPolicy struct {
}

func (e externalPolicy) Name() string {
	return PowerResourcePluginPolicyNameExternal
}

func (e externalPolicy) Start() error {
	general.Infof("%s start", PowerResourcePluginPolicyNameExternal)
	return nil
}

func (e externalPolicy) Stop() error {
	general.Infof("%s stop", PowerResourcePluginPolicyNameExternal)
	return nil
}

func (e externalPolicy) ResourceName() string {
	return "power"
}

func NewExternalPolicy(agentCtx *agent.GenericContext, conf *config.Configuration,
	_ interface{}, agentName string,
) (bool, agent.Component, error) {
	// grpc client supports "hot-plug" style service discovery and recovery
	pluginRootFolder := conf.PluginRegistrationDir
	socketPath := path.Join(pluginRootFolder, fmt.Sprintf("%s.sock", server.PowerCapAdvisorPlugin))
	advisorSvcClient := NewRecoverableAdvisorSvcClient(socketPath)

	var powerLimiter power.PowerLimiter
	if utils.IsAMD() {
		powerLimiter = amd.NewPowerLimiter()
	} else {
		powerLimiter = intel.NewRAPLLimiter()
	}
	localPowerCapper := capper.NewLocalPowerCapper(powerLimiter)

	component, err := NewPowerCappingComponent(advisorSvcClient, localPowerCapper)
	if err != nil {
		return false, agent.ComponentStub{}, err
	}

	return true, component, nil
}
