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

package apppool

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		dieTopology *machine.DieTopology
	}
	tests := []struct {
		name string
		args args
		want *Manager
	}{
		{
			name: "happy path",
			args: args{
				dieTopology: &machine.DieTopology{
					Packages: 2,
					NUMAsInPackage: map[int][]int{
						0: {0, 1, 2, 3},
						1: {4, 5, 6, 7},
					},
					DiesInNuma: map[int]sets.Int{
						0: {0: sets.Empty{}, 1: sets.Empty{}},
						1: {2: sets.Empty{}, 3: sets.Empty{}},
						2: {4: sets.Empty{}, 5: sets.Empty{}},
						3: {6: sets.Empty{}, 7: sets.Empty{}},
						4: {8: sets.Empty{}, 9: sets.Empty{}},
						5: {10: sets.Empty{}, 11: sets.Empty{}},
						6: {12: sets.Empty{}, 13: sets.Empty{}},
						7: {14: sets.Empty{}, 15: sets.Empty{}},
					},
				},
			},
			want: &Manager{
				packages: []PoolsPackage{
					&poolsPackage{
						id: 0,
						nodes: sets.Int{
							0: sets.Empty{}, 1: sets.Empty{}, 2: sets.Empty{}, 3: sets.Empty{},
						},
						ccdsByNode: map[int]sets.Int{
							0: {0: sets.Empty{}, 1: sets.Empty{}},
							1: {2: sets.Empty{}, 3: sets.Empty{}},
							2: {4: sets.Empty{}, 5: sets.Empty{}},
							3: {6: sets.Empty{}, 7: sets.Empty{}},
						},
						poolsByNode: make(map[int]AppPool),
					},
					&poolsPackage{
						id: 1,
						nodes: sets.Int{
							4: sets.Empty{}, 5: sets.Empty{}, 6: sets.Empty{}, 7: sets.Empty{},
						},
						ccdsByNode: map[int]sets.Int{
							4: {8: sets.Empty{}, 9: sets.Empty{}},
							5: {10: sets.Empty{}, 11: sets.Empty{}},
							6: {12: sets.Empty{}, 13: sets.Empty{}},
							7: {14: sets.Empty{}, 15: sets.Empty{}},
						},
						poolsByNode: make(map[int]AppPool),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := New(tt.args.dieTopology); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_AddAppPool(t *testing.T) {
	t.Parallel()
	type fields struct {
		packages []PoolsPackage
	}
	type args struct {
		nodes []int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    AppPool
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				packages: []PoolsPackage{
					&poolsPackage{
						id:    0,
						nodes: sets.Int{2: sets.Empty{}, 3: sets.Empty{}},
						ccdsByNode: map[int]sets.Int{
							2: sets.Int{4: sets.Empty{}, 5: sets.Empty{}},
							3: sets.Int{6: sets.Empty{}, 7: sets.Empty{}},
						},
						poolsByNode: make(map[int]AppPool),
					},
				},
			},
			args: args{
				nodes: []int{3},
			},
			want: AppPool(&appPool{
				packageID: 0,
				numaNodes: []int{3},
				ccds:      map[int][]int{3: {6, 7}},
			}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := Manager{
				packages: tt.fields.packages,
			}
			got, err := m.AddAppPool(tt.args.nodes)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAppPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddAppPool() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_DeleteAppPool(t *testing.T) {
	t.Parallel()

	pool := &appPool{
		packageID: 0,
		numaNodes: []int{2},
		ccds:      map[int][]int{2: {4, 5}},
	}

	type fields struct {
		packages []PoolsPackage
	}
	type args struct {
		pool AppPool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				packages: []PoolsPackage{
					&poolsPackage{
						id:          0,
						nodes:       sets.Int{2: sets.Empty{}},
						ccdsByNode:  map[int]sets.Int{2: {4: sets.Empty{}, 5: sets.Empty{}}},
						poolsByNode: map[int]AppPool{2: pool},
						pools:       []AppPool{pool},
					},
				},
			},
			args: args{
				pool: pool,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := Manager{
				packages: tt.fields.packages,
			}
			if err := m.DeleteAppPool(tt.args.pool); (err != nil) != tt.wantErr {
				t.Errorf("DeleteAppPool() error = %v, wantErr %v", err, tt.wantErr)
			}

			pools := tt.fields.packages[0].GetAppPools()
			assert.Empty(t, pools)
		})
	}
}
