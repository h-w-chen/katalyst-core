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

package controller

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/numapackage"
	"reflect"
	"testing"
)

func Test_calcSoftAllocs(t *testing.T) {
	t.Parallel()

	mU0 := new(mockMBUnit)
	mU0.On("GetNUMANodes").Return([]int{0})
	mU0.On("GetTaskType").Return(numapackage.TaskTypeLowPriority)
	mU1 := new(mockMBUnit)
	mU1.On("GetNUMANodes").Return([]int{1})
	mU1.On("GetTaskType").Return(numapackage.TaskTypeSOCKET)
	mU2 := new(mockMBUnit)
	mU2.On("GetNUMANodes").Return([]int{2})
	mU2.On("GetTaskType").Return(numapackage.TaskTypeSOCKET)
	mU3 := new(mockMBUnit)
	mU3.On("GetNUMANodes").Return([]int{3})
	mU3.On("GetTaskType").Return(numapackage.TaskTypeLowPriority)

	mMonitor := new(mockMonitor)
	mMonitor.On("GetMB", 0).Return(map[int]int{0: 6_000, 1: 6_000})
	mMonitor.On("GetMB", 1).Return(map[int]int{2: 25_000, 3: 15_000})
	mMonitor.On("GetMB", 2).Return(map[int]int{4: 25_000, 5: 25_000})
	mMonitor.On("GetMB", 3).Return(map[int]int{6: 5_000, 7: 5_000})

	type args struct {
		units        []numapackage.MBUnit
		mb           int
		mbHiReserved int
		mbMonitor    monitor.Monitor
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "happy path of 4 numa nodes (2 hi, 2 lo), 8 CCDs",
			args: args{
				units:        []numapackage.MBUnit{mU0, mU1, mU2, mU3},
				mb:           116_000,
				mbHiReserved: 6_000,
				mbMonitor:    mMonitor,
			},
			want:    []int{11785, 9821, 60000, 60000},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calcSoftAllocs(tt.args.units, tt.args.mb, tt.args.mbHiReserved, tt.args.mbMonitor)
			if (err != nil) != tt.wantErr {
				t.Errorf("calcSoftAllocs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i := 0; i < 4; i++ {
				if !reflect.DeepEqual(got[i].mbUpperBound, tt.want[i]) {
					t.Errorf("calcSoftAllocs() for lo node %d got = %v, want %v", i, got[i].mbUpperBound, tt.want[i])
				}
			}
		})
	}
}
