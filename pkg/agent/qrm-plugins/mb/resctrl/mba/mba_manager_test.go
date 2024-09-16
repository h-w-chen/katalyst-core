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
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestMBAManager_CleanupResctrlLayout(t *testing.T) {
	t.Parallel()

	fsExistentFolder := afero.NewMemMapFs()
	fsExistentFolder.MkdirAll("/sys/fs/resctrl/node1", 0755)

	type fields struct {
		packages      int
		mbasByPackage MBAPackage
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
			name: "happy path clean fs",
			fields: fields{
				packages: 1,
				mbasByPackage: map[int]map[int]*MBA{
					0: {
						1: {numaNode: 1, cpus: []int{2, 3}, sharingPackage: 0},
					},
				},
			},
			args: args{
				fs: afero.NewMemMapFs(),
			},
			wantErr: false,
		},
		{
			name: "happy path with existent folders",
			fields: fields{
				packages: 1,
				mbasByPackage: map[int]map[int]*MBA{
					0: {
						1: {numaNode: 1, cpus: []int{2, 3}, sharingPackage: 0},
					},
				},
			},
			args: args{
				fs: fsExistentFolder,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MBAManager{
				packages:      tt.fields.packages,
				mbasByPackage: tt.fields.mbasByPackage,
			}
			if err := m.CleanupResctrlLayout(tt.args.fs); (err != nil) != tt.wantErr {
				t.Errorf("CleanupResctrlLayout() error = %v, wantErr %v", err, tt.wantErr)
			}

			for _, mbas := range tt.fields.mbasByPackage {
				for numaNode, _ := range mbas {
					folderToCleanup := path.Join("/sys/fs/resctrl", fmt.Sprintf("node_%d", numaNode))
					_, err := tt.args.fs.Stat(folderToCleanup)
					assert.EqualError(t, err, "open /sys/fs/resctrl/node_1: file does not exist")
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		packageByNode map[int]int
		diesByNode    map[int]sets.Int
		cpusByDie     map[int][]int
	}
	tests := []struct {
		name    string
		args    args
		want    *MBAManager
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path no error",
			args: args{
				packageByNode: map[int]int{0: 0, 1: 0, 2: 1, 3: 1},
				diesByNode: map[int]sets.Int{
					0: sets.NewInt(0),
					1: sets.NewInt(1),
					2: sets.NewInt(2),
					3: sets.NewInt(3),
				},
				cpusByDie: map[int][]int{
					0: {0, 1, 2},
					1: {3, 4, 5},
					2: {6, 7, 8},
					3: {9, 10, 11},
				},
			},
			want: &MBAManager{
				packages: 2,
				mbasByPackage: map[int]map[int]*MBA{
					0: {
						0: {
							numaNode:       0,
							cpus:           []int{0, 1, 2},
							sharingPackage: 0,
						},
						1: {
							numaNode:       1,
							cpus:           []int{3, 4, 5},
							sharingPackage: 0,
						},
					},
					1: {
						2: {
							numaNode:       2,
							cpus:           []int{6, 7, 8},
							sharingPackage: 1,
						},
						3: {
							numaNode:       3,
							cpus:           []int{9, 10, 11},
							sharingPackage: 1,
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.packageByNode, tt.args.diesByNode, tt.args.cpusByDie)
			if !tt.wantErr(t, err, fmt.Sprintf("New(%v, %v)", tt.args.packageByNode, tt.args.cpusByDie)) {
				return
			}
			assert.Equalf(t, tt.want, got, "New(%v, %v)", tt.args.packageByNode, tt.args.cpusByDie)
		})
	}
}
