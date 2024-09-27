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

package qospolicy

import (
	"fmt"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/util"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

type weightedQoSMBPolicy struct{}

func (w *weightedQoSMBPolicy) GetPlan(totalMB int, qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup, isTopTier bool) *plan.MBAlloc {
	if isTopTier {
		return w.getTopLevelPlan(totalMB, qosGroupMBs)
	}

	return w.getProportionalPlan(totalMB, qosGroupMBs)
}

func (w *weightedQoSMBPolicy) getProportionalPlan(totalMB int, qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup) *plan.MBAlloc {
	totalUsage := monitor.SumMB(qosGroupMBs)
	if totalUsage >= totalMB {
		return w.getProportionalPlanToEnlarge(totalMB, qosGroupMBs)
	}

	return w.getProportionalPlanToShrink(totalMB, qosGroupMBs)
}

func (w *weightedQoSMBPolicy) getProportionalPlanToEnlarge(totalMB int, qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup) *plan.MBAlloc {
	// enlarging is simple - all participants have identical weights of usage for the gain share
	totalUsage := monitor.SumMB(qosGroupMBs)
	ratio := float64(totalMB) / float64(totalUsage)
	return getOriginalAlloc(qosGroupMBs, ratio)
}

func (w *weightedQoSMBPolicy) getProportionalPlanToShrink(totalMB int, qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup) *plan.MBAlloc {
	totalUsage := monitor.SumMB(qosGroupMBs)
	// loss is negative
	loss := totalMB - totalUsage
	lossAlloc := w.getProportionalLossPlan(loss, qosGroupMBs)

	origAlloc := getOriginalAlloc(qosGroupMBs, 1.0)
	return plan.Merge(origAlloc, lossAlloc)
}

func getOriginalAlloc(qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup, ratio float64) *plan.MBAlloc {
	mbPlan := &plan.MBAlloc{Plan: make(map[task.QoSLevel]map[int]int)}
	for qos, groupMB := range qosGroupMBs {
		if len(groupMB.WeightedCCDMBs) == 0 {
			for ccd, mb := range groupMB.CCDMB {
				if _, ok := mbPlan.Plan[qos]; !ok {
					mbPlan.Plan[qos] = make(map[int]int)
				}
				mbPlan.Plan[qos][ccd] = int(ratio * float64(mb))
			}
		} else {
			for weight, ccdmb := range groupMB.WeightedCCDMBs {
				for ccd, mb := range ccdmb {
					qosSub := task.QoSLevel(fmt.Sprintf("%s_%d", qos, weight))
					if _, ok := mbPlan.Plan[qosSub]; !ok {
						mbPlan.Plan[qosSub] = make(map[int]int)
					}
					mbPlan.Plan[qosSub][ccd] = int(ratio * float64(mb))
				}
			}
		}
	}
	return mbPlan
}

func calcEffictiveTotal(qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup) float64 {
	var result float64
	for _, group := range qosGroupMBs {
		if len(group.WeightedCCDMBs) == 0 {
			result += float64(util.SumCCDMB(group.CCDMB))
		} else {
			for w, ccdmb := range group.WeightedCCDMBs {
				result += float64(w * util.SumCCDMB(ccdmb) / 100)
			}
		}
	}
	return result
}

// todo: revise the weight logic; the current one is not right: the higher the weight, the more loss.
func (w *weightedQoSMBPolicy) getProportionalLossPlan(loss int, qosGroupMBs map[task.QoSLevel]*monitor.MBQoSGroup) *plan.MBAlloc {
	// shrinking is a bit complex - participant may have different weight of usage for the loss share
	effectiveTotal := calcEffictiveTotal(qosGroupMBs)
	ratio := float64(loss) / effectiveTotal

	mbPlan := &plan.MBAlloc{Plan: make(map[task.QoSLevel]map[int]int)}
	for qos, groupMB := range qosGroupMBs {
		if len(groupMB.WeightedCCDMBs) == 0 {
			for ccd, mb := range groupMB.CCDMB {
				if _, ok := mbPlan.Plan[qos]; !ok {
					mbPlan.Plan[qos] = make(map[int]int)
				}
				mbPlan.Plan[qos][ccd] = int(ratio * float64(mb))
			}
		} else {
			for weight, ccdmb := range groupMB.WeightedCCDMBs {
				for ccd, mb := range ccdmb {
					qosSub := task.QoSLevel(fmt.Sprintf("%s_%d", qos, weight))
					if _, ok := mbPlan.Plan[qosSub]; !ok {
						mbPlan.Plan[qosSub] = make(map[int]int)
					}
					mbPlan.Plan[qosSub][ccd] = int(ratio * float64(mb) / 100 * float64(weight))
				}
			}
		}
	}

	return mbPlan
}

func (w *weightedQoSMBPolicy) getTopLevelPlan(totalMB int, qosGroups map[task.QoSLevel]*monitor.MBQoSGroup) *plan.MBAlloc {
	// don't set throttling at all for top level QoS group's CCDs; instead allow more than the totalMB
	mbPlan := &plan.MBAlloc{Plan: make(map[task.QoSLevel]map[int]int)}
	for qos, group := range qosGroups {
		for ccd, mb := range group.CCDMB {
			if mb > 0 {
				if _, ok := mbPlan.Plan[qos]; !ok {
					mbPlan.Plan[qos] = make(map[int]int)
				}
				mbPlan.Plan[qos][ccd] = mbdomain.MaxMBPerCCD
			}
		}
	}

	return mbPlan
}

func NewWeightedQoSMBPolicy() QoSMBPolicy {
	return &weightedQoSMBPolicy{}
}
