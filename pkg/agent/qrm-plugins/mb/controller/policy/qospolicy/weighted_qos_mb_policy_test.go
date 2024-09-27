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

package qospolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

func Test_weightedQoSMBPolicy_GetPlan(t *testing.T) {
	t.Parallel()
	type args struct {
		totalMB   int
		currQoSMB map[task.QoSLevel]*monitor.MBQoSGroup
		isTopTier bool
	}
	tests := []struct {
		name string
		args args
		want *plan.MBAlloc
	}{
		{
			name: "happy path",
			args: args{
				totalMB: 1_500,
				currQoSMB: map[task.QoSLevel]*monitor.MBQoSGroup{
					"foo": {CCDMB: map[int]int{1: 200, 2: 200}},
					"bar": {CCDMB: map[int]int{1: 300, 4: 300}},
				},
				isTopTier: false,
			},
			want: &plan.MBAlloc{
				Plan: map[task.QoSLevel]map[int]int{"foo": {1: 300, 2: 300}, "bar": {1: 450, 4: 450}},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := weightedQoSMBPolicy{}
			assert.Equalf(t, tt.want, w.GetPlan(tt.args.totalMB, tt.args.currQoSMB, tt.args.isTopTier), "GetPlan(%v, %v)", tt.args.totalMB, tt.args.currQoSMB)
		})
	}
}

func Test_weightedQoSMBPolicy_getProportionalPlanToEnlarge(t *testing.T) {
	t.Parallel()
	type fields struct {
		isTopLink bool
	}
	type args struct {
		totalMB   int
		currQoSMB map[task.QoSLevel]*monitor.MBQoSGroup
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *plan.MBAlloc
	}{
		{
			name: "happy path",
			fields: fields{
				isTopLink: false,
			},
			args: args{
				totalMB: 1000,
				currQoSMB: map[task.QoSLevel]*monitor.MBQoSGroup{
					"foo": {CCDMB: map[int]int{1: 100, 2: 200}},
					"bar": {CCDMB: map[int]int{1: 100, 2: 50, 3: 50}},
				},
			},
			want: &plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
				"foo": {1: 200, 2: 400},
				"bar": {1: 200, 2: 100, 3: 100},
			}},
		},
		{
			name: "happy path of inside sub groups",
			fields: fields{
				isTopLink: false,
			},
			args: args{
				totalMB: 1000,
				currQoSMB: map[task.QoSLevel]*monitor.MBQoSGroup{
					"foo": {
						CCDMB: map[int]int{1: 100, 2: 200},
						WeightedCCDMBs: map[int]map[int]int{
							50: {1: 50, 2: 100},
							30: {1: 50, 2: 100},
						},
					},
					"bar": {CCDMB: map[int]int{1: 100, 2: 50, 3: 50}},
				},
			},
			want: &plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
				"foo_50": {1: 100, 2: 200},
				"foo_30": {1: 100, 2: 200},
				"bar":    {1: 200, 2: 100, 3: 100},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &weightedQoSMBPolicy{}
			assert.Equalf(t, tt.want, w.getProportionalPlanToEnlarge(tt.args.totalMB, tt.args.currQoSMB), "getProportionalPlan(%v, %v)", tt.args.totalMB, tt.args.currQoSMB)
		})
	}
}

func Test_weightedQoSMBPolicy_getTopLevelPlan(t *testing.T) {
	t.Parallel()
	type fields struct {
		isTopLink bool
	}
	type args struct {
		totalMB   int
		currQoSMB map[task.QoSLevel]*monitor.MBQoSGroup
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *plan.MBAlloc
	}{
		{
			name: "happy path",
			fields: fields{
				isTopLink: true,
			},
			args: args{
				totalMB:   1000,
				currQoSMB: map[task.QoSLevel]*monitor.MBQoSGroup{"foo": {CCDMB: map[int]int{1: 100, 2: 100}}},
			},
			want: &plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{"foo": {1: 256_000, 2: 256_000}}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &weightedQoSMBPolicy{}
			assert.Equalf(t, tt.want, w.getTopLevelPlan(tt.args.totalMB, tt.args.currQoSMB), "getTopLevelPlan(%v, %v)", tt.args.totalMB, tt.args.currQoSMB)
		})
	}
}

// todo: fix test case failure
func Test_weightedQoSMBPolicy_getProportionalLossPlan(t *testing.T) {
	t.Parallel()
	type args struct {
		loss        int
		qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup
	}
	tests := []struct {
		name string
		args args
		want *plan.MBAlloc
	}{
		{
			name: "weights inside",
			args: args{
				loss: 20_000,
				qosGroupMBs: map[task.QoSLevel]*monitor.MBQoSGroup{
					"foo": {
						CCDMB: map[int]int{2: 5_000, 3: 5_000},
						WeightedCCDMBs: map[int]map[int]int{
							50: {2: 2000, 3: 1000},
							16: {2: 1000, 6: 2000},
							10: {2: 2000, 5: 2000},
						},
					},
					"bar": {CCDMB: map[int]int{2: 5_000, 3: 5_000}},
				},
			},
			want: &plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
				"foo_50": {2: 1615, 3: 807},
				"foo_16": {2: 258, 6: 516},
				"foo_10": {2: 323, 5: 323},
				"bar":    {2: 8088, 3: 8080},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &weightedQoSMBPolicy{}
			assert.Equalf(t, tt.want, w.getProportionalLossPlan(tt.args.loss, tt.args.qosGroupMBs), "getProportionalLossPlan(%v, %v)", tt.args.loss, tt.args.qosGroupMBs)
		})
	}
}
