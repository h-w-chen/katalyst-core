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

package advisor

import (
	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type groupPCtrlState struct {
	pCtrl          pController
	ccdCapMB       int
	lowestObserved int
	belowCount     int
	belowThreshold int
}

func (g *groupPCtrlState) getCapUpdate(maxObservedMB int) int {
	delta := g.pCtrl.update(maxObservedMB)
	newCap := g.ccdCapMB + delta
	return newCap
}

func (g *groupPCtrlState) setCCDCapMB(group string, cap, maxObservedMB int) {
	oldCap := g.ccdCapMB
	oldLowest := g.lowestObserved
	oldBelowCount := g.belowCount
	candidateCap := cap

	if cap >= g.ccdCapMB {
		g.belowCount++
		if g.belowCount < g.belowThreshold {
			g.logCCDCapUpdate(group, candidateCap, maxObservedMB, oldCap, oldLowest, oldBelowCount, "wait_below_threshold")
			return
		}

		reason := "increase_after_below_threshold"
		ceiling := g.lowestObserved * 51 / 50
		if cap > ceiling {
			cap = ceiling
			reason = "increase_limited_by_lowest_observed_ceiling"
		}
		g.ccdCapMB = cap
		delta := (g.ccdCapMB - g.lowestObserved) / 100
		if delta < 1 {
			delta = 1
		}
		g.lowestObserved += delta
		g.logCCDCapUpdate(group, candidateCap, maxObservedMB, oldCap, oldLowest, oldBelowCount, reason)
		return
	}

	g.belowCount = 0
	g.ccdCapMB = cap
	reason := "decrease_due_to_observed_above_target"
	if cap < g.lowestObserved {
		g.lowestObserved = cap
		reason = "decrease_and_update_lowest_observed"
	}
	g.logCCDCapUpdate(group, candidateCap, maxObservedMB, oldCap, oldLowest, oldBelowCount, reason)
}

func (g *groupPCtrlState) logCCDCapUpdate(group string, candidateCap, maxObservedMB, oldCap, oldLowest, oldBelowCount int, reason string) {
	if !klog.V(6).Enabled() {
		return
	}

	general.Infof("[mbm] [pController] group=%s reason=%s maxObserved=%d target=%d candidateCap=%d cap=%d->%d lowestObserved=%d->%d belowCount=%d->%d belowThreshold=%d",
		group, reason, maxObservedMB, g.pCtrl.target, candidateCap,
		oldCap, g.ccdCapMB, oldLowest, g.lowestObserved, oldBelowCount, g.belowCount, g.belowThreshold)
}

func newGroupPCtrlState(Kp float64, target int, maxValue int) *groupPCtrlState {
	return &groupPCtrlState{
		pCtrl: pController{
			kp:     Kp,
			target: target,
		},
		ccdCapMB:       maxValue,
		lowestObserved: maxValue,
		belowThreshold: 30,
	}
}

type pController struct {
	kp     float64
	target int
}

func (p *pController) update(measurement int) int {
	gap := float64(p.target - measurement)
	return int(p.kp * gap)
}
