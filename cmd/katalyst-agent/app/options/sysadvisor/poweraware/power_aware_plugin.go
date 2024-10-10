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
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/sysadvisor/poweraware"
)

type PowerAwarePluginOptions struct {
	Disabled                  bool
	DryRun                    bool
	DisablePowerCapping       bool
	DisablePowerPressureEvict bool
}

func (p *PowerAwarePluginOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("power-aware-plugin")
	fs.BoolVar(&p.Disabled, "power-aware-Disabled", p.Disabled, "flag for disabling power aware advisor")
	fs.BoolVar(&p.DryRun, "power-aware-dryrun", p.DryRun, "flag for dry run power aware advisor")
	fs.BoolVar(&p.DisablePowerPressureEvict, "power-pressure-evict-Disabled", p.DisablePowerPressureEvict, "flag for power aware plugin disabling power pressure eviction")
	fs.BoolVar(&p.DisablePowerCapping, "power-capping-Disabled", p.DisablePowerCapping, "flag for power aware plugin disabling power capping")
}

func (p *PowerAwarePluginOptions) ApplyTo(o *poweraware.PowerAwarePluginConfiguration) error {
	o.DryRun = p.DryRun
	o.Disabled = p.Disabled
	o.DisablePowerPressureEvict = p.DisablePowerPressureEvict
	o.DisablePowerCapping = p.DisablePowerCapping
	return nil
}

// NewPowerAwarePluginOptions creates a new Options with a default config.
func NewPowerAwarePluginOptions() *PowerAwarePluginOptions {
	return &PowerAwarePluginOptions{}
}
