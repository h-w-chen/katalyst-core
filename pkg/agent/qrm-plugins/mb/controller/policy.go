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

// mbAlloc keeps the total MB allocated to a unit
type mbAlloc struct {
	unit         numapackage.MBUnit
	mbUpperBound int // MB in MBps
}

func getUnitMBUsage(u numapackage.MBUnit, mbMonitor monitor.Monitor) int {
	var sum int
	for _, n := range u.GetNUMANodes() {
		for _, mb := range mbMonitor.GetMB(n) {
			sum += mb
		}
	}

	return sum
}

func getGroupMBUsages(units []numapackage.MBUnit, mbMonitor monitor.Monitor) (hiMB, loMB int) {
	for _, u := range units {
		uMB := getUnitMBUsage(u, mbMonitor)
		if u.GetTaskType() == numapackage.TaskTypeSOCKET {
			hiMB += uMB
		} else {
			loMB += uMB
		}
	}
	return
}
