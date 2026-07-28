package plan

import (
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	powerspec "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
)

func TestLinearPlannerGetThrottlePlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		budget  int
		current int
		kp      float64
		want    int
	}{
		{
			name:    "decrease at least one watt for small gap",
			budget:  900,
			current: 943,
			kp:      defaultKp,
			want:    942,
		},
		{
			name:    "decrease proportionally for large gap",
			budget:  1000,
			current: 2000,
			kp:      defaultKp,
			want:    1980,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			planner := &linearPlanner{kp: tt.kp}
			got := planner.getThrottlePlan(tt.budget, tt.current, capper.LevelDecode)

			if got.Target != tt.want {
				t.Fatalf("unexpected target: got %d, want %d", got.Target, tt.want)
			}
			if got.Target >= tt.current {
				t.Fatalf("target should be below current: got target %d, current %d", got.Target, tt.current)
			}
			if got.Level != capper.LevelDecode {
				t.Fatalf("unexpected level: got %s, want %s", got.Level, capper.LevelDecode)
			}
		})
	}
}

func TestLinearPlannerGetPlanNoPowerAlert(t *testing.T) {
	t.Parallel()

	planner := &linearPlanner{kp: defaultKp}
	got, err := planner.GetPlan(&powerspec.PowerSpec{
		Alert:  powerspec.PowerAlertOK,
		Budget: 0,
	}, capper.LevelDecode, 943)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected reset plan, got nil")
	}
	if got.Op != "reset" {
		t.Fatalf("unexpected op: got %s, want reset", got.Op)
	}
	if got.Target != 0 {
		t.Fatalf("unexpected target: got %d, want 0", got.Target)
	}
	if got.Level != capper.LevelAll {
		t.Fatalf("unexpected level: got %s, want %s", got.Level, capper.LevelAll)
	}
}

func TestSafetyNetPlannerClampsThrottleTargetBySafetyFloor(t *testing.T) {
	t.Parallel()

	planner := &safetyNetPlanner{
		innerPlanner: &linearPlanner{
			kp: 1,
		},
		maxTargetReductionRatio: defaultMaxTargetReductionRatio,
	}

	got, err := planner.GetPlan(&powerspec.PowerSpec{
		Alert:  powerspec.PowerAlertP0,
		Budget: 100,
	}, capper.LevelAll, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected cap plan, got nil")
	}
	if got.Target != 900 {
		t.Fatalf("unexpected target: got %d, want 900", got.Target)
	}
}

func TestSafetyNetPlannerSkipsThrottleBelowSafetyFloor(t *testing.T) {
	t.Parallel()

	planner := &safetyNetPlanner{
		innerPlanner: &linearPlanner{
			kp: 1,
		},
		maxTargetReductionRatio: defaultMaxTargetReductionRatio,
	}

	_, err := planner.GetPlan(&powerspec.PowerSpec{
		Alert:  powerspec.PowerAlertP0,
		Budget: 100,
	}, capper.LevelAll, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := planner.GetPlan(&powerspec.PowerSpec{
		Alert:  powerspec.PowerAlertP0,
		Budget: 100,
	}, capper.LevelAll, 890)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != nil {
		t.Fatalf("expected no plan below safety floor, got %+v", got)
	}
}
