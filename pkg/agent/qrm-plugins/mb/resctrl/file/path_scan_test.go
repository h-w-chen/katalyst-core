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

package file

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	resctrlconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/consts"
)

func Test_getResctrlMonGroups(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/sys/fs/resctrl/dedicated/mon_groups/podPODxxx", resctrlconsts.FolderPerm)
	_ = fs.MkdirAll("/sys/fs/resctrl/dedicated/mon_groups/podPODyyy", resctrlconsts.FolderPerm)

	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path",
			args: args{
				fs: fs,
			},
			want: []string{
				"/sys/fs/resctrl/dedicated/mon_groups/podPODxxx",
				"/sys/fs/resctrl/dedicated/mon_groups/podPODyyy",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetResctrlMonGroups(tt.args.fs)
			if !tt.wantErr(t, err, fmt.Sprintf("getResctrlMonGroups(%v)", tt.args.fs)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getResctrlMonGroups(%v)", tt.args.fs)
		})
	}
}

func TestGetResctrlCtrlGroups(t *testing.T) {
	t.Parallel()

	fsTest := afero.NewMemMapFs()
	_ = fsTest.MkdirAll("/sys/fs/resctrl/system", 0o755)

	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path",
			args: args{
				fs: fsTest,
			},
			want:    []string{"system"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetResctrlCtrlGroups(tt.args.fs)
			if !tt.wantErr(t, err, fmt.Sprintf("GetResctrlCtrlGroups(%v)", tt.args.fs)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetResctrlCtrlGroups(%v)", tt.args.fs)
		})
	}
}
