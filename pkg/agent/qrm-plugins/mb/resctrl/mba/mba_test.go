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

package mba

import (
	"fmt"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestMBA_CreateResctrlControlGroup(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	type fields struct {
		numaNode       int
		cpus           []int
		sharingPackage int
	}
	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path no error",
			fields: fields{
				numaNode:       1,
				cpus:           []int{2, 3},
				sharingPackage: 0,
			},
			args: args{
				fs: fs,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MBA{
				numaNode:       tt.fields.numaNode,
				cpus:           tt.fields.cpus,
				sharingPackage: tt.fields.sharingPackage,
			}
			tt.wantErr(t, m.CreateResctrlControlGroup(tt.args.fs), fmt.Sprintf("CreateResctrlControlGroup(%v)", tt.args.fs))

			folderToCleanup := path.Join("/sys/fs/resctrl", fmt.Sprintf("node_%d", tt.fields.numaNode))
			_, err := tt.args.fs.Stat(folderToCleanup)
			assert.NoError(t, err)
		})
	}
}