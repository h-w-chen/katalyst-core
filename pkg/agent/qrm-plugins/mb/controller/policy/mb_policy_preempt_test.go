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

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

type mockBoundedPolicy struct {
	mock.Mock
}

func (m *mockBoundedPolicy) GetPlan(upperBoundMB int, currQoSMB map[task.QoSLevel]map[int]int) *plan.MBAlloc {
	args := m.Called(upperBoundMB, currQoSMB)
	return args.Get(0).(*plan.MBAlloc)
}

func Test_preemptPolicy_GetPlan(t *testing.T) {
	t.Parallel()

	boundedPolicy := new(mockBoundedPolicy)
	boundedPolicy.On("GetPlan",
		29_414,
		map[task.QoSLevel]map[int]int{"dedicated_cores": {12: 10_000, 13: 10_000}},
	).Return(&plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
		"dedicated_cores": {12: 25_000, 13: 14_414},
	}})
	boundedPolicy.On("GetPlan",
		24_586,
		map[task.QoSLevel]map[int]int{
			"shared_cores":    {8: 8_000, 9: 8_000},
			"reclaimed_cores": {8: 1_000, 9: 1_000},
			"system_cores":    {9: 3_000},
		},
	).Return(&plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
		"shared_cores":    {8: 8_000, 9: 8_000},
		"reclaimed_cores": {8: 2_000, 9: 2_000},
		"system_cores":    {9: 4_000},
	}})

	boundedPolicy2 := new(mockBoundedPolicy)
	boundedPolicy2.On("GetPlan",
		1_000,
		map[task.QoSLevel]map[int]int{"dedicated_cores": {6: 30_000, 7: 29_000}},
	).Return(&plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
		"dedicated_cores": {6: 30_500, 7: 29_500},
	}})
	boundedPolicy2.On("GetPlan",
		33_000,
		map[task.QoSLevel]map[int]int{"reclaimed_cores": map[int]int{2: 1000, 3: 1000}},
	).Return(&plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
		"reclaimed_cores": map[int]int{2: 17_500, 3: 17_500},
	}})

	type fields struct {
		boundedMBPolicy BoundedMBPolicy
	}
	type args struct {
		domain    *mbdomain.MBDomain
		currQoSMB map[task.QoSLevel]map[int]int
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
				boundedMBPolicy: boundedPolicy,
			},
			args: args{
				domain: &mbdomain.MBDomain{
					NodeCCDs:      map[int][]int{4: {8, 9}, 5: {10, 11}, 6: {12, 13}},
					PreemptyNodes: sets.Int{5: sets.Empty{}},
				},
				currQoSMB: map[task.QoSLevel]map[int]int{
					"dedicated_cores": {12: 10_000, 13: 10_000},
					"shared_cores":    {8: 8_000, 9: 8_000},
					"reclaimed_cores": {8: 1_000, 9: 1_000},
					"system_cores":    {9: 3_000},
				},
			},
			want: &plan.MBAlloc{
				Plan: map[consts.QoSLevel]map[int]int{
					"dedicated_cores": map[int]int{10: 12_500, 11: 12_500, 12: 25_000, 13: 14_414},
					"reclaimed_cores": map[int]int{8: 2000, 9: 2000},
					"shared_cores":    map[int]int{8: 8000, 9: 8000},
					"system_cores":    map[int]int{9: 4000}}},
		},
		{
			name: "dedicated already close to max",
			fields: fields{
				boundedMBPolicy: boundedPolicy2,
			},
			args: args{
				domain: &mbdomain.MBDomain{
					NodeCCDs:      map[int][]int{1: {2, 3}, 2: {4, 5}, 3: {6, 7}},
					PreemptyNodes: sets.Int{2: sets.Empty{}},
				},
				currQoSMB: map[task.QoSLevel]map[int]int{
					"dedicated_cores": map[int]int{6: 30_000, 7: 29_000},
					"reclaimed_cores": map[int]int{2: 1000, 3: 1000},
				},
			},
			want: &plan.MBAlloc{Plan: map[task.QoSLevel]map[int]int{
				"dedicated_cores": map[int]int{4: 12_500, 5: 12_500, 6: 30_500, 7: 29_500},
				"reclaimed_cores": map[int]int{2: 17_500, 3: 17_500},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := preemptPolicy{
				boundedMBPolicy: tt.fields.boundedMBPolicy,
			}
			assert.Equalf(t, tt.want, p.GetPlan(tt.args.domain, tt.args.currQoSMB), "GetPlan(%v, %v)", tt.args.domain, tt.args.currQoSMB)
		})
	}
}
