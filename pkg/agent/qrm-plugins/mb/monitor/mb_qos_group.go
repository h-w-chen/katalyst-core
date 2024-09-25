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

package monitor

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/util"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

// MBQoSGroup keeps MB of qos control group at level of CCD
type MBQoSGroup struct {
	//nodes []int
	//ccds  sets.Int

	CCDMB map[int]int
}

func SumMB(groups map[task.QoSLevel]*MBQoSGroup) int {
	sum := 0

	for _, group := range groups {
		sum += util.SumCCDMB(group.CCDMB)
	}
	return sum
}

func GetQoSKeys(qosGroups map[task.QoSLevel]*MBQoSGroup) []task.QoSLevel {
	keys := make([]task.QoSLevel, len(qosGroups))
	i := 0
	for qos, _ := range qosGroups {
		keys[i] = qos
		i++
	}
	return keys
}
