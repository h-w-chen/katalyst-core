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

package task

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"reflect"
	"testing"
)

func Test_taskManager_GetTasks(t1 *testing.T) {
	t1.Parallel()
	type fields struct {
		nodeTaskIDs map[int]sets.Int
	}
	type args struct {
		node int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []int
	}{
		{
			name: "happy path",
			fields: fields{
				nodeTaskIDs: map[int]sets.Int{1: {123: sets.Empty{}, 124: sets.Empty{}}},
			},
			args: args{
				node: 1,
			},
			want: []int{123, 124},
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := taskManager{
				nodeTaskIDs: tt.fields.nodeTaskIDs,
			}
			if got := t.GetTasks(tt.args.node); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockPodResource struct {
	mock.Mock
}

func (m *mockPodResource) GetNumaNode(pod *v1.Pod) (int, error) {
	args := m.Called(pod)
	return args.Int(0), args.Error(1)
}

func (m *mockPodResource) GetPid(pod *v1.Pod) (int, error) {
	args := m.Called(pod)
	return args.Int(0), args.Error(1)
}

func Test_taskManager_newTask(t1 *testing.T) {
	t1.Parallel()

	fs := afero.NewMemMapFs()

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID: "pod-12345",
		},
	}
	mockPodRes := new(mockPodResource)
	mockPodRes.On("GetNumaNode", pod).Return(2, nil)
	mockPodRes.On("GetPid", pod).Return(123, nil)

	type fields struct {
		podResource PodResource
		nodeTaskIDs map[int]sets.Int
	}
	type args struct {
		fs  afero.Fs
		pod *v1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Task
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				podResource: mockPodRes,
				nodeTaskIDs: nil,
			},
			args: args{
				fs:  fs,
				pod: pod,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := taskManager{
				podResource: tt.fields.podResource,
				nodeTaskIDs: tt.fields.nodeTaskIDs,
			}
			got, err := t.NewTask(tt.args.pod)
			if (err != nil) != tt.wantErr {
				t1.Errorf("NewTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("NewTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}
