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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor/stat"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
)

type mockQoSPolicy struct {
	mock.Mock
	QoSMBPolicy
}

func (m *mockQoSPolicy) GetPlan(upperBoundMB int, groups, gloablUsage map[qosgroup.QoSGroup]*stat.MBQoSGroup, isTopTier bool) *plan.MBAlloc {
	args := m.Called(upperBoundMB, groups, gloablUsage, isTopTier)
	return args.Get(0).(*plan.MBAlloc)
}

func Test_priorityChainedMBPolicy_GetPlan(t *testing.T) {
	t.Parallel()

	currPolicy := new(mockQoSPolicy)
	currPolicy.On("GetPlan", 120_000, map[qosgroup.QoSGroup]*stat.MBQoSGroup{
		"dedicated": {CCDMB: map[int]*stat.MBData{2: {TotalMB: 15_000}, 3: {TotalMB: 15_000}, 4: {TotalMB: 20_000}, 5: {TotalMB: 20_000}}},
		"shared-50": {CCDMB: map[int]*stat.MBData{0: {TotalMB: 7_000}, 1: {TotalMB: 10_000}, 7: {TotalMB: 5_000}}},
		"system":    {CCDMB: map[int]*stat.MBData{0: {TotalMB: 3_000}, 7: {TotalMB: 5_000}}},
	}, map[qosgroup.QoSGroup]*stat.MBQoSGroup(nil), true).Return(&plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{
		"dedicated": {2: 25_000, 3: 25_000, 4: 25_000, 5: 25_000},
		"shared-50": {0: 25_000, 1: 25_000, 7: 25_000},
		"system":    {0: 25_000, 7: 25000},
	}})

	nextPolicy := new(mockQoSPolicy)
	nextPolicy.On("GetPlan", 20_000, map[qosgroup.QoSGroup]*stat.MBQoSGroup{
		"shared-30": {CCDMB: map[int]*stat.MBData{6: {TotalMB: 7_000}}},
	}, map[qosgroup.QoSGroup]*stat.MBQoSGroup(nil), false).Return(&plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{
		"shared-30": {6: 20_000},
	}})

	type fields struct {
		topTiers map[qosgroup.QoSGroup]struct{}
		tier     QoSMBPolicy
		next     QoSMBPolicy
	}
	type args struct {
		totalMB   int
		groups    map[qosgroup.QoSGroup]*stat.MBQoSGroup
		isTopTier bool
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
				topTiers: map[qosgroup.QoSGroup]struct{}{
					"dedicated": {},
					"shared-50": {},
					"system":    {},
				},
				tier: currPolicy,
				next: nextPolicy,
			},
			args: args{
				totalMB: 120_000,
				groups: map[qosgroup.QoSGroup]*stat.MBQoSGroup{
					"dedicated": {CCDMB: map[int]*stat.MBData{2: {TotalMB: 15_000}, 3: {TotalMB: 15_000}, 4: {TotalMB: 20_000}, 5: {TotalMB: 20_000}}},
					"shared-50": {CCDMB: map[int]*stat.MBData{0: {TotalMB: 7_000}, 1: {TotalMB: 10_000}, 7: {TotalMB: 5_000}}},
					"shared-30": {CCDMB: map[int]*stat.MBData{6: {TotalMB: 7_000}}},
					"system":    {CCDMB: map[int]*stat.MBData{0: {TotalMB: 3_000}, 7: {TotalMB: 5_000}}},
				},
				isTopTier: true,
			},
			want: &plan.MBAlloc{
				Plan: map[qosgroup.QoSGroup]map[int]int{
					"dedicated": {2: 25_000, 3: 25_000, 4: 25_000, 5: 25_000},
					"shared-50": {0: 25_000, 1: 25_000, 7: 25_000},
					"shared-30": {6: 20_000},
					"system":    {0: 25_000, 7: 25_000},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := chainedQosPolicy{
				currQoSLevels: tt.fields.topTiers,
				current:       tt.fields.tier,
				next:          tt.fields.next,
			}
			assert.Equalf(t, tt.want, p.GetPlan(tt.args.totalMB, tt.args.groups, nil, tt.args.isTopTier), "GetPlan(%v, %v)", tt.args.totalMB, tt.args.groups)
		})
	}
}
