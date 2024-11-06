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
	qrmconfig "github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
	"time"

	cliflag "k8s.io/component-base/cli/flag"
)

const defaultIncubationInterval = time.Second * 30

type MBOptions struct {
	IncubationInterval time.Duration
}

func NewMBOptions() *MBOptions {
	return &MBOptions{IncubationInterval: defaultIncubationInterval}
}

func (m *MBOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("io_resource_plugin")
	fs.DurationVar(&m.IncubationInterval, "mb-incubation-interval", m.IncubationInterval,
		"time to protect socket pod before it is fully exercise memory bandwidth")
}

func (m *MBOptions) ApplyTo(conf *qrmconfig.MBQRMPluginConfig) error {
	conf.IncubationInterval = m.IncubationInterval

	return nil
}
