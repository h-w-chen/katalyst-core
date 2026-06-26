package plan

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
)

type PowerPlan struct {
	Op     string
	Level  capper.Level
	Target int
}

type Planner interface {
	GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (PowerPlan, error)
}

type linearPlanner struct {
	kp float32
}

func (l *linearPlanner) GetPlan(spec *spec.PowerSpec, levelHint capper.Level, currTotalPower int) (PowerPlan, error) {
	if spec == nil || len(spec.Alert) == 0 {
		return PowerPlan{
			Op:     "reset",
			Level:  capper.LevelAll,
			Target: 0,
		}, nil
	}

	delta := float32(spec.Budget - currTotalPower)
	target := currTotalPower + int(delta*l.kp)
	return PowerPlan{
		Op:     "target",
		Level:  levelHint,
		Target: target,
	}, nil
}

func New() Planner {
	return &linearPlanner{
		kp: 0.02,
	}
}
