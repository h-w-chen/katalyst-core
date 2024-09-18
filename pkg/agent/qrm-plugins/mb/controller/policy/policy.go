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

func getUnitMB(u numapackage.MBUnit, mbMonitor monitor.Monitor) int {
	var sum int
	for _, n := range u.GetNUMANodes() {
		for _, mb := range mbMonitor.GetMB(n) {
			sum += mb
		}
	}

	return sum
}

func getGroupMB(units []numapackage.MBUnit, mbMonitor monitor.Monitor) int {
	sum := 0
	for _, u := range units {
		sum += getUnitMB(u, mbMonitor)
	}

	return sum
}

func getHiLoGroupMBs(units []numapackage.MBUnit, mbMonitor monitor.Monitor) (hiMB, loMB int) {
	hiUnits, loUnis := divideUnitsIntoHiLo(units)
	hiMB, loMB = getGroupMB(hiUnits, mbMonitor), getGroupMB(loUnis, mbMonitor)
	return
}

func divideUnitsIntoHiLo(units []numapackage.MBUnit) (hi, lo []numapackage.MBUnit) {
	for _, u := range units {
		if u.GetTaskType() == numapackage.TaskTypeSOCKET {
			hi = append(hi, u)
			continue
		}

		lo = append(lo, u)
	}
	return
}

func prorateAlloc(own, total, allocatable int) int {
	return int(float64(allocatable) * float64(own) / float64(total))
}

func CcdDistributeMB(total int, mbCCD map[int]int) map[int]int {
	currMB := 0
	for _, v := range mbCCD {
		currMB += v
	}

	ccdAllocs := make(map[int]int)
	for ccd, v := range mbCCD {
		ccdAllocs[ccd] = prorateAlloc(v, currMB, total)
	}

	return ccdAllocs
}
