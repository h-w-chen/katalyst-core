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
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/numapackage"
)

type mockMBUnit struct {
	mock.Mock
	numapackage.MBUnit
}

func (m *mockMBUnit) GetNUMANodes() []int {
	args := m.Called()
	return args.Get(0).([]int)
}

func (m *mockMBUnit) GetTaskType() numapackage.TaskType {
	args := m.Called()
	return numapackage.TaskType(args.String(0))
}

func (m *mockMBUnit) GetLifeCyclePhase() numapackage.UnitPhase {
	args := m.Called()
	return numapackage.UnitPhase(args.String(0))
}

type mockMonitor struct {
	mock.Mock
	monitor.Monitor
}

func (m *mockMonitor) GetMB(node int) map[int]int {
	args := m.Called(node)
	return args.Get(0).(map[int]int)
}

func Test_getGroupMBUsages(t *testing.T) {
	t.Parallel()

	mMBUnit := new(mockMBUnit)
	mMBUnit.On("GetNUMANodes").Return([]int{4, 5})
	mMBUnit.On("GetTaskType").Return(numapackage.TaskTypeSOCKET)

	mMonitor := new(mockMonitor)
	mMonitor.On("GetMB", 4).Return(map[int]int{8: 3, 9: 12})
	mMonitor.On("GetMB", 5).Return(map[int]int{10: 4, 11: 15})

	type args struct {
		units     []numapackage.MBUnit
		mbMonitor monitor.Monitor
	}
	tests := []struct {
		name     string
		args     args
		wantHiMB int
		wantLoMB int
	}{
		{
			name: "happy path of 2 units",
			args: args{
				units:     []numapackage.MBUnit{mMBUnit},
				mbMonitor: mMonitor,
			},
			wantHiMB: 34,
			wantLoMB: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotHiMB, gotLoMB := getHiLoGroupMBs(tt.args.units, tt.args.mbMonitor)
			if gotHiMB != tt.wantHiMB {
				t.Errorf("getHiLoGroupMBs() gotHiMB = %v, want %v", gotHiMB, tt.wantHiMB)
			}
			if gotLoMB != tt.wantLoMB {
				t.Errorf("getHiLoGroupMBs() gotLoMB = %v, want %v", gotLoMB, tt.wantLoMB)
			}
		})
	}
}

func Test_ccdDistributeMB(t *testing.T) {
	t.Parallel()
	type args struct {
		total int
		mbCCD map[int]int
	}
	tests := []struct {
		name string
		args args
		want map[int]int
	}{
		{
			name: "happy path of equal ccds",
			args: args{
				total: 100,
				mbCCD: map[int]int{6: 20, 7: 20},
			},
			want: map[int]int{6: 50, 7: 50},
		},
		{
			name: "proportional distribution",
			args: args{
				total: 90,
				mbCCD: map[int]int{6: 80, 7: 40},
			},
			want: map[int]int{6: 60, 7: 30},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CcdDistributeMB(tt.args.total, tt.args.mbCCD); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CcdDistributeMB() = %v, want %v", got, tt.want)
			}
		})
	}
}
