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

package monitor

import (
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/readmb"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/writemb"
)

type mockReadMBReader struct {
	mock.Mock
}

func (m *mockReadMBReader) GetMB(qosGroup string) (map[int]int, error) {
	args := m.Called(qosGroup)
	return args.Get(0).(map[int]int), args.Error(1)
}

type mockWriteMBReader struct {
	mock.Mock
}

func (m *mockWriteMBReader) GetMB(ccd int) (int, error) {
	args := m.Called(ccd)
	return args.Int(0), args.Error(1)
}

func Test_mbMonitor_GetMBQoSGroups(t1 *testing.T) {
	t1.Parallel()

	rmbReader := new(mockReadMBReader)
	rmbReader.On("GetMB", "shared-50").Return(map[int]int{2: 200, 3: 300}, nil)

	wmbReader := new(mockWriteMBReader)
	wmbReader.On("GetMB", 2).Return(20, nil)
	wmbReader.On("GetMB", 3).Return(30, nil)

	stubFs := afero.NewMemMapFs()
	_ = stubFs.MkdirAll("/sys/fs/resctrl/shared-50", 0o755)

	type fields struct {
		rmbReader readmb.ReadMBReader
		wmbReader writemb.WriteMBReader
		fs        afero.Fs
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[task.QoSGroup]*MBQoSGroup
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				rmbReader: rmbReader,
				wmbReader: wmbReader,
				fs:        stubFs,
			},
			want: map[task.QoSGroup]*MBQoSGroup{
				"shared-50": {
					CCDs: sets.Int{2: sets.Empty{}, 3: sets.Empty{}},
					CCDMB: map[int]*MBData{
						2: {ReadsMB: 200, WritesMB: 20},
						3: {ReadsMB: 300, WritesMB: 30},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := mbMonitor{
				grmbReader: tt.fields.rmbReader,
				wmbReader:  tt.fields.wmbReader,
				fs:         tt.fields.fs,
			}
			got, err := t.GetMBQoSGroups()
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetMBQoSGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetMBQoSGroups() got = %v, want %v", got, tt.want)
			}
		})
	}
}
