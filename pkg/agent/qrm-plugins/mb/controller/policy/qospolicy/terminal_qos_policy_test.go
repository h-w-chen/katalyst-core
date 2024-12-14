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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/strategy"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
)

func Test_getTopMostPlan(t *testing.T) {
	t.Parallel()
	type args struct {
		totalMB     int
		mbQoSGroups map[qosgroup.QoSGroup]*monitor.MBQoSGroup
	}
	tests := []struct {
		name string
		args args
		want *plan.MBAlloc
	}{
		{
			name: "happy path",
			args: args{
				totalMB: 1_000,
				mbQoSGroups: map[qosgroup.QoSGroup]*monitor.MBQoSGroup{
					"dedicated": {CCDs: sets.Int{4: sets.Empty{}, 5: sets.Empty{}}},
					"shared-50": {CCDs: sets.Int{0: sets.Empty{}, 1: sets.Empty{}}},
					"system":    {CCDs: sets.Int{1: sets.Empty{}}},
				},
			},
			want: &plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{
				"dedicated": {4: 35_000, 5: 35_000},
				"shared-50": {0: 35_000, 1: 35_000},
				"system":    {1: 35_000},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := terminalQoSPolicy{
				ccdMBMin: 8_000,
			}
			assert.Equalf(t, tt.want, p.getTopMostPlan(tt.args.totalMB, tt.args.mbQoSGroups), "getTopMostPlan(%v, %v)", tt.args.totalMB, tt.args.mbQoSGroups)
		})
	}
}

type mockLeafPlanner struct {
	mock.Mock
	strategy.LowPrioPlanner
}

func (m *mockLeafPlanner) GetPlan(capacity int, mbQoSGroups map[qosgroup.QoSGroup]*monitor.MBQoSGroup) *plan.MBAlloc {
	args := m.Called(capacity, mbQoSGroups)
	return args.Get(0).(*plan.MBAlloc)
}

func Test_getLeafPlan(t *testing.T) {
	t.Parallel()

	underUsed := map[qosgroup.QoSGroup]*monitor.MBQoSGroup{
		"shared-30": {
			CCDMB: map[int]*monitor.MBData{6: {
				TotalMB: 4_000,
			}},
		},
	}
	mockEasePlanner := new(mockLeafPlanner)
	mockEasePlanner.On("GetPlan", 15_000, underUsed).Return(&plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{"shared-30": {6: 6_000}}})

	overUsed := map[qosgroup.QoSGroup]*monitor.MBQoSGroup{
		"shared-30": {
			CCDs: sets.Int{6: sets.Empty{}},
			CCDMB: map[int]*monitor.MBData{6: {
				TotalMB: 11_000,
			}},
		},
	}
	mockThrottlePlanner := new(mockLeafPlanner)
	mockThrottlePlanner.On("GetPlan", 15_000, overUsed).Return(&plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{"shared-30": {6: 3_456}}})

	type fields struct {
		throttlePlanner, easePlanner strategy.LowPrioPlanner
	}
	type args struct {
		totalMB     int
		mbQoSGroups map[qosgroup.QoSGroup]*monitor.MBQoSGroup
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *plan.MBAlloc
	}{
		{
			name: "under pressure leads to easing",
			fields: fields{
				throttlePlanner: nil,
				easePlanner:     mockEasePlanner,
			},
			args: args{
				totalMB:     15_000,
				mbQoSGroups: underUsed,
			},
			want: &plan.MBAlloc{
				Plan: map[qosgroup.QoSGroup]map[int]int{"shared-30": map[int]int{6: 6_000}}},
		},
		{
			name: "almost saturation path lead to throttling",
			fields: fields{
				throttlePlanner: mockThrottlePlanner,
				easePlanner:     nil,
			},
			args: args{
				totalMB:     15_000,
				mbQoSGroups: overUsed,
			},
			want: &plan.MBAlloc{
				Plan: map[qosgroup.QoSGroup]map[int]int{"shared-30": map[int]int{6: 3_456}}},
		},
		{
			name: "in between neither throttle nor ease",
			fields: fields{
				throttlePlanner: nil,
				easePlanner:     nil,
			},
			args: args{
				totalMB: 15_000,
				mbQoSGroups: map[qosgroup.QoSGroup]*monitor.MBQoSGroup{
					"shared-30": {
						CCDs: sets.Int{6: sets.Empty{}},
						CCDMB: map[int]*monitor.MBData{6: {
							TotalMB: 8_000,
						}},
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := terminalQoSPolicy{
				ccdMBMin:        8_000,
				throttlePlanner: tt.fields.throttlePlanner,
				easePlanner:     tt.fields.easePlanner,
			}
			assert.Equalf(t, tt.want, p.getLeafPlan(tt.args.totalMB, tt.args.mbQoSGroups), "getLeafPlan(%v, %v)", tt.args.totalMB, tt.args.mbQoSGroups)

			if tt.fields.throttlePlanner != nil {
				mock.AssertExpectationsForObjects(t, tt.fields.throttlePlanner)
			}
			if tt.fields.easePlanner != nil {
				mock.AssertExpectationsForObjects(t, tt.fields.easePlanner)
			}
		})
	}
}
