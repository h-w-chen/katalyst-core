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

package combined

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/domain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/plan"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

// EnhancedAdvisor is the advisor that treats resctrl groups of identical priority as if a logical group distributing
// the ccd mb quotas among the real groups. It targets the scenarios where the groups of same priority don't share ccd.
type EnhancedAdvisor struct {
	inner advisor.Advisor
}

func (d *EnhancedAdvisor) GetPlan(ctx context.Context, domainsMon *monitor.DomainStats) (*plan.MBPlan, error) {
	domainStats, groupInfos, err := d.combineDomainStats(domainsMon)
	if err != nil {
		return nil, err
	}
	mbPlan, err := d.inner.GetPlan(ctx, domainStats)
	if err != nil {
		return nil, err
	}
	return d.splitPlan(mbPlan, groupInfos), nil
}

func (d *EnhancedAdvisor) combineDomainStats(domainsMon *monitor.DomainStats) (*monitor.DomainStats, *monitor.GroupInfo, error) {
	combinedDomainStats := &monitor.DomainStats{
		Incomings:            make(map[int]monitor.DomainMonStat),
		Outgoings:            make(map[int]monitor.DomainMonStat),
		OutgoingGroupSumStat: make(map[string][]monitor.MBInfo),
	}
	combinedGroupInfo := &monitor.GroupInfo{
		DomainGroups: make(map[int]monitor.DomainGroupMapping),
	}
	var err error

	for id, domainMon := range domainsMon.Incomings {
		combinedDomainStats.Incomings[id], combinedGroupInfo.DomainGroups[id], err = preProcessGroupInfo(domainMon)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to preProcessGroupInfo for incoming domain %d: %w", id, err)
		}
	}

	for id, domainMon := range domainsMon.Outgoings {
		combinedDomainStats.Outgoings[id], _, err = preProcessGroupInfo(domainMon)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to preProcessGroupInfo for incoming domain %d: %w", id, err)
		}
	}

	combinedDomainStats.OutgoingGroupSumStat = preProcessGroupSumStat(domainsMon.OutgoingGroupSumStat)

	return combinedDomainStats, combinedGroupInfo, nil
}

func (d *EnhancedAdvisor) splitPlan(mbPlan *plan.MBPlan, groupInfos *monitor.GroupInfo) *plan.MBPlan {
	for groupKey, ccdPlan := range mbPlan.MBGroups {
		if !strings.Contains(groupKey, "combined-") {
			continue
		}
		for _, domainGroupInfos := range groupInfos.DomainGroups {
			for group, groupInfo := range domainGroupInfos[groupKey] {
				for ccd, weight := range groupInfo {
					if _, exists := ccdPlan[ccd]; exists {
						if mbPlan.MBGroups[group] == nil {
							mbPlan.MBGroups[group] = make(plan.GroupCCDPlan)
						}
						mbPlan.MBGroups[group][ccd] = int(float64(ccdPlan[ccd]) * weight)
					}
				}
			}
		}
		delete(mbPlan.MBGroups, groupKey)
	}
	return mbPlan
}

func NewEnhancedAdvisor(emitter metrics.MetricEmitter, domains domain.Domains, ccdMinMB, ccdMaxMB int, defaultDomainCapacity int,
	capPercent int, XDomGroups []string, groupNeverThrottles []string,
	groupCapacity map[string]int,
) advisor.Advisor {
	innerAdvisor := advisor.NewDomainAdvisor(emitter, domains,
		ccdMinMB, ccdMaxMB,
		defaultDomainCapacity, capPercent,
		XDomGroups, groupNeverThrottles,
		groupCapacity)
	return &EnhancedAdvisor{
		inner: innerAdvisor,
	}
}
