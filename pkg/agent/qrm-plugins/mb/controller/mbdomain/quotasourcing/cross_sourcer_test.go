package quotasourcing

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrossSourcer_AttributeMBToSources(t *testing.T) {
	t.Parallel()
	type args struct {
		domainTargets []DomainMB
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			name: "none constraint",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         -1,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
					{
						Target:         -1,
						MBSource:       5_000,
						MBSourceRemote: 2_000,
					},
				},
			},
			want: []int{-1, -1},
		},
		{
			name: "one side of constraint",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         -1,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
					{
						Target:         7_000,
						MBSource:       14_000,
						MBSourceRemote: 6_000,
					},
				},
			},
			want: []int{8765, 7648},
		},
		{
			name: "flipping side of constraint",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         7_000,
						MBSource:       14_000,
						MBSourceRemote: 6_000,
					},
					{
						Target:         -1,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
				},
			},
			want: []int{7648, 8765},
		},
		{
			name: "both domains have constraints",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         7_000,
						MBSource:       14_000,
						MBSourceRemote: 6_000,
					},
					{
						Target:         12_000,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
				},
			},
			want: []int{7648, 8765},
		},
		{
			name: "cross point is the best",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         12_000,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
					{
						Target:         12_000,
						MBSource:       10_000,
						MBSourceRemote: 3_000,
					},
				},
			},
			want: []int{12_000, 12_000},
		},
		{
			name: "little socket traffic",
			args: args{
				domainTargets: []DomainMB{
					{
						Target:         70_198,
						MBSource:       27_700,
						MBSourceRemote: 27_700 - 18_490,
					},
					{
						Target:         59_000,
						MBSource:       14_121,
						MBSourceRemote: 14_121 - 5_180,
					},
				},
			},
			want: []int{54993, 52891},
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
			want: []int{34372, 18388},
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
			want: []int{40_540, 36_426},
		},
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
			want: []int{21_964, 28_233},
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
			want: []int{39_667, 24_263},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := CrossSourcer{}
			if got := c.AttributeMBToSources(tt.args.domainTargets); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AttributeMBToSources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getLeftEndpoint(t *testing.T) {
	t.Parallel()
	type args struct {
		hostDomain   DomainMB
		remoteDomain DomainMB
	}
	tests := []struct {
		name            string
		args            args
		wantHostQuota   int
		wantRemoteQuota int
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "trivial diagonal line",
			args: args{
				hostDomain: DomainMB{
					Target:         8_000,
					MBSource:       6_000,
					MBSourceRemote: 2_000,
				},
				remoteDomain: DomainMB{
					Target:         111_111_111,
					MBSource:       5_000,
					MBSourceRemote: 2_000,
				},
			},
			wantHostQuota:   4_000,
			wantRemoteQuota: 13_333,
			wantErr:         assert.NoError,
		},
		{
			name: "no local",
			args: args{
				hostDomain: DomainMB{
					Target:         8_000,
					MBSource:       6_000,
					MBSourceRemote: 6_000,
				},
				remoteDomain: DomainMB{
					Target:         111_111_111,
					MBSource:       5_000,
					MBSourceRemote: 2_000,
				},
			},
			wantHostQuota:   4_000,
			wantRemoteQuota: 20_000,
			wantErr:         assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotHostQuota, gotRemoteQuota, err := getLeftEndpoint(tt.args.hostDomain, tt.args.remoteDomain)
			if !tt.wantErr(t, err, fmt.Sprintf("getLeftEndpoint(%v, %v)", tt.args.hostDomain, tt.args.remoteDomain)) {
				return
			}
			assert.Equalf(t, tt.wantHostQuota, gotHostQuota, "getLeftEndpoint(%v, %v)", tt.args.hostDomain, tt.args.remoteDomain)
			assert.Equalf(t, tt.wantRemoteQuota, gotRemoteQuota, "getLeftEndpoint(%v, %v)", tt.args.hostDomain, tt.args.remoteDomain)
		})
	}
}
