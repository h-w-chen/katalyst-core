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

package qrm

import (
	cliflag "k8s.io/component-base/cli/flag"

	qrmconfig "github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
)

type PowerOptions struct {
	PolicyName string
}

func NewPowerOptions() *PowerOptions {
	return &PowerOptions{
		PolicyName: "external-power-cap",
	}
}

func (p *PowerOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("power_resource_plugin")

	fs.StringVar(&p.PolicyName, "power-resource-plugin-policy",
		p.PolicyName, "The policy power resource plugin should use")
}

func (p *PowerOptions) ApplyTo(conf *qrmconfig.PowerQRMPluginConfig) error {
	conf.PolicyName = p.PolicyName
	return nil
}
