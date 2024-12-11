package strategy

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor/action"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
)

type mockEvicableProber struct {
	mock.Mock
}

func (m *mockEvicableProber) HasEvictablePods() bool {
	args := m.Called()
	return args.Bool(0)
}

func Test_evictFirstStrategy_RecommendAction(t *testing.T) {
	t.Parallel()

	mockPorberTrue := new(mockEvicableProber)
	mockPorberTrue.On("HasEvictablePods").Return(true)

	mockPorberFalse := new(mockEvicableProber)
	mockPorberFalse.On("HasEvictablePods").Return(false)

	type fields struct {
		coefficient     exponentialDecay
		evictableProber EvictableProber
		dvfsUsed        int
	}
	type args struct {
		actualWatt  int
		desiredWatt int
		alert       spec.PowerAlert
		internalOp  spec.InternalOp
		ttl         time.Duration
	}
	var tests = []struct {
		name         string
		fields       fields
		args         args
		want         action.PowerAction
		wantDVFSUsed int
	}{
		{
			name: "internal op is always respected if exists",
			fields: fields{
				coefficient:     exponentialDecay{},
				evictableProber: nil,
				dvfsUsed:        0,
			},
			args: args{
				actualWatt:  100,
				desiredWatt: 80,
				internalOp:  spec.InternalOpFreqCap,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpFreqCap,
				Arg: 80,
			},
			wantDVFSUsed: 20,
		},
		{
			name:   "p2 is noop",
			fields: fields{},
			args: args{
				actualWatt:  100,
				desiredWatt: 80,
				alert:       spec.PowerAlertP2,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpNoop,
				Arg: 0,
			},
		},
		{
			name: "evict first if possible",
			fields: fields{
				coefficient:     exponentialDecay{b: defaultDecayB},
				evictableProber: mockPorberTrue,
				dvfsUsed:        0,
			},
			args: args{
				actualWatt:  100,
				desiredWatt: 80,
				alert:       spec.PowerAlertP0,
				ttl:         time.Minute * 15,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpEvict,
				Arg: 14,
			},
			wantDVFSUsed: 0,
		},
		{
			name: "if no evictable and there is room, go for dvfs",
			fields: fields{
				evictableProber: mockPorberFalse,
				dvfsUsed:        0,
			},
			args: args{
				actualWatt:  100,
				desiredWatt: 95,
				alert:       spec.PowerAlertP0,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpFreqCap,
				Arg: 95,
			},
			wantDVFSUsed: 5,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &evictFirstStrategy{
				coefficient:     tt.fields.coefficient,
				evictableProber: tt.fields.evictableProber,
				dvfsUsed:        tt.fields.dvfsUsed,
			}
			if got := e.RecommendAction(tt.args.actualWatt, tt.args.desiredWatt, tt.args.alert, tt.args.internalOp, tt.args.ttl); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RecommendAction() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, tt.wantDVFSUsed, e.dvfsUsed)
			if tt.fields.evictableProber != nil {
				mock.AssertExpectationsForObjects(t, tt.fields.evictableProber)
			}
		})
	}

}
