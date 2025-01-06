package domaintarget

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/strategy/ccdtarget"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor/stat"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
)

func Test_extremeThrottlePlanner_GetPlan(t *testing.T) {
	t.Parallel()
	type fields struct {
		ccdGroupPlanner ccdtarget.CCDMBPlanner
	}
	type args struct {
		capacity    int
		mbQoSGroups map[qosgroup.QoSGroup]*stat.MBQoSGroup
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *plan.MBAlloc
	}{
		{
			name: "extreme throttling always get plan of min mb",
			fields: fields{
				ccdGroupPlanner: &ccdtarget.CCDGroupPlanner{
					CCDMBMin: 4_000,
					CCDMBMax: 35_000,
				},
			},
			args: args{
				capacity: 30_000,
				mbQoSGroups: map[qosgroup.QoSGroup]*stat.MBQoSGroup{
					"shared-30": {
						CCDs: sets.Int{0: sets.Empty{}, 1: sets.Empty{}},
						CCDMB: map[int]*stat.MBData{
							0: {TotalMB: 2_000},
							1: {TotalMB: 10_000},
						},
					},
				},
			},
			want: &plan.MBAlloc{Plan: map[qosgroup.QoSGroup]map[int]int{
				"shared-30": {0: 4_000, 1: 4_000},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := extremeThrottlePlanner{
				ccdGroupPlanner: tt.fields.ccdGroupPlanner,
			}
			if got := e.GetPlan(tt.args.capacity, tt.args.mbQoSGroups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}
