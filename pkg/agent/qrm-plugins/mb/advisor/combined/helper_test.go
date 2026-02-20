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

package combined

import (
	"reflect"
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/priority"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

func Test_preProcessGroupInfo(t *testing.T) {
	t.Parallel()

	priority.GetInstance().AddWeight("machine", 9_000)

	tests := []struct {
		name           string
		stats          monitor.GroupMBStats
		wantResult     monitor.GroupMBStats
		wantGroupInfos monitor.DomainGroupMapping
		wantErr        bool
	}{
		{
			name:           "empty stats",
			stats:          monitor.GroupMBStats{},
			wantResult:     monitor.GroupMBStats{},
			wantGroupInfos: monitor.DomainGroupMapping{},
			wantErr:        false,
		},
		{
			name: "single group - no combination needed",
			stats: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					1: {LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
			wantResult: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					1: {LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
			wantGroupInfos: monitor.DomainGroupMapping{},
			wantErr:        false,
		},
		{
			name: "two groups with different weights - no combination",
			stats: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
				},
				"share-50": {
					1: {LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
			wantResult: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
				},
				"share-50": {
					1: {LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
			wantGroupInfos: monitor.DomainGroupMapping{},
			wantErr:        false,
		},
		{
			name: "two groups with same weight - combine into one",
			stats: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
				},
				"machine": {
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
			},
			wantResult: monitor.GroupMBStats{
				"combined-9000": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
			},
			wantGroupInfos: monitor.DomainGroupMapping{
				"combined-9000": {
					"dedicated": {0: struct{}{}},
					"machine":   {1: struct{}{}},
				},
			},
			wantErr: false,
		},
		{
			name: "three groups with two weights",
			stats: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
				},
				"machine": {
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
				"system": {
					2: {LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
			},
			wantResult: monitor.GroupMBStats{
				"combined-9000": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
				"system": {
					2: {LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
			},
			wantGroupInfos: monitor.DomainGroupMapping{
				"combined-9000": {
					"dedicated": {0: struct{}{}},
					"machine":   {1: struct{}{}},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed groups - some combined some not",
			stats: monitor.GroupMBStats{
				"dedicated": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
				},
				"machine": {
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
				"share-50": {
					2: {LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
			},
			wantResult: monitor.GroupMBStats{
				"combined-9000": {
					0: {LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
					1: {LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
				"share-50": {
					2: {LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
			},
			wantGroupInfos: monitor.DomainGroupMapping{
				"combined-9000": {
					"dedicated": {0: struct{}{}},
					"machine":   {1: struct{}{}},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotResult, gotGroupInfos, err := preProcessGroupInfo(tt.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("preProcessGroupInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("preProcessGroupInfo() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if !reflect.DeepEqual(gotGroupInfos, tt.wantGroupInfos) {
				t.Errorf("preProcessGroupInfo() gotGroupInfos = %v, want %v", gotGroupInfos, tt.wantGroupInfos)
			}
		})
	}
}

func Test_preProcessGroupSumStat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		sumStats map[string][]monitor.MBInfo
		want     map[string][]monitor.MBInfo
	}{
		{
			name:     "empty stats",
			sumStats: map[string][]monitor.MBInfo{},
			want:     map[string][]monitor.MBInfo{},
		},
		{
			name: "single group - no combination needed",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
			want: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
			},
		},
		{
			name: "two groups with different weights - no combination",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
				"share-50": {
					{LocalMB: 3_000, RemoteMB: 1_000, TotalMB: 4_000},
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
				},
			},
			want: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
				"share-50": {
					{LocalMB: 3_000, RemoteMB: 1_000, TotalMB: 4_000},
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
				},
			},
		},
		{
			name: "two groups with same weight - combine and sum",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
				"machine": {
					{LocalMB: 3_000, RemoteMB: 1_000, TotalMB: 4_000},
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
				},
			},
			want: map[string][]monitor.MBInfo{
				"combined-9000": {
					{LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
					{LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
			},
		},
		{
			name: "three groups with same weight - combine and sum all",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
				"machine": {
					{LocalMB: 3_000, RemoteMB: 1_000, TotalMB: 4_000},
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
				},
				"system": {
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
					{LocalMB: 1_000, RemoteMB: 500, TotalMB: 1_500},
				},
			},
			want: map[string][]monitor.MBInfo{
				"combined-9000": {
					{LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
					{LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
				"system": {
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
					{LocalMB: 1_000, RemoteMB: 500, TotalMB: 1_500},
				},
			},
		},
		{
			name: "mixed groups - some combined some not",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 5_000, RemoteMB: 3_000, TotalMB: 8_000},
					{LocalMB: 4_000, RemoteMB: 2_000, TotalMB: 6_000},
				},
				"machine": {
					{LocalMB: 3_000, RemoteMB: 1_000, TotalMB: 4_000},
					{LocalMB: 2_000, RemoteMB: 1_000, TotalMB: 3_000},
				},
				"share-50": {
					{LocalMB: 1_000, RemoteMB: 500, TotalMB: 1_500},
					{LocalMB: 800, RemoteMB: 400, TotalMB: 1_200},
				},
			},
			want: map[string][]monitor.MBInfo{
				"combined-9000": {
					{LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
					{LocalMB: 6_000, RemoteMB: 3_000, TotalMB: 9_000},
				},
				"share-50": {
					{LocalMB: 1_000, RemoteMB: 500, TotalMB: 1_500},
					{LocalMB: 800, RemoteMB: 400, TotalMB: 1_200},
				},
			},
		},
		{
			name: "single domain stats",
			sumStats: map[string][]monitor.MBInfo{
				"dedicated": {
					{LocalMB: 10_000, RemoteMB: 5_000, TotalMB: 15_000},
				},
				"machine": {
					{LocalMB: 8_000, RemoteMB: 4_000, TotalMB: 12_000},
				},
			},
			want: map[string][]monitor.MBInfo{
				"combined-9000": {
					{LocalMB: 18_000, RemoteMB: 9_000, TotalMB: 27_000},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := preProcessGroupSumStat(tt.sumStats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("preProcessGroupSumStat() = %v, want %v", got, tt.want)
			}
		})
	}
}
