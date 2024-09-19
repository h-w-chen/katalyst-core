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

package mbm

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/raw"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/task"
)

type mockTaskMgr struct {
	task.TaskManager
	mock.Mock
}

func (m *mockTaskMgr) GetTasks(node int) []int {
	args := m.Called(node)
	return args.Get(0).([]int)
}

func Test_taskMonitor_scanFs(t1 *testing.T) {
	t1.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/sys/fs/resctrl/mon_groups/node_2_pid_2515624", 0755)

	mockCalc := new(mockMBCalc)
	mockCalc.On("CalcMB", "/sys/fs/resctrl/mon_groups/node_2_pid_2515624", []int{4, 5}).Return(map[int]int{4: 222, 5: 888})

	type fields struct {
		taskMBs      map[int]map[int]int
		taskManager  task.TaskManager
		mbCalculator raw.MBCalculator
		nodeCCDs     map[int][]int
		chStop       chan struct{}
	}
	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "happy path",
			fields: fields{
				taskMBs:      make(map[int]map[int]int),
				taskManager:  nil,
				mbCalculator: mockCalc,
				nodeCCDs:     map[int][]int{2: {4, 5}},
				chStop:       nil,
			},
			args: args{
				fs: fs,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := &taskMonitor{
				taskMBs:      tt.fields.taskMBs,
				taskManager:  tt.fields.taskManager,
				mbCalculator: tt.fields.mbCalculator,
				nodeCCDs:     tt.fields.nodeCCDs,
				chStop:       tt.fields.chStop,
			}
			t.scanFs(tt.args.fs)

			t1.Logf("%v", tt.fields.taskMBs)
			assert.Equal(t1, map[int]map[int]int{2515624: {4: 222, 5: 888}}, tt.fields.taskMBs)
		})
	}
}

func Test_taskMonitor_AggregateNodeMB(t1 *testing.T) {
	t1.Parallel()

	dummyTaskMgr := new(mockTaskMgr)
	dummyTaskMgr.On("GetTasks", 2).Return([]int{123, 124})

	type fields struct {
		taskMBs     map[int]map[int]int
		taskManager task.TaskManager
		nodeCCDs    map[int][]int
	}
	type args struct {
		node int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[int]int
	}{
		{
			name: "happy path of 2 tasks of 1 node",
			fields: fields{
				taskMBs: map[int]map[int]int{
					123: {4: 10, 5: 10},
					124: {4: 11, 5: 12},
				},
				taskManager: dummyTaskMgr,
				nodeCCDs:    map[int][]int{2: {4, 5}},
			},
			args: args{
				node: 2,
			},
			want: map[int]int{4: 21, 5: 22},
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := &taskMonitor{
				taskMBs:     tt.fields.taskMBs,
				taskManager: tt.fields.taskManager,
				nodeCCDs:    tt.fields.nodeCCDs,
			}
			assert.Equalf(t1, tt.want, t.AggregateNodeMB(tt.args.node), "AggregateNodeMB(%v)", tt.args.node)
		})
	}
}

func Test_taskMonitor_GetTaskMB(t1 *testing.T) {
	t1.Parallel()
	type fields struct {
		taskMBs map[int]map[int]int
	}
	type args struct {
		pid int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[int]int
	}{
		{
			name: "happy path",
			fields: fields{
				taskMBs: map[int]map[int]int{2: {4: 44, 5: 55}},
			},
			args: args{
				pid: 2,
			},
			want: map[int]int{4: 44, 5: 55},
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := &taskMonitor{
				taskMBs: tt.fields.taskMBs,
			}
			assert.Equalf(t1, tt.want, t.GetTaskMB(tt.args.pid), "GetTaskMB(%v)", tt.args.pid)
		})
	}
}
