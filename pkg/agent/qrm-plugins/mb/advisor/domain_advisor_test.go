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

package advisor

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/adjuster"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/distributor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/quota"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/resource"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/sankey"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/domain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/plan"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

func Test_domainAdvisor_getEffectiveCapacity(t *testing.T) {
	t.Parallel()
	type fields struct {
		domains               domain.Domains
		xDomGroups            sets.String
		groupCapacityInMB     map[string]int
		defaultDomainCapacity int
	}
	type args struct {
		domID         int
		incomingStats monitor.GroupMBStats
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "trivial base capacity",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, nil, 88888),
				},
				defaultDomainCapacity: 12_345,
			},
			args: args{
				domID:         0,
				incomingStats: monitor.GroupMBStats{},
			},
			want:    12_345,
			wantErr: false,
		},
		{
			name: "min of active settings takes effect",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, nil, 88888),
				},
				groupCapacityInMB:     map[string]int{"shared-60": 9_999, "shared-50": 11_111},
				defaultDomainCapacity: 12_345,
			},
			args: args{
				domID: 0,
				incomingStats: monitor.GroupMBStats{
					"shared-50": monitor.GroupMB{0: {TotalMB: 5_555}},
					"shared-60": monitor.GroupMB{1: {TotalMB: 2_222}},
				},
			},
			want:    9_999,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &domainAdvisor{
				domains:               tt.fields.domains,
				xDomGroups:            tt.fields.xDomGroups,
				groupCapacityInMB:     tt.fields.groupCapacityInMB,
				defaultDomainCapacity: tt.fields.defaultDomainCapacity,
			}
			got, err := d.getEffectiveCapacity(tt.args.domID, tt.args.incomingStats)
			if (err != nil) != tt.wantErr {
				t.Errorf("getEffectiveCapacity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getEffectiveCapacity() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_domainAdvisor_calcIncomingQuotas(t *testing.T) {
	t.Parallel()
	type fields struct {
		domains               domain.Domains
		XDomGroups            sets.String
		GroupCapacityInMB     map[string]int
		defaultDomainCapacity int
	}
	type args struct {
		ctx context.Context
		mon *monitor.DomainStats
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[int]*resource.MBGroupIncomingStat
		wantErr bool
	}{
		{
			name: "happy path of no dynamic capacity",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, sets.NewInt(0, 1), 88888),
				},
				defaultDomainCapacity: 10_000,
			},
			args: args{
				ctx: context.TODO(),
				mon: &monitor.DomainStats{
					Incomings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								1: {TotalMB: 2_200},
							},
							"shared-50": map[int]monitor.MBInfo{
								0: {TotalMB: 5_500},
								1: {TotalMB: 1_100},
							},
						},
					},
				},
			},
			want: map[int]*resource.MBGroupIncomingStat{
				0: {
					CapacityInMB: 10_000,
					FreeInMB:     10_000 - 2_200 - 6_600,
					GroupSorted: []sets.String{
						sets.NewString("shared-60"),
						sets.NewString("shared-50"),
					},
					GroupTotalUses: map[string]int{
						"shared-60": 2_200,
						"shared-50": 5_500 + 1_100,
					},
					GroupLimits: map[string]int{
						"shared-60": 10_000,
						"shared-50": 10_000 - 2_200,
					},
					ResourceState: resource.State("fit"),
				},
			},
			wantErr: false,
		},
		{
			name: "dynamic capacity in force",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, sets.NewInt(0, 1), 88888),
				},
				GroupCapacityInMB: map[string]int{
					"shared-60": 6_000,
				}, defaultDomainCapacity: 10_000,
			},
			args: args{
				ctx: context.TODO(),
				mon: &monitor.DomainStats{
					Incomings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								1: {TotalMB: 2_200},
							},
							"shared-50": map[int]monitor.MBInfo{
								0: {TotalMB: 5_500},
								1: {TotalMB: 1_100},
							},
						},
					},
				},
			},
			want: map[int]*resource.MBGroupIncomingStat{
				0: {
					CapacityInMB: 6_000,
					FreeInMB:     0,
					GroupSorted: []sets.String{
						sets.NewString("shared-60"),
						sets.NewString("shared-50"),
					},
					GroupTotalUses: map[string]int{
						"shared-60": 2_200,
						"shared-50": 5_500 + 1_100,
					},
					GroupLimits: map[string]int{
						"shared-60": 6_000,
						"shared-50": 6_000 - 2_200,
					},
					ResourceState: resource.State("underStress"),
				},
			},
			wantErr: false,
		},
		//{
		//	name: "integration test case",
		//	fields: fields{
		//		domains:               domain.Domains{
		//			0: domain.NewDomain(0, sets.NewInt(0,1,2,3,4,5), 123),
		//			1: domain.NewDomain(1, sets.NewInt(6,7,8,9,10,11), 123),
		//			2: domain.NewDomain(2, sets.NewInt(16,17,18,19,20,21), 123),
		//			3: domain.NewDomain(3, sets.NewInt(22,23,24,25,26,27), 123),
		//		},
		//		XDomGroups:            nil,
		//		GroupCapacityInMB:     nil,
		//		defaultDomainCapacity: 100 * 1024,
		//	},
		//	args:    args{
		//		ctx: context.TODO(),
		//		mon: &monitor.DomainStats{},
		//	},
		//	want:    nil,
		//	wantErr: false,
		//},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &domainAdvisor{
				domains:               tt.fields.domains,
				xDomGroups:            tt.fields.XDomGroups,
				groupCapacityInMB:     tt.fields.GroupCapacityInMB,
				defaultDomainCapacity: tt.fields.defaultDomainCapacity,
			}
			got, err := d.calcIncomingDomainStats(tt.args.ctx, tt.args.mon)
			if (err != nil) != tt.wantErr {
				t.Errorf("calcIncomingDomainStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calcIncomingDomainStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_domainAdvisor_GetPlan(t *testing.T) {
	t.Parallel()
	type fields struct {
		domains               domain.Domains
		defaultDomainCapacity int
		XDomGroups            sets.String
		GroupCapacityInMB     map[string]int
		quotaStrategy         quota.Decider
		flower                sankey.DomainFlower
		adjusters             map[string]adjuster.Adjuster
	}
	type args struct {
		ctx        context.Context
		domainsMon *monitor.DomainStats
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *plan.MBPlan
		wantErr bool
	}{
		{
			name: "happy path of no plan update",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, sets.NewInt(0, 1), 88888),
					1: domain.NewDomain(1, sets.NewInt(2, 3), 88888),
				},
				defaultDomainCapacity: 30_000,
				XDomGroups:            nil,
				GroupCapacityInMB:     nil,
				quotaStrategy:         quota.New(),
				flower:                sankey.New(),
				adjusters:             map[string]adjuster.Adjuster{},
			},
			args: args{
				ctx: context.TODO(),
				domainsMon: &monitor.DomainStats{
					Incomings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								0: {
									LocalMB:  10_000,
									RemoteMB: 2_000,
									TotalMB:  12_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								1: {
									LocalMB:  11_000,
									RemoteMB: 2_500,
									TotalMB:  13_500,
								},
							},
						},
						1: {
							"shared-60": map[int]monitor.MBInfo{
								2: {
									LocalMB:  10_000,
									RemoteMB: 2_000,
									TotalMB:  12_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								3: {
									LocalMB:  11_000,
									RemoteMB: 2_500,
									TotalMB:  13_500,
								},
							},
						},
					},
					Outgoings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								0: {
									LocalMB:  10_000,
									RemoteMB: 2_000,
									TotalMB:  12_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								1: {
									LocalMB:  11_000,
									RemoteMB: 2_500,
									TotalMB:  13_500,
								},
							},
						},
						1: {
							"shared-60": map[int]monitor.MBInfo{
								2: {
									LocalMB:  10_000,
									RemoteMB: 2_000,
									TotalMB:  12_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								3: {
									LocalMB:  11_000,
									RemoteMB: 2_500,
									TotalMB:  13_500,
								},
							},
						},
					},
					OutgoingGroupSumStat: map[string][]monitor.MBInfo{
						"shared-60": {
							0: {
								LocalMB:  10_000,
								RemoteMB: 2_000,
								TotalMB:  12_000,
							},
							1: {
								LocalMB:  10_000,
								RemoteMB: 2_000,
								TotalMB:  12_000,
							},
						},
						"shared-50": {
							0: {
								LocalMB:  11_000,
								RemoteMB: 2_500,
								TotalMB:  13_500,
							},
							1: {
								LocalMB:  11_000,
								RemoteMB: 2_500,
								TotalMB:  13_500,
							},
						},
					},
				},
			},
			want: &plan.MBPlan{MBGroups: map[string]plan.GroupCCDPlan{
				"shared-60": {
					0: 20_000, // high group enjoy full capacity bounded by ccd-max 20G
					2: 20_000,
				},
				"shared-50": {
					1: 18_000, // low group has whatever high ones left behind
					3: 18_000,
				},
			}},
			wantErr: false,
		},
		{
			name: "happy path of yes plan update",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, sets.NewInt(0, 1), 88888),
					1: domain.NewDomain(1, sets.NewInt(2, 3), 88888),
				},
				defaultDomainCapacity: 30_000,
				XDomGroups:            nil,
				GroupCapacityInMB:     nil,
				quotaStrategy:         quota.New(),
				flower:                sankey.New(),
				adjusters:             map[string]adjuster.Adjuster{},
			},
			args: args{
				ctx: context.TODO(),
				domainsMon: &monitor.DomainStats{
					Incomings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								0: {
									LocalMB:  10_000,
									RemoteMB: 5_000,
									TotalMB:  15_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								1: {
									LocalMB:  11_000,
									RemoteMB: 4_500,
									TotalMB:  15_500,
								},
							},
						},
						1: {
							"shared-60": map[int]monitor.MBInfo{
								2: {
									LocalMB:  10_000,
									RemoteMB: 5_000,
									TotalMB:  15_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								3: {
									LocalMB:  11_000,
									RemoteMB: 4_500,
									TotalMB:  15_500,
								},
							},
						},
					},
					Outgoings: map[int]monitor.GroupMBStats{
						0: {
							"shared-60": map[int]monitor.MBInfo{
								0: {
									LocalMB:  10_000,
									RemoteMB: 5_000,
									TotalMB:  15_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								1: {
									LocalMB:  11_000,
									RemoteMB: 4_500,
									TotalMB:  15_500,
								},
							},
						},
						1: {
							"shared-60": map[int]monitor.MBInfo{
								2: {
									LocalMB:  10_000,
									RemoteMB: 5_000,
									TotalMB:  15_000,
								},
							},
							"shared-50": map[int]monitor.MBInfo{
								3: {
									LocalMB:  11_000,
									RemoteMB: 4_500,
									TotalMB:  15_500,
								},
							},
						},
					},
					OutgoingGroupSumStat: map[string][]monitor.MBInfo{
						"shared-60": {
							0: {
								LocalMB:  10_000,
								RemoteMB: 5_000,
								TotalMB:  15_000,
							},
							1: {
								LocalMB:  10_000,
								RemoteMB: 5_000,
								TotalMB:  15_000,
							},
						},
						"shared-50": {
							0: {
								LocalMB:  11_000,
								RemoteMB: 4_000,
								TotalMB:  15_000,
							},
							1: {
								LocalMB:  11_000,
								RemoteMB: 4_000,
								TotalMB:  15_000,
							},
						},
					},
				},
			},
			want: &plan.MBPlan{MBGroups: map[string]plan.GroupCCDPlan{
				"shared-60": {0: 20_000, 2: 20_000},
				"shared-50": {1: 13_500, 3: 13_500},
			}},
			wantErr: false,
		},
		{
			name: "2 levels of groups",
			fields: fields{
				domains: domain.Domains{
					0: domain.NewDomain(0, sets.NewInt(0, 1, 2, 3, 4, 5), 15_000),
					1: domain.NewDomain(1, sets.NewInt(6, 7, 8, 9, 10, 11), 15_000),
					2: domain.NewDomain(2, sets.NewInt(16, 17, 18, 19, 20, 21), 15_000),
					3: domain.NewDomain(3, sets.NewInt(22, 23, 24, 25, 26, 27), 15_000),
				},
				defaultDomainCapacity: 100 * 1024,
				quotaStrategy:         nil,
				flower:                nil,
				adjusters:             nil,
			},
			args: args{
				ctx:        context.TODO(),
				domainsMon: &monitor.DomainStats{
					/*&{map[
						0: 	map[
							/:			map[0:{15 302 317} 2:{3 37 40} 4:{11 39 50} 6:{3 27 30} 8:{121 149 270} 10:{1193 325 1518}]
							shared-50:	map[0:{0 0 0} 2:{0 0 0} 4:{0 0 0} 6:{0 0 0} 8:{0 0 0} 10:{0 0 0}]
						]
						1:	map[
							/:			map[1:{10 48 58} 3:{35 88 123} 5:{29 112 141} 7:{18 64 82} 9:{84 65 149} 11:{27 63 90}]
							shared-50:	map[1:{43 308 351} 3:{3 16 19} 5:{2 8 10} 7:{30 275 305} 9:{19 162 181} 11:{10 7 17}]
						]
						2:	map[
							/:			map[16:{6 62 68} 18:{15 45 60} 20:{9 86 95} 22:{86 55 141} 24:{8 67 75} 26:{22 73 95}]
							shared-50:	map[16:{0 0 0} 18:{0 0 0} 20:{0 0 0} 22:{0 0 0} 24:{0 0 0} 26:{0 0 0}]
						]
						3:	map[
							/:			map[17:{33 72 105} 19:{16 101 117} 21:{64 106 170} 23:{51 127 178} 25:{35 108 143} 27:{44 82 126}]
							shared-50:	map[17:{0 0 0} 19:{0 0 0} 21:{0 0 0} 23:{0 0 0} 25:{0 0 0} 27:{0 0 0}]
						]
					] map[
						0:	map[
							/:			map[0:{15 302 317} 2:{3 37 40} 4:{11 39 50} 6:{3 27 30} 8:{121 149 270} 10:{1193 325 1518}]
							shared-50:	map[0:{0 0 0} 2:{0 0 0} 4:{0 0 0} 6:{0 0 0} 8:{0 0 0} 10:{0 0 0}]
						]
						1:	map[
							/:			map[1:{10 48 58} 3:{35 88 123} 5:{29 112 141} 7:{18 64 82} 9:{84 65 149} 11:{27 63 90}]
							shared-50:	map[1:{43 308 351} 3:{3 16 19} 5:{2 8 10} 7:{30 275 305} 9:{19 162 181} 11:{10 7 17}]
						]
						2:	map[
							/:			map[16:{6 62 68} 18:{15 45 60} 20:{9 86 95} 22:{86 55 141} 24:{8 67 75} 26:{22 73 95}]
							shared-50:	map[16:{0 0 0} 18:{0 0 0} 20:{0 0 0} 22:{0 0 0} 24:{0 0 0} 26:{0 0 0}]
						]
						3:	map[
							/:			map[17:{33 72 105} 19:{16 101 117} 21:{64 106 170} 23:{51 127 178} 25:{35 108 143} 27:{44 82 126}]
							shared-50:	map[17:{0 0 0} 19:{0 0 0} 21:{0 0 0} 23:{0 0 0} 25:{0 0 0} 27:{0 0 0}]
						]
					] map[
						/:				[{1346 879 2225} {203 440 643} {146 388 534} {243 596 839}]
						shared-50:		[{0 0 0} {107 776 883} {0 0 0} {0 0 0}]]}
					*/
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &domainAdvisor{
				domains:               tt.fields.domains,
				defaultDomainCapacity: tt.fields.defaultDomainCapacity,
				xDomGroups:            tt.fields.XDomGroups,
				groupCapacityInMB:     tt.fields.GroupCapacityInMB,
				quotaStrategy:         tt.fields.quotaStrategy,
				flower:                tt.fields.flower,
				adjusters:             tt.fields.adjusters,
				ccdDistribute:         distributor.New(0, 20_000),
				emitter:               &metrics.DummyMetrics{},
			}
			got, err := d.GetPlan(tt.args.ctx, tt.args.domainsMon)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPlan() got = %v, want %v", got, tt.want)
			}
		})
	}
}
