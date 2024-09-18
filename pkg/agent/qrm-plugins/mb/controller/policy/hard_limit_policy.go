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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/apppool"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

func getHardAlloc(unit apppool.AppPool, ownUSage, allHiUsages, allLowUSages, totalAllocatable int) int {
	if unit.GetTaskType() == apppool.TaskTypeSOCKET {
		return ProrateAlloc(ownUSage, allLowUSages+allHiUsages, totalAllocatable) + ProrateAlloc(ownUSage, allHiUsages, SocketLoungeMB)
	}
	return ProrateAlloc(ownUSage, allLowUSages+allHiUsages, totalAllocatable)
}

// getHardAllocs distributes non-reserved bandwidth to given group of units in proportion to their current usages
func getHardAllocs(units []apppool.AppPool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error) {
	hiMB, loMB := getHiLoGroupMBs(units, mbMonitor)

	// mbHiReserved shall be always left for SOCKETs if any
	if hiMB > 0 {
		mb -= mbHiReserved
	}

	result := make([]MBAlloc, len(units))
	for i, u := range units {
		mbCurr := getUnitMB(u, mbMonitor)
		result[i] = MBAlloc{
			AppPool:      u,
			MBUpperBound: getHardAlloc(u, mbCurr, hiMB, loMB, mb),
		}
	}

	return result, nil
}

func getReserveAllocs(unitToReserves []apppool.AppPool) ([]MBAlloc, error) {
	results := make([]MBAlloc, len(unitToReserves))
	for i, u := range unitToReserves {
		results[i] = MBAlloc{
			AppPool:      u,
			MBUpperBound: SocketNodeReservedMB * len(u.GetNUMANodes()),
		}
	}

	return results, nil
}

func divideGroupIntoReserveOrOthers(units []apppool.AppPool) (toReserves, others []apppool.AppPool) {
	for _, u := range units {
		if u.GetLifeCyclePhase() == apppool.UnitPhaseAdmitted && u.GetTaskType() == apppool.TaskTypeSOCKET ||
			u.GetLifeCyclePhase() == apppool.UnitPhaseReserved {
			toReserves = append(toReserves, u)
			continue
		}

		others = append(others, u)
	}
	return
}

func calcPreemptAllocs(units []apppool.AppPool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error) {
	unitToReserves, unitOthers := divideGroupIntoReserveOrOthers(units)
	allocToReserves, err := getReserveAllocs(unitToReserves)
	if err != nil {
		return nil, err
	}

	var mbToReserve int
	for _, u := range unitToReserves {
		mbToReserve += len(u.GetNUMANodes()) * SocketNodeReservedMB
	}

	mbToAllocate := mb - mbToReserve
	allocToHardLimits, err := getHardAllocs(unitOthers, mbToAllocate, mbHiReserved, mbMonitor)
	if err != nil {
		return nil, err
	}

	return append(allocToHardLimits, allocToReserves...), nil
}
