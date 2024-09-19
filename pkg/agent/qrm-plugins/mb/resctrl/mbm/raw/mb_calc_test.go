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

package raw

import (
	"github.com/spf13/afero"
	"reflect"
	"testing"
	"time"
)

func Test_readRawData(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/sys/fs/resctrl/node_1/mon_data/mon_L3_02", []byte("1234567890123456789"), 0644)
	_ = afero.WriteFile(fs, "/sys/fs/resctrl/node_1/mon_data/mon_L3_03", []byte("Unavailable"), 0644)

	type args struct {
		fs   afero.Fs
		path string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "happy path to get byte count",
			args: args{
				fs:   fs,
				path: "/sys/fs/resctrl/node_1/mon_data/mon_L3_02",
			},
			want: 1234567890123456789,
		},
		{
			name: "Unavailable should be ignored",
			args: args{
				fs:   fs,
				path: "/sys/fs/resctrl/node_1/mon_data/mon_L3_03",
			},
			want: -1,
		},
		{
			name: "file not exist should be ignore",
			args: args{
				fs:   fs,
				path: "/sys/fs/resctrl/node_100/mon_data/mon_L3_02",
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := readRawData(tt.args.fs, tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readRawData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calcAverageMBinMBps(t *testing.T) {
	t.Parallel()
	type args struct {
		currV    int64
		nowTime  time.Time
		lastV    int64
		lastTime time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				currV:    12_340_000_000,
				nowTime:  time.Date(2024, 9, 18, 17, 30, 45, 0, time.UTC),
				lastV:    10_000_000_000,
				lastTime: time.Date(2024, 9, 18, 17, 30, 43, 0, time.UTC),
			},
			want:    1_170,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calcAverageMBinMBps(tt.args.currV, tt.args.nowTime, tt.args.lastV, tt.args.lastTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("calcAverageMBinMBps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("calcAverageMBinMBps() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMB(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/sys/fs/resctrl/node_2/mon_data/mon_L3_04/mbm_total_bytes", []byte("2000000000"), 0644)

	type args struct {
		fs         afero.Fs
		monGroup   string
		dies       []int
		ts         time.Time
		dataKeeper rawDataKeeper
	}
	tests := []struct {
		name string
		args args
		want map[int]int
	}{
		{
			name: "happy path",
			args: args{
				fs:       fs,
				monGroup: "/sys/fs/resctrl/node_2",
				dies:     []int{4, 5},
				ts:       time.Date(2024, 9, 18, 19, 57, 46, 0, time.UTC),
				dataKeeper: map[string]rawData{
					"/sys/fs/resctrl/node_2/mon_data/mon_L3_04/mbm_total_bytes": {value: 1_000_000_000, readTime: time.Date(2024, 9, 18, 19, 57, 45, 0, time.UTC)},
				},
			},
			want: map[int]int{4: 1000},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := getMB(tt.args.fs, tt.args.monGroup, tt.args.dies, tt.args.ts, tt.args.dataKeeper); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMB() = %v, want %v", got, tt.want)
			}
		})
	}
}
