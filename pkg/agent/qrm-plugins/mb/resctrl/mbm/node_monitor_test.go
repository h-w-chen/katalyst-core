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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/raw"
)

type mockMBCalc struct {
	mock.Mock
}

func (m *mockMBCalc) CalcMB(monGroup string, dies []int) map[int]int {
	args := m.Called(monGroup, dies)
	return args.Get(0).(map[int]int)
}

type mockTaskMonitor struct {
	mock.Mock
	TaskMonitor
}

func (m *mockTaskMonitor) AggregateNodeMB(node int) map[int]int {
	args := m.Called(node)
	return args.Get(0).(map[int]int)
}

func Test_nodeMonitor_run(t *testing.T) {
	t.Parallel()

	mockCalc := new(mockMBCalc)
	mockCalc.On("CalcMB", "/sys/fs/resctrl/node_1", []int{2}).Return(map[int]int{2: 222})

	mockTaskMon := new(mockTaskMonitor)
	mockTaskMon.On("AggregateNodeMB", 1).Return(map[int]int{2: 111})

	type fields struct {
		ccdByNode    map[int][]int
		mbCalculator raw.MBCalculator
		taskMonitor  TaskMonitor
		mbCCD        map[int]int
		chStop       chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		mbsWant map[int]int
	}{
		{
			name: "happy path of 1 ccd and once",
			fields: fields{
				ccdByNode:    map[int][]int{1: {2}},
				mbCalculator: mockCalc,
				taskMonitor:  mockTaskMon,
				mbCCD:        map[int]int{},
				chStop:       make(chan struct{}),
			},
			mbsWant: map[int]int{2: 333},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := nodeMonitor{
				ccdByNode:    tt.fields.ccdByNode,
				mbCalculator: tt.fields.mbCalculator,
				taskMonitor:  tt.fields.taskMonitor,
				mbCCD:        tt.fields.mbCCD,
			}
			n.scan()
			assert.Equal(t, tt.mbsWant, tt.fields.mbCCD)
		})
	}
}

func Test_nodeMonitor_GetMB(t *testing.T) {
	t.Parallel()
	type fields struct {
		ccdByNode map[int][]int
		mbCCD     map[int]int
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
			name: "happy path",
			fields: fields{
				ccdByNode: map[int][]int{1: {2, 3}},
				mbCCD:     map[int]int{2: 2020, 3: 3030},
			},
			args: args{
				node: 1,
			},
			want: map[int]int{2: 2020, 3: 3030},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := &nodeMonitor{
				ccdByNode: tt.fields.ccdByNode,
				mbCCD:     tt.fields.mbCCD,
			}
			assert.Equalf(t, tt.want, n.GetMB(tt.args.node), "GetMB(%v)", tt.args.node)
		})
	}
}
