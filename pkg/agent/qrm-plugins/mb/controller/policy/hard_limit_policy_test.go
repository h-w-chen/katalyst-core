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

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/apppool"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

func Test_calcPreemptAllocs(t *testing.T) {
	t.Parallel()

	mU0 := new(mockMBUnit)
	mU0.On("GetNUMANodes").Return([]int{0})
	mU0.On("GetTaskType").Return(apppool.TaskTypeLowPriority)
	mU0.On("GetLifeCyclePhase").Return(apppool.UnitPhaseRunning)
	mU1 := new(mockMBUnit)
	mU1.On("GetNUMANodes").Return([]int{1})
	mU1.On("GetTaskType").Return(apppool.TaskTypeSOCKET)
	mU1.On("GetLifeCyclePhase").Return(apppool.UnitPhaseAdmitted)
	mU2 := new(mockMBUnit)
	mU2.On("GetNUMANodes").Return([]int{2})
	mU2.On("GetTaskType").Return(apppool.TaskTypeSOCKET)
	mU2.On("GetLifeCyclePhase").Return(apppool.UnitPhaseRunning)
	mU3 := new(mockMBUnit)
	mU3.On("GetNUMANodes").Return([]int{3})
	mU3.On("GetTaskType").Return(apppool.TaskTypeLowPriority)
	mU3.On("GetLifeCyclePhase").Return(apppool.UnitPhaseRunning)

	mMonitor := new(mockMonitor)
	mMonitor.On("GetMB", 0).Return(map[int]int{0: 6_000, 1: 6_000})
	mMonitor.On("GetMB", 2).Return(map[int]int{4: 25_000, 5: 25_000})
	mMonitor.On("GetMB", 3).Return(map[int]int{6: 5_000, 7: 5_000})

	type args struct {
		units     []apppool.AppPool
		mbMonitor monitor.Monitor
	}
	tests := []struct {
		name    string
		args    args
		want    []MBAlloc
		wantErr bool
	}{
		{
			name: "happy path of 4 numa nodes, 1 in hi admit, 1 hi run, 2 lo run",
			args: args{
				units:     []apppool.AppPool{mU3, mU2, mU1, mU0},
				mbMonitor: mMonitor,
			},
			want: []MBAlloc{
				{AppPool: mU3, MBUpperBound: 10416},
				{AppPool: mU2, MBUpperBound: 58083},
				{AppPool: mU0, MBUpperBound: 12500},
				{AppPool: mU1, MBUpperBound: 35000},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calcPreemptAllocs(tt.args.units, 116000, 6000, tt.args.mbMonitor)
			if (err != nil) != tt.wantErr {
				t.Errorf("calcPreemptAllocs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calcPreemptAllocs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
