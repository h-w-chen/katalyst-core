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
	"reflect"
	"testing"

	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
)

func TestNewTask(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/proc/555/task/555", 0555)

	pod := &v1.Pod{}

	type args struct {
		fs   afero.Fs
		pod  *v1.Pod
		node int
		pid  int
	}
	tests := []struct {
		name    string
		args    args
		want    *Task
		wantErr bool
	}{
		{
			name: "happy path no error",
			args: args{
				fs:   fs,
				pod:  pod,
				node: 1111,
				pid:  555,
			},
			want: &Task{
				pod:       pod,
				numaNode:  1111,
				idProcess: 555,
				idThreads: []string{"555"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := newTask(tt.args.fs, tt.args.pod, tt.args.node, tt.args.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_OnReady(t *testing.T) {
	t.Parallel()

	fsClean := afero.NewMemMapFs()

	fsExistent := afero.NewMemMapFs()
	_ = fsExistent.MkdirAll("/sys/fs/resctrl/mon_groups/node_3_pid_123", 0755)

	type fields struct {
		numaNode  int
		pod       *v1.Pod
		idProcess int
		idThreads []string
	}
	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path no error",
			fields: fields{
				numaNode:  3,
				pod:       nil,
				idProcess: 123,
				idThreads: []string{"123", "456"},
			},
			args: args{
				fs: fsClean,
			},
			wantErr: false,
		},
		{
			name: "fail if mon group exists",
			fields: fields{
				numaNode:  3,
				pod:       nil,
				idProcess: 123,
				idThreads: []string{"123"},
			},
			args: args{
				fs: fsExistent,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := Task{
				numaNode:  tt.fields.numaNode,
				pod:       tt.fields.pod,
				idProcess: tt.fields.idProcess,
				idThreads: tt.fields.idThreads,
			}
			if err := t.CreateResctrlMoniker(tt.args.fs); (err != nil) != tt.wantErr {
				t1.Errorf("CreateResctrlMoniker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_OnTerminate(t1 *testing.T) {
	t1.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/sys/fs/resctrl/mon_groups/node_3_pid_123", 0755)

	type fields struct {
		numaNode  int
		pod       *v1.Pod
		idProcess int
		idThreads []string
	}
	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path no error",
			fields: fields{
				numaNode:  3,
				idProcess: 123,
			},
			args: args{
				fs: fs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := Task{
				numaNode:  tt.fields.numaNode,
				pod:       tt.fields.pod,
				idProcess: tt.fields.idProcess,
				idThreads: tt.fields.idThreads,
			}
			if err := t.CleanupResctrlMoniker(tt.args.fs); (err != nil) != tt.wantErr {
				t1.Errorf("CleanupResctrlMoniker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
