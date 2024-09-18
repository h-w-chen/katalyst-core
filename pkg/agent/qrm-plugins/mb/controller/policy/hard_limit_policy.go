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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/numapackage"
)

func getHardAlloc(unit numapackage.MBUnit, ownUSage, allHiUsages, allLowUSages, totalAllocatable int) int {
	if unit.GetTaskType() == numapackage.TaskTypeSOCKET {
		return prorateAlloc(ownUSage, allLowUSages+allHiUsages, totalAllocatable) + prorateAlloc(ownUSage, allHiUsages, SocketLoungeMB)
	}
	return prorateAlloc(ownUSage, allLowUSages+allHiUsages, totalAllocatable)
}

// getHardAllocs distributes non-reserved bandwidth to given group of units in proportion to their current usages
func getHardAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error) {
	hiMB, loMB := getHiLoGroupMBs(units, mbMonitor)

	// mbHiReserved shall be always left for SOCKETs if any
	if hiMB > 0 {
		mb -= mbHiReserved
	}

	result := make([]MBUnitAlloc, len(units))
	for i, u := range units {
		mbCurr := getUnitMB(u, mbMonitor)
		result[i] = MBUnitAlloc{
			Unit:         u,
			MBUpperBound: getHardAlloc(u, mbCurr, hiMB, loMB, mb),
		}
	}

	return result, nil
}

func getReserveAllocs(unitToReserves []numapackage.MBUnit) ([]MBUnitAlloc, error) {
	results := make([]MBUnitAlloc, len(unitToReserves))
	for i, u := range unitToReserves {
		results[i] = MBUnitAlloc{
			Unit:         u,
			MBUpperBound: SOCKETReverseMBPerNode * len(u.GetNUMANodes()),
		}
	}

	return results, nil
}

func divideGroupIntoReserveOrOthers(units []numapackage.MBUnit) (toReserves, others []numapackage.MBUnit) {
	for _, u := range units {
		if u.GetLifeCyclePhase() == numapackage.UnitPhaseAdmitted && u.GetTaskType() == numapackage.TaskTypeSOCKET ||
			u.GetLifeCyclePhase() == numapackage.UnitPhaseReserved {
			toReserves = append(toReserves, u)
			continue
		}

		others = append(others, u)
	}
	return
}

func calcPreemptAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error) {
	unitToReserves, unitOthers := divideGroupIntoReserveOrOthers(units)
	allocToReserves, err := getReserveAllocs(unitToReserves)
	if err != nil {
		return nil, err
	}

	var mbToReserve int
	for _, u := range unitToReserves {
		mbToReserve += len(u.GetNUMANodes()) * SOCKETReverseMBPerNode
	}

	mbToAllocate := mb - mbToReserve
	allocToHardLimits, err := getHardAllocs(unitOthers, mbToAllocate, mbHiReserved, mbMonitor)
	if err != nil {
		return nil, err
	}

	return append(allocToHardLimits, allocToReserves...), nil
}
