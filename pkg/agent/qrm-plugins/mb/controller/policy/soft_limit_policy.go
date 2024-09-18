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

// calcSoftAllocs distributes package bandwidth to given group of units in proportion to their current usages
func calcSoftAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error) {
	hiUnits, loUnits := divideUnitsIntoHiLo(units)

	hiMB := getGroupMB(hiUnits, mbMonitor)
	loMB := getGroupMB(loUnits, mbMonitor)

	results := make([]MBUnitAlloc, 0)

	for _, u := range loUnits {
		uMB := getUnitMB(u, mbMonitor)
		results = append(results, MBUnitAlloc{
			Unit:         u,
			MBUpperBound: prorateAlloc(uMB, loMB+hiMB, mb-mbHiReserved),
		})
	}

	for _, u := range hiUnits {
		results = append(results, MBUnitAlloc{
			Unit:         u,
			MBUpperBound: SocketNodeMaxMB * len(u.GetNUMANodes()),
		})
	}

	return results, nil
}
