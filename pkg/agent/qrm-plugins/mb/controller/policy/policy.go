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

type MBAllocPolicy interface {
	// DistributeCCDMBs distributes total mb to CCDs
	DistributeCCDMBs(total int, mbCCD map[int]int) map[int]int

	// CalcSoftAllocs returns MB allocations to various mb units in regular mode (prioritizing Socket pods)
	CalcSoftAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error)

	// CalcPreemptAllocs returns MB allocations in hard limit preempt mode (to ensure bandwidth reserved for admitting Socket pod)
	CalcPreemptAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error)
}

type mbAllocPolicy struct{}

func (m mbAllocPolicy) DistributeCCDMBs(total int, mbCCD map[int]int) map[int]int {
	return distributeCCDMBs(total, mbCCD)
}

func (m mbAllocPolicy) CalcSoftAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error) {
	return calcSoftAllocs(units, mb, mbHiReserved, mbMonitor)
}

func (m mbAllocPolicy) CalcPreemptAllocs(units []numapackage.MBUnit, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBUnitAlloc, error) {
	return calcPreemptAllocs(units, mb, mbHiReserved, mbMonitor)
}

func New() MBAllocPolicy {
	return &mbAllocPolicy{}
}
