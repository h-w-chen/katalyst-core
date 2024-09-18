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

type MBAllocPolicy interface {

	// CalcSoftAllocs returns MB allocations to various mb units in regular mode (prioritizing Socket pods)
	CalcSoftAllocs(units []apppool.Pool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error)

	// CalcPreemptAllocs returns MB allocations in hard limit preempt mode (to ensure bandwidth reserved for admitting Socket pod)
	CalcPreemptAllocs(units []apppool.Pool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error)
}

type mbAllocPolicy struct{}

func (m mbAllocPolicy) CalcSoftAllocs(units []apppool.Pool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error) {
	return calcSoftAllocs(units, mb, mbHiReserved, mbMonitor)
}

func (m mbAllocPolicy) CalcPreemptAllocs(units []apppool.Pool, mb int, mbHiReserved int, mbMonitor monitor.Monitor) ([]MBAlloc, error) {
	return calcPreemptAllocs(units, mb, mbHiReserved, mbMonitor)
}

func New() MBAllocPolicy {
	return &mbAllocPolicy{}
}
