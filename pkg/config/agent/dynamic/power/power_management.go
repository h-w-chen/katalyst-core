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

package power

import (
	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic/crd"
)

type PowerManagementConfiguration struct {
	DisablePowerAdvisor bool
	PowerReductionRatio int

	DisablePowerCapping bool
}

func NewPowerManagementConfiguration() *PowerManagementConfiguration {
	// set the safe static-default values initially
	// overridden once by the startup args when app boosts
	// regularly updated by kcc-cnc mechanism later on
	return &PowerManagementConfiguration{
		DisablePowerAdvisor: true,
		PowerReductionRatio: 10,
		DisablePowerCapping: true,
	}
}

func (c *PowerManagementConfiguration) ApplyConfiguration(conf *crd.DynamicConfigCRD) {
	if conf == nil {
		return
	}

	pmc := conf.PowerManagementConfiguration
	if pmc == nil {
		return
	}

	if disablePowerAdvisor := pmc.Spec.Config.DisablePowerAdvisor; disablePowerAdvisor != nil {
		c.DisablePowerAdvisor = *disablePowerAdvisor
	}

	if powerReductionRatio := pmc.Spec.Config.PowerReductionRatio; powerReductionRatio != nil {
		c.PowerReductionRatio = int(*powerReductionRatio)
	}

	if disablePowerCapping := pmc.Spec.Config.DisablePowerCapping; disablePowerCapping != nil {
		c.DisablePowerCapping = *disablePowerCapping
	}
}
