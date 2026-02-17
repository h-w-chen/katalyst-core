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

package adjuster

import (
	"reflect"
	"testing"
)

func TestFeedbackAdjuster_AdjustOutgoingTargets(t *testing.T) {
	t.Parallel()
	type fields struct {
		maxRemoteInMB int
	}
	type args struct {
		targets  []int
		currents []int
	}
	tests := []struct {
		name   string
		fields fields
		args0  args
		args1  args
		want   []int
	}{
		{
			name: "happy path",
			fields: fields{
				maxRemoteInMB: 15_000,
			},
			args0: args{
				targets:  []int{10_000, 20_000},
				currents: nil,
			},
			args1: args{
				targets:  []int{10_000, 20_000},
				currents: []int{12_000, 25_000},
			},
			want: []int{8_333, 16_000},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := feedbackAdjuster{}
			_ = r.AdjustOutgoingTargets(tt.args0.targets, tt.args0.currents)
			if got := r.AdjustOutgoingTargets(tt.args1.targets, tt.args1.currents); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdjustOutgoingTargets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_feedback(t *testing.T) {
	type args struct {
		x0 int
		x1 int
		y1 int
	}
	tests := []struct {
		name   string
		args   args
		wantY0 int
	}{
		{
			name: "fixed-target feedback",
			args: args{
				x0: 306,
				x1: 220,
				y1: 306, // 306 as the fixed target
			},
			wantY0: 425,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotY0 := feedback(tt.args.x0, tt.args.x1, tt.args.y1); gotY0 != tt.wantY0 {
				t.Errorf("feedback() = %v, want %v", gotY0, tt.wantY0)
			}
		})
	}
}
