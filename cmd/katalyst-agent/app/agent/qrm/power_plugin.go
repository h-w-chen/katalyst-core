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
	"fmt"
	"k8s.io/klog/v2"
	"sync"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/config"
)

const (
	QRMPluginNamePower = "qrm_power_plugin"
)

// powerPolicyInitializers is used to store the initializing function for power resource-plugin policies
var powerPolicyInitializers sync.Map

// RegisterPowerPolicyInitializer is used to register user-defined power resource plugin init functions
func RegisterPowerPolicyInitializer(name string, initFunc agent.InitFunc) {
	klog.V(6).Infof("pap: reg policy %s", name)
	powerPolicyInitializers.Store(name, initFunc)
}

// getPowerPolicyInitializers returns those policies with initialized functions
func getPowerPolicyInitializers() map[string]agent.InitFunc {
	klog.V(6).Infof("pap: get all registered polices...")
	agents := make(map[string]agent.InitFunc)
	powerPolicyInitializers.Range(func(key, value interface{}) bool {
		klog.V(6).Infof("pap: policy %v", key)
		agents[key.(string)] = value.(agent.InitFunc)
		return true
	})
	return agents
}

// InitQRMPowerPlugins initializes power plugin with the named policy as specified by the configuration
func InitQRMPowerPlugins(agentCtx *agent.GenericContext, conf *config.Configuration, extraConf interface{}, agentName string) (bool, agent.Component, error) {
	policyName := conf.PowerQRMPluginConfig.PolicyName

	initializers := getPowerPolicyInitializers()
	initFunc, ok := initializers[policyName]
	if !ok {
		return false, agent.ComponentStub{}, fmt.Errorf("invalid policy name %v for power resource plugin", policyName)
	}

	return initFunc(agentCtx, conf, extraConf, agentName)
}
