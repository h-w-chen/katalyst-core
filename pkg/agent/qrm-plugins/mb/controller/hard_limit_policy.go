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

package controller

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
func getHardAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]mbAlloc, error) {
	hiMB, loMB := getHiLoGroupMBs(units, mbMonitor)

	// mbHiReserved shall be always left for SOCKETs if any
	if hiMB > 0 {
		mb -= mbHiReserved
	}

	result := make([]mbAlloc, len(units))
	for i, u := range units {
		mbCurr := getUnitMB(u, mbMonitor)
		result[i] = mbAlloc{
			unit:         u,
			mbUpperBound: getHardAlloc(u, mbCurr, hiMB, loMB, mb),
		}
	}

	return result, nil
}

func getReserveAllocs(unitToReserves []numapackage.MBUnit) ([]mbAlloc, error) {
	results := make([]mbAlloc, len(unitToReserves))
	for i, u := range unitToReserves {
		results[i] = mbAlloc{
			unit:         u,
			mbUpperBound: SOCKETReverseMBPerNode * len(u.GetNUMANodes()),
		}
	}

	return results, nil
}

func divideGroupIntoReserveOrOthers(p numapackage.MBPackage) (toReserves, others []numapackage.MBUnit) {
	for _, u := range p.GetUnits() {
		if u.GetLifeCyclePhase() == numapackage.UnitPhaseAdmitted && u.GetTaskType() == numapackage.TaskTypeSOCKET ||
			u.GetLifeCyclePhase() == numapackage.UnitPhaseReserved {
			toReserves = append(toReserves, u)
			continue
		}

		others = append(others, u)
	}
	return
}

func calcPreemptAllocs(p numapackage.MBPackage, mbMonitor monitor.Monitor) ([]mbAlloc, error) {
	unitToReserves, unitOthers := divideGroupIntoReserveOrOthers(p)
	allocToReserves, err := getReserveAllocs(unitToReserves)
	if err != nil {
		return nil, err
	}

	var mbToReserve int
	for _, u := range unitToReserves {
		mbToReserve += len(u.GetNUMANodes()) * SOCKETReverseMBPerNode
	}

	mbToAllocate := TotalPackageMB - mbToReserve
	allocToHardLimits, err := getHardAllocs(unitOthers, mbToAllocate, SocketLoungeMB, mbMonitor)
	if err != nil {
		return nil, err
	}

	return append(allocToHardLimits, allocToReserves...), nil
}
