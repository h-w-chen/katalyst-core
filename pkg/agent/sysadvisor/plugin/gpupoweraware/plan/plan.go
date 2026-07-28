package plan

import (
	"math"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	defaultKp                      = 0.02
	defaultMaxTargetReductionRatio = 0.1
)

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

type safetyNetPlanner struct {
	innerPlanner            Planner
	alertBaselinePower      int
	maxTargetReductionRatio float64
}

func (s *safetyNetPlanner) GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (*PowerPlan, error) {
	if hasNoPowerAlert(spec) {
		s.alertBaselinePower = 0
	} else {
		s.updateAlertBaselinePower(currTotalPower)
	}

	powerPlan, err := s.innerPlanner.GetPlan(spec, levelHint, currTotalPower)
	if err != nil {
		return nil, err
	}

	return s.applyThrottleSafetyNet(powerPlan, currTotalPower), nil
}

func (s *safetyNetPlanner) updateAlertBaselinePower(currTotalPower int) {
	if currTotalPower <= 0 {
		return
	}
	if s.alertBaselinePower == 0 || currTotalPower > s.alertBaselinePower {
		s.alertBaselinePower = currTotalPower
	}
}

func (s *safetyNetPlanner) applyThrottleSafetyNet(powerPlan *PowerPlan, currTotalPower int) *PowerPlan {
	if powerPlan == nil || powerPlan.Op != OpCap || s.alertBaselinePower <= 0 {
		return powerPlan
	}
	if s.maxTargetReductionRatio <= 0 {
		return powerPlan
	}

	minTarget := int(math.Ceil(float64(s.alertBaselinePower) * (1 - s.maxTargetReductionRatio)))
	if minTarget <= 0 || powerPlan.Target >= minTarget {
		return powerPlan
	}

	if currTotalPower <= minTarget {
		general.Warningf("pap-gpu: planner: skip further throttle because current power %d is already below safety floor %d, baseline %d",
			currTotalPower, minTarget, s.alertBaselinePower)
		return nil
	}

	general.Warningf("pap-gpu: planner: clamp throttle target from %d to safety floor %d, baseline %d",
		powerPlan.Target, minTarget, s.alertBaselinePower)
	powerPlan.Target = minTarget
	return powerPlan
}

func New() Planner {
	return &safetyNetPlanner{
		innerPlanner: &persistentPlanner{
			innerPlanner: &linearPlanner{
				kp: defaultKp,
			},
		},
		maxTargetReductionRatio: defaultMaxTargetReductionRatio,
	}
}
