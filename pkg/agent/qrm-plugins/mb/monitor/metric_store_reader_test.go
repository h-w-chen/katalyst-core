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

package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	utilmetric "github.com/kubewharf/katalyst-core/pkg/util/metric"
)

func Test_metricStoreMBReader_getMBQoSGroups(t *testing.T) {
	t.Parallel()

	testTimestamp := time.Date(2024, 11, 11, 15, 0, 1, 0, time.UTC)
	metricTimestamp := time.Date(2024, 11, 11, 15, 0, 0, 0, time.UTC)
	dummyStore := utilmetric.NewMetricStore()
	dummyStore.SetByStringIndex("rmb", map[string]map[int]utilmetric.MetricData{
		"dedicated": {1: {
			Value: 100,
			Time:  &metricTimestamp,
		}},
	})
	dummyStore.SetByStringIndex("wmb", map[string]map[int]utilmetric.MetricData{
		"dedicated": {1: {
			Value: 35,
			Time:  &metricTimestamp,
		}},
	})

	dummyStoreUnbalanced := utilmetric.NewMetricStore()
	dummyStoreUnbalanced.SetByStringIndex("rmb", map[string]map[int]utilmetric.MetricData{
		"dedicated": {1: {
			Value: 100,
			Time:  &metricTimestamp,
		}},
	})
	dummyStoreUnbalanced.SetByStringIndex("wmb", map[string]map[int]utilmetric.MetricData{
		"share-30": {4: {
			Value: 35,
			Time:  &metricTimestamp,
		}},
	})

	type fields struct {
		metricStore *utilmetric.MetricStore
	}
	type args struct {
		now time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[task.QoSGroup]*MBQoSGroup
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				metricStore: dummyStore,
			},
			args: args{now: testTimestamp},
			want: map[task.QoSGroup]*MBQoSGroup{
				"dedicated": {
					CCDs: sets.Int{1: sets.Empty{}},
					CCDMB: map[int]*MBData{1: {
						ReadsMB:  100,
						WritesMB: 35,
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "happy path of unbalanced store",
			fields: fields{
				metricStore: dummyStoreUnbalanced,
			},
			args: args{now: testTimestamp},
			want: map[task.QoSGroup]*MBQoSGroup{
				"dedicated": {
					CCDs: sets.Int{1: sets.Empty{}},
					CCDMB: map[int]*MBData{
						1: {ReadsMB: 100, WritesMB: 0},
					},
				},
				"share-30": {
					CCDs: sets.Int{4: sets.Empty{}},
					CCDMB: map[int]*MBData{
						4: {ReadsMB: 0, WritesMB: 35},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &metricStoreMBReader{
				metricStore: tt.fields.metricStore,
			}
			got, err := m.getMBQoSGroups(tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMBQoSGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
