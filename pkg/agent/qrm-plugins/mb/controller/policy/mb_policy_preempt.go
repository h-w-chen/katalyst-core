/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policy

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/util"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type preemptPolicy struct {
	boundedMBPolicy BoundedMBPolicy
}

func getReservationPlan(domain *mbdomain.MBDomain, total int, preemptingNodes []int) *plan.MBAlloc {
	if len(preemptingNodes) == 0 {
		return nil
	}

	ccds := make([]int, 0)
	for _, node := range preemptingNodes {
		ccds = append(ccds, domain.NodeCCDs[node]...)
	}

	ccdAverage := total / len(ccds)
	ccdMB := make(map[int]int)
	for _, ccd := range ccds {
		ccdMB[ccd] = ccdAverage
	}

	return &plan.MBAlloc{
		Plan: map[task.QoSLevel]map[int]int{
			task.QoSLevelDedicatedCores: ccdMB,
		},
	}
}

func (p preemptPolicy) GetPlan(domain *mbdomain.MBDomain, currQoSMB map[task.QoSLevel]map[int]int) *plan.MBAlloc {
	preemptingNodes := domain.GetPreemptingNodes()
	mbToServe := mbdomain.ReservedPerNuma * len(preemptingNodes)
	reservationPlan := getReservationPlan(domain, mbToServe, preemptingNodes)
	general.InfofV(6, "mbm: domain %d hard reservation mb plan: %v", domain.ID, reservationPlan)

	mbFree := mbdomain.DomainTotalMB - mbToServe - util.Sum(currQoSMB)
	mbDedicated := util.SumCCDMB(currQoSMB[task.QoSLevelDedicatedCores])
	if mbDedicated > 0 { // if no dedicated pod, all free bandwidth shall be allocated for all (non-dedicated)
		mbFree -= mbdomain.LoungeMB
	}

	nonReservePlan := p.getNonReservePlan(mbFree, currQoSMB)
	general.InfofV(6, "mbm: domain %d non-reservation qos mb plan: %v", domain.ID, nonReservePlan)

	return plan.Merge(reservationPlan, nonReservePlan)
}

func (p preemptPolicy) getNonReservePlan(mbFree int, currQoSMB map[task.QoSLevel]map[int]int) *plan.MBAlloc {
	switch {
	case mbFree > 0:
		return p.getNonReservePlanToIncrease(mbFree, currQoSMB)
	case mbFree < 0:
		return p.getNonReservePlanToDecrease(mbFree, currQoSMB)
	default:
		return nil
	}
}

func (p preemptPolicy) getNonReservePlanToIncrease(mbFree int, currQoSMB map[task.QoSLevel]map[int]int) *plan.MBAlloc {
	// we treat dedicated qos differently from others - dedicated pods have a so-called "lounge" privileged zone
	// which is already excluded from the regular free;
	// the regular free (which is shared among all) shall be split to carious qos-ccd based on their current usages
	mbDedicated := util.SumCCDMB(currQoSMB[task.QoSLevelDedicatedCores])
	mbOthers := util.SumCCDMB(currQoSMB[task.QoSLevelSharedCores]) +
		util.SumCCDMB(currQoSMB[task.QoSLevelReclaimedCores]) + util.SumCCDMB(currQoSMB[task.QoSLevelSystemCores])
	distributions := util.CoefficientWeightedSplit(mbFree, []int{mbDedicated, mbOthers}, []int{1, 1})

	// ensure dedicated qos won't exceed its max
	mbIncreasableDedicated := util.GetMaxDedicatedToIncrease(currQoSMB[task.QoSLevelDedicatedCores])
	if mbIncreasableDedicated > distributions[0]+mbdomain.LoungeMB {
		mbIncreasableDedicated = distributions[0] + mbdomain.LoungeMB
	}

	mbFreeOthers := mbFree + mbdomain.LoungeMB - mbIncreasableDedicated

	var planDedicated *plan.MBAlloc
	if mbIncreasableDedicated > 0 {
		dedicatedMB := map[task.QoSLevel]map[int]int{
			task.QoSLevelDedicatedCores: currQoSMB[task.QoSLevelDedicatedCores],
		}
		planDedicated = p.boundedMBPolicy.GetPlan(mbIncreasableDedicated, dedicatedMB)
	}

	otherMB := map[task.QoSLevel]map[int]int{}
	for qos, ccdMB := range currQoSMB {
		if qos == task.QoSLevelDedicatedCores {
			continue
		}
		otherMB[qos] = ccdMB
	}
	planOthers := p.boundedMBPolicy.GetPlan(mbFreeOthers, otherMB)

	return plan.Merge(planDedicated, planOthers)
}

func (p preemptPolicy) getNonReservePlanToDecrease(mbFree int, currQoSMB map[task.QoSLevel]map[int]int) *plan.MBAlloc {
	// always throttle non-dedicated qos groups to spare -mbFree MB
	sharedMB := util.SumCCDMB(currQoSMB[task.QoSLevelSharedCores])
	reclaimedMB := util.SumCCDMB(currQoSMB[task.QoSLevelReclaimedCores])
	systemMB := util.SumCCDMB(currQoSMB[task.QoSLevelSystemCores])

	var dedicatedPlan *plan.MBAlloc
	mbDedicated := util.SumCCDMB(currQoSMB[task.QoSLevelDedicatedCores])
	if mbDedicated > 0 {
		dedicatedPlan = p.boundedMBPolicy.GetPlan(mbdomain.LoungeMB, map[task.QoSLevel]map[int]int{
			task.QoSLevelDedicatedCores: currQoSMB[task.QoSLevelDedicatedCores],
		})
	}

	toThrottles := util.CoefficientWeightedSplit(mbFree, []int{sharedMB, reclaimedMB, systemMB}, []int{5, 1, 5})
	sharedPlan := p.boundedMBPolicy.GetPlan(toThrottles[0], map[task.QoSLevel]map[int]int{
		task.QoSLevelSharedCores: currQoSMB[task.QoSLevelSharedCores],
	})
	reclaimedPlan := p.boundedMBPolicy.GetPlan(toThrottles[1], map[task.QoSLevel]map[int]int{
		task.QoSLevelReclaimedCores: currQoSMB[task.QoSLevelReclaimedCores],
	})
	systemPlan := p.boundedMBPolicy.GetPlan(toThrottles[1], map[task.QoSLevel]map[int]int{
		task.QoSLevelSystemCores: currQoSMB[task.QoSLevelSystemCores],
	})

	_ = plan.Merge(dedicatedPlan, sharedPlan, reclaimedPlan, systemPlan)
	panic("impl")
}
