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

package policydeliver

import (
	"reflect"
	"testing"
)

func Test_ccdDistributeMB(t *testing.T) {
	t.Parallel()
	type args struct {
		total int
		mbCCD map[int]int
	}
	tests := []struct {
		name string
		args args
		want map[int]int
	}{
		{
			name: "happy path of equal ccds",
			args: args{
				total: 100,
				mbCCD: map[int]int{6: 20, 7: 20},
			},
			want: map[int]int{6: 50, 7: 50},
		},
		{
			name: "proportional distribution",
			args: args{
				total: 90,
				mbCCD: map[int]int{6: 80, 7: 40},
			},
			want: map[int]int{6: 60, 7: 30},
		},
		{
			name: "treat as equal when both are too small",
			args: args{
				total: 60_000,
				mbCCD: map[int]int{6: 5_000, 7: 15_000},
			},
			want: map[int]int{6: 30_000, 7: 30_000},
		},
		{
			name: "equally split for preempt node",
			args: args{
				total: 35_000,
				mbCCD: map[int]int{6: 0, 7: 0},
			},
			want: map[int]int{6: 17_500, 7: 17_500},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := distributeCCDMBs(tt.args.total, tt.args.mbCCD); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("distributeCCDMBs() = %v, want %v", got, tt.want)
			}
		})
	}
}
