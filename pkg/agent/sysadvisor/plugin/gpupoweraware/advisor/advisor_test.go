package advisor

import (
	"context"
	"testing"

	gpucapper "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
)

type fakeSpecFetcher struct {
	powerSpec *spec.PowerSpec
	err       error
}

func (f *fakeSpecFetcher) GetPowerSpec(context.Context) (*spec.PowerSpec, error) {
	return f.powerSpec, f.err
}

type fakePowerReader struct {
	totalPower int
	err        error
}

func (f *fakePowerReader) Start() error {
	return nil
}

func (f *fakePowerReader) GetTotalPower(context.Context) (int, error) {
	return f.totalPower, f.err
}

type fakePlanner struct {
	powerPlan     *plan.PowerPlan
	err           error
	levelHint     gpucapper.Level
	levelHintSeen bool
}

func (f *fakePlanner) GetPlan(_ *spec.PowerSpec, levelHint gpucapper.Level, _ int) (*plan.PowerPlan, error) {
	f.levelHint = levelHint
	f.levelHintSeen = true
	return f.powerPlan, f.err
}

type fakePowerCapper struct {
	resetCalled        bool
	capWithLevelCalled bool
	level              gpucapper.Level
	targetWatts        int
	currWatt           int
}

func (f *fakePowerCapper) Init() error {
	return nil
}

func (f *fakePowerCapper) Start() error {
	return nil
}

func (f *fakePowerCapper) Stop() error {
	return nil
}

func (f *fakePowerCapper) Reset() {
	f.resetCalled = true
}

func (f *fakePowerCapper) Cap(context.Context, int, int) {}

func (f *fakePowerCapper) CapWithLevel(_ context.Context, level gpucapper.Level, targetWatts, currWatt int) {
	f.capWithLevelCalled = true
	f.level = level
	f.targetWatts = targetWatts
	f.currWatt = currWatt
}

func TestGPUAdvisorRunOnceSkipsNilPlan(t *testing.T) {
	t.Parallel()

	powerCapper := &fakePowerCapper{}
	advisor := &gpuAdvisor{
		specFetcher: &fakeSpecFetcher{powerSpec: &spec.PowerSpec{Alert: spec.PowerAlertOK}},
		powerReader: &fakePowerReader{totalPower: 943},
		planner:     &fakePlanner{},
		capper:      powerCapper,
	}

	advisor.runOnce(context.Background())

	if powerCapper.resetCalled {
		t.Fatal("reset should not be called for nil plan")
	}
	if powerCapper.capWithLevelCalled {
		t.Fatal("cap should not be called for nil plan")
	}
}

func TestGPUAdvisorRunOnceAppliesResetPlan(t *testing.T) {
	t.Parallel()

	powerCapper := &fakePowerCapper{}
	advisor := &gpuAdvisor{
		specFetcher: &fakeSpecFetcher{powerSpec: &spec.PowerSpec{Alert: spec.PowerAlertOK}},
		powerReader: &fakePowerReader{totalPower: 943},
		planner: &fakePlanner{powerPlan: &plan.PowerPlan{
			Op:    plan.OpReset,
			Level: gpucapper.LevelAll,
		}},
		capper: powerCapper,
	}

	advisor.runOnce(context.Background())

	if !powerCapper.resetCalled {
		t.Fatal("reset should be called for reset plan")
	}
	if powerCapper.capWithLevelCalled {
		t.Fatal("cap should not be called for reset plan")
	}
}

func TestGPUAdvisorRunOnceAppliesCapPlan(t *testing.T) {
	t.Parallel()

	powerCapper := &fakePowerCapper{}
	advisor := &gpuAdvisor{
		specFetcher: &fakeSpecFetcher{powerSpec: &spec.PowerSpec{Alert: spec.PowerAlertP0, Budget: 900}},
		powerReader: &fakePowerReader{totalPower: 943},
		planner: &fakePlanner{powerPlan: &plan.PowerPlan{
			Op:     plan.OpCap,
			Level:  gpucapper.LevelDecode,
			Target: 942,
		}},
		capper: powerCapper,
	}

	advisor.runOnce(context.Background())

	if powerCapper.resetCalled {
		t.Fatal("reset should not be called for cap plan")
	}
	if !powerCapper.capWithLevelCalled {
		t.Fatal("cap should be called for cap plan")
	}
	if powerCapper.level != gpucapper.LevelDecode {
		t.Fatalf("unexpected level: got %s, want %s", powerCapper.level, gpucapper.LevelDecode)
	}
	if powerCapper.targetWatts != 942 {
		t.Fatalf("unexpected target watts: got %d, want 942", powerCapper.targetWatts)
	}
	if powerCapper.currWatt != 943 {
		t.Fatalf("unexpected current watts: got %d, want 943", powerCapper.currWatt)
	}
}

func TestGPUAdvisorRunOncePassesLevelHintByAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		alert     spec.PowerAlert
		wantLevel gpucapper.Level
	}{
		{name: "s0 uses all", alert: spec.PowerAlertS0, wantLevel: gpucapper.LevelAll},
		{name: "p0 uses all", alert: spec.PowerAlertP0, wantLevel: gpucapper.LevelAll},
		{name: "p1 uses all", alert: spec.PowerAlertP1, wantLevel: gpucapper.LevelAll},
		{name: "p2 uses decode", alert: spec.PowerAlertP2, wantLevel: gpucapper.LevelDecode},
		{name: "p3 uses decode", alert: powerAlertP3, wantLevel: gpucapper.LevelDecode},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			planner := &fakePlanner{}
			advisor := &gpuAdvisor{
				specFetcher: &fakeSpecFetcher{powerSpec: &spec.PowerSpec{Alert: tt.alert, Budget: 900}},
				powerReader: &fakePowerReader{totalPower: 943},
				planner:     planner,
				capper:      &fakePowerCapper{},
			}

			advisor.runOnce(context.Background())

			if !planner.levelHintSeen {
				t.Fatal("planner should be called")
			}
			if planner.levelHint != tt.wantLevel {
				t.Fatalf("unexpected level hint: got %s, want %s", planner.levelHint, tt.wantLevel)
			}
		})
	}
}
