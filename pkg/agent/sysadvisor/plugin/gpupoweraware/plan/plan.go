package plan

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const defaultKp = 0.02

const (
	OpCap   = "cap"
	OpReset = "reset"
)

type PowerPlan struct {
	Op     string
	Level  capper.Level
	Target int
}

type Planner interface {
	GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (*PowerPlan, error)
}

type linearPlanner struct {
	kp float64
}

func (l *linearPlanner) GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (*PowerPlan, error) {
	if hasNoPowerAlert(spec) {
		return l.getResetPlan(), nil
	}

	if spec.Budget < currTotalPower {
		return l.getThrottlePlan(spec.Budget, currTotalPower, levelHint), nil
	}

	general.InfofV(6, "already below budget, no need to throttle")
	return nil, nil
}

func hasNoPowerAlert(powerSpec *spec.PowerSpec) bool {
	return powerSpec == nil || len(powerSpec.Alert) == 0 || powerSpec.Alert == spec.PowerAlertOK
}

func (l *linearPlanner) getThrottlePlan(budget, current int, levelHint capper.Level) *PowerPlan {
	gap := current - budget
	toDecrease := int(float64(gap) * l.kp)
	if toDecrease == 0 {
		toDecrease = 1
	}

	target := current - toDecrease
	return &PowerPlan{
		Op:     OpCap,
		Level:  levelHint,
		Target: target,
	}
}

func (l *linearPlanner) getResetPlan() *PowerPlan {
	return &PowerPlan{
		Op:     OpReset,
		Level:  capper.LevelAll,
		Target: 0,
	}
}

type persistentPlanner struct {
	innerPlanner Planner
	priorPlan    *PowerPlan
}

func (p *persistentPlanner) GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (*PowerPlan, error) {
	if hasNoPowerAlert(spec) {
		if p.priorPlan != nil && p.priorPlan.Op == OpReset {
			return nil, nil
		}
	}

	powerPlan, err := p.innerPlanner.GetPlan(spec, levelHint, currTotalPower)
	if err == nil {
		p.priorPlan = powerPlan
	}
	return powerPlan, err
}

func New() Planner {
	return &persistentPlanner{
		innerPlanner: &linearPlanner{
			kp: defaultKp,
		},
	}
}
