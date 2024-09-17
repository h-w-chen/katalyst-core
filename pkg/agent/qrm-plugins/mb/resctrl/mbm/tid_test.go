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
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

func Test_getThreads(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/proc/123/task/123", 0555)
	_ = fs.MkdirAll("/proc/123/task/456", 0555)

	type args struct {
		fs  afero.Fs
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "happy path of 2 threads",
			args: args{
				fs:  fs,
				pid: 123,
			},
			want:    []string{"123", "456"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := getThreads(tt.args.fs, tt.args.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("getThreads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getThreads() got = %v, want %v", got, tt.want)
			}
		})
	}
}
