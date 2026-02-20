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
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/priority"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

// groupByWeight extracts the common logic of grouping by weight
func groupByWeight[T any](stats map[string]T) map[int][]string {
	groups := make(map[int][]string, len(stats))
	for group := range stats {
		weight := priority.GetInstance().GetWeight(group)
		groups[weight] = append(groups[weight], group)
	}
	return groups
}

func getCombinedGroups(mbStats monitor.GroupMBStats) map[string]sets.String {
	combinedGroups := make(map[string]sets.String)

	groups := groupByWeight(mbStats)
	for weight, equivGroups := range groups {
		if len(equivGroups) == 1 {
			continue
		}

		combinedGroupKey := fmt.Sprintf("combined-%d", weight)
		combinedGroups[combinedGroupKey] = sets.NewString(equivGroups...)
	}

	return combinedGroups
}

func isCombined(name string, combinedGroups map[string]sets.String) bool {
	for _, groups := range combinedGroups {
		if _, found := groups[name]; found {
			return true
		}
	}

	return false
}

func combineGroupMBStats(stats monitor.GroupMBStats, combinedGroups map[string]sets.String) monitor.GroupMBStats {
	combinedStats := make(monitor.GroupMBStats)

	// copy over the groups no need to combine
	for group, mbStat := range stats {
		if isCombined(group, combinedGroups) {
			continue
		}
		combinedStats[group] = mbStat
	}

	// combine the groups needed to combine
	// each ccd sums up all groups
	for combinedGroup, groups := range combinedGroups {
		combinedCCDMB := make(monitor.GroupMB)
		for group := range groups {
			groupMB, ok := stats[group]
			if !ok {
				continue
			}
			for ccd, mbInfo := range groupMB {
				if mbInfo.TotalMB > combinedCCDMB[ccd].TotalMB {
					combinedCCDMB[ccd] = mbInfo
				}
			}
		}
		combinedStats[combinedGroup] = combinedCCDMB
	}

	return combinedStats
}

func locateGroupCCDs(combinedGroupStats monitor.GroupMB, groups sets.String, groupStats monitor.GroupMBStats) map[string]sets.Int {
	groupCCDs := make(map[string]sets.Int)

	for ccd, mbInfo := range combinedGroupStats {
		var maxGroup string
		for group := range groups {
			if groupStats[group][ccd].TotalMB == mbInfo.TotalMB {
				maxGroup = group
			}
		}

		if len(maxGroup) == 0 {
			continue
		}
		if _, exist := groupCCDs[maxGroup]; !exist {
			groupCCDs[maxGroup] = make(sets.Int)
		}
		groupCCDs[maxGroup].Insert(ccd)
	}

	return groupCCDs
}

// preProcessGroupInfo combines groups with same priority together
func preProcessGroupInfo(stats monitor.GroupMBStats) (monitor.GroupMBStats, monitor.DomainGroupMapping, error) {
	combinedGroups := getCombinedGroups(stats)
	combinedStats := combineGroupMBStats(stats, combinedGroups)

	combinedGroupCCDs := monitor.DomainGroupMapping{}
	for combinedGroup, realGroups := range combinedGroups {
		combinedGroupStat := combinedStats[combinedGroup]
		combinedGroupCCDs[combinedGroup] = make(monitor.CombinedGroupMapping)
		for group, ccds := range locateGroupCCDs(combinedGroupStat, realGroups, stats) {
			combinedGroupCCDs[combinedGroup][group] = monitor.CCDSet(ccds)
		}
	}

	return combinedStats, combinedGroupCCDs, nil
}

func preProcessGroupSumStat(sumStats map[string][]monitor.MBInfo) map[string][]monitor.MBInfo {
	groups := groupByWeight(sumStats)

	result := make(map[string][]monitor.MBInfo)

	for weight, equivGroups := range groups {
		if len(equivGroups) == 1 {
			result[equivGroups[0]] = sumStats[equivGroups[0]]
			continue
		}

		newKey := fmt.Sprintf("combined-%d", weight)
		sumList := make([]monitor.MBInfo, len(sumStats[equivGroups[0]]))

		for _, group := range equivGroups {
			for id, stat := range sumStats[group] {
				sumList[id].LocalMB += stat.LocalMB
				sumList[id].RemoteMB += stat.RemoteMB
				sumList[id].TotalMB += stat.TotalMB
			}
		}
		result[newKey] = sumList
	}

	return result
}
