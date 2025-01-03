package domaintarget

import (
	policyconfig "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/config"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/strategy/ccdtarget"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor/stat"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
)

type fullEasePlanner struct {
	ccdGroupPlanner *ccdtarget.CCDGroupPlanner
}

func (t fullEasePlanner) GetQuota(capacity, currentUsage int) int {
	allocatable := capacity - policyconfig.PolicyConfig.MBEaseThreshold
	if allocatable <= 0 {
		return 0
	}
	return allocatable
}

func (t fullEasePlanner) Name() string {
	return "full ease planner"
}

func (t fullEasePlanner) GetPlan(capacity int, mbQoSGroups map[qosgroup.QoSGroup]*stat.MBQoSGroup) *plan.MBAlloc {
	allocatable := capacity - policyconfig.PolicyConfig.MBEaseThreshold
	if allocatable <= 0 {
		return nil
	}

	// distribute total among all proportionally
	totalUsage := stat.SumMB(mbQoSGroups)
	ratio := float64(allocatable) / float64(totalUsage)
	return t.ccdGroupPlanner.GetProportionalPlan(ratio, mbQoSGroups)
}

func newFullEasePlanner(planner *ccdtarget.CCDGroupPlanner) DomainMBAdjuster {
	return &fullEasePlanner{
		ccdGroupPlanner: planner,
	}
}

type halfEasePlanner struct {
	innerPlanner fullEasePlanner
}

func (s halfEasePlanner) GetQuota(capacity, currentUsage int) int {
	constraintCapacity := (capacity + policyconfig.PolicyConfig.MBEaseThreshold + currentUsage) / 2
	return s.innerPlanner.GetQuota(constraintCapacity, currentUsage)
}

func (s halfEasePlanner) Name() string {
	return "half ease planner"
}

func (s halfEasePlanner) GetPlan(capacity int, mbQoSGroups map[qosgroup.QoSGroup]*stat.MBQoSGroup) *plan.MBAlloc {
	totalUsage := stat.SumMB(mbQoSGroups)
	// step ease planner eases 1/2 newly allocatable only at ease step
	constraintCapacity := (capacity + policyconfig.PolicyConfig.MBEaseThreshold + totalUsage) / 2
	return s.innerPlanner.GetPlan(constraintCapacity, mbQoSGroups)
}

func newHalfEasePlanner(planner *ccdtarget.CCDGroupPlanner) DomainMBAdjuster {
	return &halfEasePlanner{
		innerPlanner: fullEasePlanner{ccdGroupPlanner: planner},
	}
}
