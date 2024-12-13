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
		prevPower       int
		inDVFS          bool
	}
	type args struct {
		actualWatt  int
		desiredWatt int
		alert       spec.PowerAlert
		internalOp  spec.InternalOp
		ttl         time.Duration
	}
	var tests = []struct {
		name       string
		fields     fields
		args       args
		want       action.PowerAction
		wantInDVFS bool
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
			wantInDVFS: true,
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
			wantInDVFS: true,
		},
		{
			name: "if no evictable and there is partial room, go for constrint dvfs",
			fields: fields{
				evictableProber: mockPorberFalse,
				dvfsUsed:        8,
			},
			args: args{
				actualWatt:  100,
				desiredWatt: 95,
				alert:       spec.PowerAlertP0,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpFreqCap,
				Arg: 98,
			},
			wantInDVFS: true,
		},
		{
			name: "if no evictable and there is no room, noop",
			fields: fields{
				evictableProber: mockPorberFalse,
				dvfsUsed:        10,
				inDVFS:          true,
			},
			args: args{
				actualWatt:  100,
				desiredWatt: 95,
				alert:       spec.PowerAlertP0,
			},
			want: action.PowerAction{
				Op:  spec.InternalOpNoop,
				Arg: 0,
			},
			wantInDVFS: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &evictFirstStrategy{
				coefficient:     tt.fields.coefficient,
				evictableProber: tt.fields.evictableProber,
				dvfsTracker:     dvfsTracker{dvfsAccumEffect: tt.fields.dvfsUsed},
			}
			if got := e.RecommendAction(tt.args.actualWatt, tt.args.desiredWatt, tt.args.alert, tt.args.internalOp, tt.args.ttl); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RecommendAction() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, tt.wantInDVFS, e.dvfsTracker.inDVFS)
			if tt.fields.evictableProber != nil {
				mock.AssertExpectationsForObjects(t, tt.fields.evictableProber)
			}
		})
	}

}

func Test_dvfsTracker_update(t *testing.T) {
	t.Parallel()
	type fields struct {
		dvfsUsed  int
		indvfs    bool
		prevPower int
	}
	type args struct {
		actualWatt  int
		desiredWatt int
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantDVFSTracker dvfsTracker
	}{
		{
			name: "not in dvfs, not to accumulate",
			fields: fields{
				dvfsUsed:  3,
				indvfs:    false,
				prevPower: 100,
			},
			args: args{
				actualWatt:  90,
				desiredWatt: 85,
			},
			wantDVFSTracker: dvfsTracker{
				dvfsAccumEffect: 3,
				inDVFS:          false,
				prevPower:       90,
			},
		},
		{
			name: "in dvfs, accumulate if lower power is observed",
			fields: fields{
				dvfsUsed:  3,
				indvfs:    true,
				prevPower: 100,
			},
			args: args{
				actualWatt:  90,
				desiredWatt: 85,
			},
			wantDVFSTracker: dvfsTracker{
				dvfsAccumEffect: 13,
				inDVFS:          true,
				prevPower:       90,
			},
		},
		{
			name: "in dvfs, not to accumulate if higher power is observed",
			fields: fields{
				dvfsUsed:  3,
				indvfs:    true,
				prevPower: 100,
			},
			args: args{
				actualWatt:  101,
				desiredWatt: 85,
			},
			wantDVFSTracker: dvfsTracker{
				dvfsAccumEffect: 3,
				inDVFS:          true,
				prevPower:       101,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &dvfsTracker{
				dvfsAccumEffect: tt.fields.dvfsUsed,
				inDVFS:          tt.fields.indvfs,
				prevPower:       tt.fields.prevPower,
			}
			d.update(tt.args.actualWatt, tt.args.desiredWatt)
			assert.Equal(t, &tt.wantDVFSTracker, d)
		})
	}
}
