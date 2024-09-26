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

package qospolicy

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

// valveQoSMBPolicy is a special form of qos mb policy
// its middle tier ("floating") combines to upper tier - if applicable, or falls back to lower tier
type valveQoSMBPolicy struct {
	either, or QoSMBPolicy
	filter     func(mbQoSGroups map[task.QoSLevel]*monitor.MBQoSGroup, isTopMost bool) bool
}

func (v valveQoSMBPolicy) GetPlan(totalMB int, mbQoSGroups map[task.QoSLevel]*monitor.MBQoSGroup, isTopMost bool) *plan.MBAlloc {
	if v.filter(mbQoSGroups, isTopMost) {
		return v.either.GetPlan(totalMB, mbQoSGroups, isTopMost)
	}

	return v.or.GetPlan(totalMB, mbQoSGroups, isTopMost)
}

func NewValveQoSMBPolicy(condition func(mbQoSGroups map[task.QoSLevel]*monitor.MBQoSGroup, isTopMost bool) bool,
	either, or QoSMBPolicy,
) QoSMBPolicy {
	return &valveQoSMBPolicy{
		either: either,
		or:     or,
		filter: condition,
	}
}
