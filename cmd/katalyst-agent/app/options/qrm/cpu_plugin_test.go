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
