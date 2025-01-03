package quotasourcing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_categorySourcer_AttributeMBToSources_matrix(t1 *testing.T) {
	t1.Parallel()
	type args struct {
		domainTargets []DomainMB
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		// both major local traffic
		{
			name: "both to ease, major local",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         70_198,
						MBSource:       27_700,
						MBSourceRemote: 7_700,
					},
					{
						Target:         59_000,
						MBSource:       14_121,
						MBSourceRemote: 5_180,
					},
				},
			},
			want: []int{63_952, 65_245},
		},
		{
			name: "both to throttle, major local",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         30_198,
						MBSource:       47_700,
						MBSourceRemote: 18_490,
					},
					{
						Target:         29_000,
						MBSource:       35_121,
						MBSourceRemote: 5_180,
					},
				},
			},
			want: []int{46_291, 12_908},
		},
		{
			name: "one to throttle, the other to ease, major local",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         30_198,
						MBSource:       47_700,
						MBSourceRemote: 18_490,
					},
					{
						Target:         60_000,
						MBSource:       35_121,
						MBSourceRemote: 5_180,
					},
				},
			},
			want: []int{36_182, 54_016},
		},
		// both major remote traffic
		{
			name: "both throttle, major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         30_198,
						MBSource:       37_700,
						MBSourceRemote: 30_000,
					},
					{
						Target:         20_000,
						MBSource:       44_121,
						MBSourceRemote: 40_180,
					},
				},
			},
			want: []int{21_852, 28_347},
		},
		{
			name: "both to ease, major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         70_198,
						MBSource:       37_700,
						MBSourceRemote: 30_000,
					},
					{
						Target:         50_000,
						MBSource:       44_121,
						MBSourceRemote: 40_180,
					},
				},
			},
			want: []int{55_740, 64_457},
		},
		{
			name: "one to throttle, the other to ease, major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         30_198,
						MBSource:       37_700,
						MBSourceRemote: 30_000,
					},
					{
						Target:         55_000,
						MBSource:       44_121,
						MBSourceRemote: 40_180,
					},
				},
			},
			want: []int{66_573, 18_625},
		},
		// mixed traffic: one major local, the other major remote
		{
			name: "both to ease, one major local, the other major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         70_198,
						MBSource:       37_700,
						MBSourceRemote: 30_000,
					},
					{
						Target:         60_000,
						MBSource:       34_121,
						MBSourceRemote: 10_180,
					},
				},
			},
			want: []int{41_486, 38_448},
		},
		{
			name: "both to throttle, one major local, the other major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         9_198,
						MBSource:       37_700,
						MBSourceRemote: 30_000,
					},
					{
						Target:         30_000,
						MBSource:       44_121,
						MBSourceRemote: 10_180,
					},
				},
			},
			want: []int{0, 38325},
		},
		{
			name: "one to throttle major local, one to ease major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         20_198,
						MBSource:       37_700,
						MBSourceRemote: 9_000,
					},
					{
						Target:         50_000,
						MBSource:       44_121,
						MBSourceRemote: 28_180,
					},
				},
			},
			want: []int{0, 31_559},
		},
		{
			name: "one to ease major local, one to throttle major remote",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         80_198,
						MBSource:       37_700,
						MBSourceRemote: 9_000,
					},
					{
						Target:         20_000,
						MBSource:       44_121,
						MBSourceRemote: 28_180,
					},
				},
			},
			want: []int{83_333, 0},
		},
		// special cases for total remote, simulating cross-fire
		{
			name: "both major total remote, both to ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         80_198,
						MBSource:       17_700,
						MBSourceRemote: 17_700,
					},
					{
						Target:         30_000,
						MBSource:       44_121,
						MBSourceRemote: 44_121,
					},
				},
			},
			want: []int{30_000, 80_198},
		},
		{
			name: "both major total remote, both to throttle",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         20_198,
						MBSource:       57_700,
						MBSourceRemote: 57_700,
					},
					{
						Target:         35_000,
						MBSource:       44_121,
						MBSourceRemote: 44_121,
					},
				},
			},
			want: []int{35_000, 20_198},
		},
		{
			name: "both major total remote, one to throttle and the other ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         40_198,
						MBSource:       57_700,
						MBSourceRemote: 57_700,
					},
					{
						Target:         20_000,
						MBSource:       24_121,
						MBSourceRemote: 24_121,
					},
				},
			},
			want: []int{20_000, 40_198},
		},
		// another special case: both total local
		{
			name: "both major total remote, one to throttle and the other ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         40_198,
						MBSource:       57_700,
						MBSourceRemote: 0,
					},
					{
						Target:         20_000,
						MBSource:       24_121,
						MBSourceRemote: 0,
					},
				},
			},
			want: []int{40_198, 20_000},
		},
		// really conner case: one total local, the other total remote
		{
			name: "one major total local, the other total remote, one to throttle and the other ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         40_198,
						MBSource:       57_700,
						MBSourceRemote: 0,
					},
					{
						Target:         20_000,
						MBSource:       24_121,
						MBSourceRemote: 24_121,
					},
				},
			},
			want: []int{16_077, 24_121},
		},
		{
			name: "one major total local, the other total remote, one to throttle and the other ease - impossible?",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         20_000,
						MBSource:       57_700,
						MBSourceRemote: 0,
					},
					{
						Target:         40_198,
						MBSource:       24_121,
						MBSourceRemote: 24_121,
					},
				},
			},
			want: []int{0, 20_000},
		},
		{
			name: "one major total local, the other total remote, both to ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         40_198,
						MBSource:       24_121,
						MBSourceRemote: 0,
					},
					{
						Target:         20_000,
						MBSource:       10_700,
						MBSourceRemote: 10_700,
					},
				},
			},
			want: []int{26_809, 133_88},
		},
		// test case based on integration half-cross
		{
			name: "one major total local, the other half local; one to throttle, one to ease",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         35_198,
						MBSource:       50_151,
						MBSourceRemote: 0,
					},
					{
						Target:         81_641,
						MBSource:       51_517 + 20_485,
						MBSourceRemote: 20_485,
					},
				},
			},
			want: []int{2_409, 114_430},
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := categorySourcer{}
			assert.Equalf(t1, tt.want, t.AttributeMBToSources(tt.args.domainTargets), "AttributeMBToSources(%v)", tt.args.domainTargets)
		})
	}
}