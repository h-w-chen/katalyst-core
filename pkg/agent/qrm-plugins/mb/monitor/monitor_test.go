package monitor

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

type mockTaskManager struct {
	mock.Mock
}

func (m *mockTaskManager) GetTasks() []*task.Task {
	args := m.Called()
	return args.Get(0).([]*task.Task)
}

type mockTaskMBReader struct {
	mock.Mock
}

func (m *mockTaskMBReader) ReadMB(taskID string) map[int]int {
	args := m.Called(taskID)
	return args.Get(0).(map[int]int)
}

func Test_mbMonitor_GetQoSMBs(t1 *testing.T) {
	t1.Parallel()

	taskManager := new(mockTaskManager)
	taskManager.On("GetTasks").Return([]*task.Task{{
		PodUID:   "123-45-6789",
		QoSLevel: "test",
	}})

	taskMBReader := new(mockTaskMBReader)
	taskMBReader.On("ReadMB", "123-45-6789").Return(map[int]int{
		0: 1_000, 1: 2_500,
	})

	type fields struct {
		taskManager task.Manager
		mbReader    resctrl.TaskMBReader
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[task.QoSLevel]map[int]int
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				taskManager: taskManager,
				mbReader:    taskMBReader,
			},
			want:    map[task.QoSLevel]map[int]int{"test": {0: 1000, 1: 2500}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := mbMonitor{
				taskManager: tt.fields.taskManager,
				mbReader:    tt.fields.mbReader,
			}
			got, err := t.GetQoSMBs()
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetQoSMBs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetQoSMBs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
