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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
)

func Test_DefaultConstraintDomainMBPolicy_no_throttle_with_reclaimed_only(t *testing.T) {
	t.Parallel()

	policy := NewDefaultConstraintDomainMBPolicy()
	domain := &mbdomain.MBDomain{
		NumaNodes: nil,
		CCDNode:   nil,
		NodeCCDs:  map[int][]int{1: {4, 5, 6, 7}},
	}
	qosMB := map[task.QoSLevel]*monitor.MBQoSGroup{
		task.QoSLevelSystemCores:    {CCDMB: map[int]int{4: 5_000, 5: 5_000}},
		task.QoSLevelReclaimedCores: {CCDMB: map[int]int{4: 1_000, 5: 2_000}},
	}
	mbplan := policy.GetPlan(domain, qosMB)
	assert.Equalf(t,
		&plan.MBAlloc{
			Plan: map[task.QoSLevel]map[int]int{
				task.QoSLevelSystemCores:    map[int]int{4: 256_000, 5: 256_000},
				task.QoSLevelReclaimedCores: map[int]int{4: 256_000, 5: 256_000},
			},
		},
		mbplan,
		"system and reclaimed MB should be freely allocated to the maxium, if no other qos exists")
}
