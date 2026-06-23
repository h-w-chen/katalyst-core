package plan

import "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"

type PowerPlan struct {
	Op     string
	Level  string
	Target int
}

type Planner interface {
	GetPlan(spec *spec.PowerSpec, level string, currTotalPower int) (PowerPlan, error)
}

type linearPlanner struct {
	kp float32
}

func (l linearPlanner) GetPlan(spec *spec.PowerSpec, level string, currTotalPower int) (PowerPlan, error) {
	//TODO implement me
	panic("implement me")
}

func New() Planner {
	return &linearPlanner{
		kp: 0.02,
	}
}
