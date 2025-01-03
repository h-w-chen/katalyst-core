package domaintarget

import (
	policyconfig "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/config"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/strategy/ccdtarget"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor/stat"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
)

// extremeThrottlePlanner implements the extreme throttling by generating the plan
// that set all groups to their min allocation
type extremeThrottlePlanner struct {
	ccdGroupPlanner *ccdtarget.CCDGroupPlanner
}

func (e extremeThrottlePlanner) GetQuota(capacity, currentUsage int) int {
	return 4_000
}

func (e extremeThrottlePlanner) Name() string {
	return "extreme throttle planner"
}

func (e extremeThrottlePlanner) GetPlan(capacity int, mbQoSGroups map[qosgroup.QoSGroup]*stat.MBQoSGroup) *plan.MBAlloc {
	return e.ccdGroupPlanner.GetFixedPlan(e.ccdGroupPlanner.CCDMBMin, mbQoSGroups)
}

func newExtremeThrottlePlanner(ccdPlanner *ccdtarget.CCDGroupPlanner) DomainMBAdjuster {
	return &extremeThrottlePlanner{
		ccdGroupPlanner: ccdPlanner,
	}
}

// halfThrottlePlanner forces qos groups to yield half of mb in use
type halfThrottlePlanner struct {
	ccdGroupPlanner *ccdtarget.CCDGroupPlanner
}

func (h halfThrottlePlanner) GetQuota(capacity, currentUsage int) int {
	allocatable := currentUsage / 2
	// summarized low prio qos plans should  not exceeding the ease bar
	if allocatable > capacity-policyconfig.PolicyConfig.EaseThreshold {
		allocatable = capacity - policyconfig.PolicyConfig.EaseThreshold
	}
	return allocatable
}

func (h halfThrottlePlanner) Name() string {
	return "half throttle planner"
}

func (h halfThrottlePlanner) GetPlan(capacity int, mbQoSGroups map[qosgroup.QoSGroup]*stat.MBQoSGroup) *plan.MBAlloc {
	totalUsage := stat.SumMB(mbQoSGroups)
	allocatable := h.GetQuota(capacity, totalUsage)

	// distribute total among all proportionally
	ratio := float64(allocatable) / float64(totalUsage)
	return h.ccdGroupPlanner.GetProportionalPlanWithUpperLimit(ratio, mbQoSGroups, capacity-policyconfig.PolicyConfig.EaseThreshold)
}

func newHalfThrottlePlanner(ccdPlanner *ccdtarget.CCDGroupPlanner) DomainMBAdjuster {
	return &halfThrottlePlanner{
		ccdGroupPlanner: ccdPlanner,
	}
}