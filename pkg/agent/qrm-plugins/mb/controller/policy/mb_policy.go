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
	"time"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/config"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type DomainMBPolicy interface {
	GetPlan(totalMB int, domain *mbdomain.MBDomain, currQoSMB map[task.QoSGroup]*monitor.MBQoSGroup) *plan.MBAlloc
}

type domainMBPolicy struct {
	preemptMBPolicy    DomainMBPolicy
	constraintMBPolicy DomainMBPolicy
}

func calcResvForIncubation(incubates mbdomain.IncubatedCCDs, currQoSMB map[task.QoSGroup]*monitor.MBQoSGroup) int {
	var reserveToIncubate int

	for qos, qosmb := range currQoSMB {
		if qos != task.QoSGroupDedicated {
			continue
		}

		for ccd, mb := range qosmb.CCDMB {
			if _, ok := incubates[ccd]; !ok {
				// assuming it is active incubated
				continue
			}
			if mb.ReadsMB+mb.WritesMB >= config.ReservedPerCCD {
				continue
			}
			reserveToIncubate += config.ReservedPerCCD - (mb.ReadsMB + mb.WritesMB)
		}
	}

	return reserveToIncubate
}

func (d domainMBPolicy) GetPlan(totalMB int, domain *mbdomain.MBDomain, currQoSMB map[task.QoSGroup]*monitor.MBQoSGroup) *plan.MBAlloc {
	// some newly started pods may still be in so-called incubation period;
	// special care need to take to ensure they have no less than the reserved mb
	// by subtracting the mb for incubation
	domain.CleanseIncubates()
	mbForIncubation := calcResvForIncubation(domain.CloneIncubates(), currQoSMB)
	general.InfofV(6, "mbm: domain %v reserve mb for incubation %d", domain.ID, mbForIncubation)

	availableMB := totalMB - mbForIncubation
	if availableMB < 0 {
		availableMB = 0
	}

	if len(domain.GetPreemptingNodes()) != 0 {
		return d.preemptMBPolicy.GetPlan(availableMB, domain, currQoSMB)
	}

	return d.constraintMBPolicy.GetPlan(availableMB, domain, currQoSMB)
}

func newDomainMBPolicy(preemptMBPolicy, softLimitMBPolicy DomainMBPolicy) (DomainMBPolicy, error) {
	return &domainMBPolicy{
		preemptMBPolicy:    preemptMBPolicy,
		constraintMBPolicy: softLimitMBPolicy,
	}, nil
}

func NewDefaultDomainMBPolicy(incubationInterval time.Duration) (DomainMBPolicy, error) {
	return newDomainMBPolicy(NewDefaultPreemptDomainMBPolicy(), NewDefaultConstraintDomainMBPolicy())
}