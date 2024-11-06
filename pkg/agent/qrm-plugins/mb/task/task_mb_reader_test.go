package task

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
)

type mockMonGrpReader struct {
	mock.Mock
}

func (m *mockMonGrpReader) ReadMB(monGroup string, dies []int) (map[int]int, error) {
	args := m.Called(monGroup, dies)
	return args.Get(0).(map[int]int), args.Error(1)
}

func Test_taskMBReader_ReadMB(t1 *testing.T) {
	t1.Parallel()

	mockMGReader := new(mockMonGrpReader)
	mockMGReader.On("ReadMB", "/sys/fs/resctrl/reclaimed/mon_groups/pod123-321-1122", []int{4, 5}).
		Return(map[int]int{4: 111, 5: 222}, nil)

	type fields struct {
		monGroupReader resctrl.MonGroupReader
	}
	type args struct {
		task *Task
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[int]int
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				monGroupReader: mockMGReader,
			},
			args: args{
				task: &Task{
					QoSLevel: "reclaimed_cores",
					PodUID:   "123-321-1122",
					NumaNode: []int{2},
					nodeCCDs: map[int]sets.Int{2: {4: sets.Empty{}, 5: sets.Empty{}}},
				},
			},
			want:    map[int]int{4: 111, 5: 222},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := taskMBReader{
				monGroupReader: tt.fields.monGroupReader,
			}
			got, err := t.ReadMB(tt.args.task)
			if (err != nil) != tt.wantErr {
				t1.Errorf("ReadMB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("ReadMB() got = %v, want %v", got, tt.want)
			}
		})
	}
}
