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

func (g *groupPCtrlState) setCCDCapMB(cap int) {
	if cap >= g.ccdCapMB {
		g.belowCount++
		if g.belowCount < g.belowThreshold {
			return
		}
		ceiling := g.lowestObserved * 51 / 50
		if cap > ceiling {
			cap = ceiling
		}
		g.ccdCapMB = cap
		delta := (g.ccdCapMB - g.lowestObserved) / 100
		if delta < 1 {
			delta = 1
		}
		g.lowestObserved += delta
		return
	}

	g.belowCount = 0
	g.ccdCapMB = cap
	if cap < g.lowestObserved {
		g.lowestObserved = cap
	}
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
