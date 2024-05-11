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

package dynamicpolicy

import (
	"testing"
	"time"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"
	katalystbase "github.com/kubewharf/katalyst-core/cmd/base"
	kataagent "github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/config"
	configagent "github.com/kubewharf/katalyst-core/pkg/config/agent"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/global"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/kcc"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

func TestNewDynamicPolicy_MBMController(t *testing.T) {
	t.Parallel()
	type args struct {
		agentCtx  *kataagent.GenericContext
		conf      *config.Configuration
		in2       interface{}
		agentName string
	}
	tests := []struct {
		name              string
		args              args
		want              bool
		wantMBMController bool
		wantErr           bool
	}{
		{
			name: "true mbm_enabled and valid options leads to non-nil mbm controller",
			args: args{
				agentCtx: &kataagent.GenericContext{
					GenericContext: &katalystbase.GenericContext{
						EmitterPool: metricspool.DummyMetricsEmitterPool{},
					},
					MetaServer: &metaserver.MetaServer{
						MetaAgent: &agent.MetaAgent{
							KatalystMachineInfo: &machine.KatalystMachineInfo{
								CPUTopology: &machine.CPUTopology{
									CPUDetails: machine.CPUDetails{0: machine.CPUInfo{}},
								},
							},
						},
						ConfigurationManager: &kcc.DummyConfigurationManager{},
					},
				},
				conf: &config.Configuration{
					GenericConfiguration: &generic.GenericConfiguration{
						QoSConfiguration: &generic.QoSConfiguration{},
					},
					AgentConfiguration: &configagent.AgentConfiguration{
						GenericAgentConfiguration: &configagent.GenericAgentConfiguration{
							BaseConfiguration: &global.BaseConfiguration{
								ReclaimRelativeRootCgroupPath: "dummy-reclaim",
							},
							QRMAdvisorConfiguration: &global.QRMAdvisorConfiguration{
								CPUAdvisorSocketAbsPath:    "dummy-adv",
								CPUPluginSocketAbsPath:     "dummy-plu",
								MemoryAdvisorSocketAbsPath: "dummy-mem-adv",
								MemoryPluginSocketAbsPath:  "dummy-mem-plu",
							},
							GenericQRMPluginConfiguration: &qrm.GenericQRMPluginConfiguration{
								StateFileDirectory:       "/tmp", // set /tmp to tame dependency; no real file op in this test case
								UseKubeletReservedConfig: false,  // to bypass kubelet probings
							},
						},
						StaticAgentConfiguration: &configagent.StaticAgentConfiguration{
							QRMPluginsConfiguration: &qrm.QRMPluginsConfiguration{
								CPUQRMPluginConfig: &qrm.CPUQRMPluginConfig{
									PolicyName: "",
									CPUDynamicPolicyConfig: qrm.CPUDynamicPolicyConfig{
										EnableMBM:              true,
										MBMThresholdPercentage: 75,
										MBMScanInterval:        time.Second * 5,
									},
								},
							},
						},
						DynamicAgentConfiguration: &dynamic.DynamicAgentConfiguration{},
					},
				},
				in2:       nil,
				agentName: "dummy-test",
			},
			want:              true,
			wantMBMController: true, // non-nil mbm controller
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, gotComponent, err := NewDynamicPolicy(tt.args.agentCtx, tt.args.conf, tt.args.in2, tt.args.agentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDynamicPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewDynamicPolicy() got = %v, want %v", got, tt.want)
			}
			if plug, ok := gotComponent.(*kataagent.PluginWrapper); !ok {
				t.Errorf("unpexted type of returned object")
			} else {
				wrap, ok := plug.GenericPlugin.(*skeleton.PluginRegistrationWrapper)
				if !ok {
					t.Errorf("unexpected type in wrapped object")
				} else {
					policy, ok := wrap.GenericPlugin.(*DynamicPolicy)
					if !ok {
						t.Errorf("unexpected type in wrapped object")
					}
					if !tt.wantMBMController { // expecting nil controller
						if policy.mbmController != nil {
							t.Errorf("expected mbm controller nil; got non-nil")
						}
					} else { // expecting non-nil controller
						if policy.mbmController == nil {
							t.Errorf("expected a mbm controller; got nil")
						}
					}
				}
			}
		})
	}
}
