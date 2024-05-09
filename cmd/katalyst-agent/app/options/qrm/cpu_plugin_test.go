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
	"testing"
	"time"

	qrmconfig "github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
)

func TestCPUOptions_ApplyTo(t *testing.T) {
	type fields struct {
		PolicyName              string
		ReservedCPUCores        int
		SkipCPUStateCorruption  bool
		CPUDynamicPolicyOptions CPUDynamicPolicyOptions
		CPUNativePolicyOptions  CPUNativePolicyOptions
	}
	type args struct {
		conf *qrmconfig.CPUQRMPluginConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path of mbm options",
			fields: fields{
				PolicyName:             "dummy-policy",
				ReservedCPUCores:       0,
				SkipCPUStateCorruption: false,
				CPUDynamicPolicyOptions: CPUDynamicPolicyOptions{
					EnableCPUAdvisor:              false,
					EnableCPUPressureEviction:     false,
					LoadPressureEvictionSkipPools: nil,
					EnableSyncingCPUIdle:          false,
					EnableCPUIdle:                 false,
					EnableMBM:                     true,
					MBMThresholdPercentage:        88,
					MBMScanInterval:               time.Second * 2,
				},
				CPUNativePolicyOptions: CPUNativePolicyOptions{},
			},
			args: args{
				conf: &qrmconfig.CPUQRMPluginConfig{
					PolicyName:             "dummy-policy",
					ReservedCPUCores:       0,
					SkipCPUStateCorruption: false,
					CPUDynamicPolicyConfig: qrmconfig.CPUDynamicPolicyConfig{},
					CPUNativePolicyConfig:  qrmconfig.CPUNativePolicyConfig{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &CPUOptions{
				PolicyName:              tt.fields.PolicyName,
				ReservedCPUCores:        tt.fields.ReservedCPUCores,
				SkipCPUStateCorruption:  tt.fields.SkipCPUStateCorruption,
				CPUDynamicPolicyOptions: tt.fields.CPUDynamicPolicyOptions,
				CPUNativePolicyOptions:  tt.fields.CPUNativePolicyOptions,
			}
			if err := o.ApplyTo(tt.args.conf); (err != nil) != tt.wantErr {
				t.Errorf("ApplyTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
