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

package strategy

import (
	types2 "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/controller/action"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"math"
	"time"
)

// defaultDecayB is the recommended base b in decay formula: a*b^(-x)
const defaultDecayB = math.E / 2

type PowerActionStrategy interface {
	RecommendAction(currentWatt, desiredWatt int,
		alert spec.PowerAlert, internalOp spec.InternalOp, ttl time.Duration,
	) types2.PowerAction
}

type ruleBasedPowerStrategy struct {
	coefficient linearDecay
}

func (p ruleBasedPowerStrategy) RecommendAction(actualWatt, desiredWatt int,
	alert spec.PowerAlert, internalOp spec.InternalOp, ttl time.Duration,
) types2.PowerAction {
	if actualWatt <= desiredWatt {
		return types2.PowerAction{Op: spec.InternalOpPause, Arg: 0}
	}

	// stale request; ignore and return no-op; hopefully next time it will rectify
	if ttl <= -time.Minute*5 || spec.InternalOpPause == internalOp {
		return types2.PowerAction{Op: spec.InternalOpPause, Arg: 0}
	}

	if ttl <= time.Minute*2 {
		return types2.PowerAction{Op: spec.InternalOpFreqCap, Arg: desiredWatt}
	}

	op := internalOp
	if spec.InternalOpAuto == op {
		op = p.autoAction(actualWatt, desiredWatt, ttl)
	}

	if spec.InternalOpFreqCap == op {
		return types2.PowerAction{Op: spec.InternalOpFreqCap, Arg: desiredWatt}
	} else if spec.InternalOpEvict == op {
		return types2.PowerAction{
			Op:  spec.InternalOpEvict,
			Arg: p.coefficient.calcExcessiveInPercent(desiredWatt, actualWatt, ttl),
		}
	}

	return types2.PowerAction{Op: spec.InternalOpPause, Arg: 0}
}

func (p ruleBasedPowerStrategy) autoAction(actualWatt, desiredWatt int, ttl time.Duration) spec.InternalOp {
	// upstream caller guarantees actual > desired
	ttlS0, _ := spec.GetPowerAlertResponseTimeLimit(spec.PowerAlertS0)
	if ttl <= ttlS0 {
		return spec.InternalOpFreqCap
	}

	ttlF1, _ := spec.GetPowerAlertResponseTimeLimit(spec.PowerAlertP1)
	if ttl <= ttlF1 {
		return spec.InternalOpEvict
	}

	// todo: consider throttle action, after load throttle is enabled
	return spec.InternalOpPause
}

type linearDecay struct {
	b float64
}

func (d linearDecay) calcExcessiveInPercent(target, curr int, ttl time.Duration) int {
	// exponential decaying formula: a*b^(-t)
	a := 100 - target*100/curr
	decay := math.Pow(d.b, float64(-int(ttl.Minutes())/10))
	return int(float64(a) * decay)
}

var _ PowerActionStrategy = &ruleBasedPowerStrategy{}

func NewRuleBasedPowerStrategy() PowerActionStrategy {
	return ruleBasedPowerStrategy{coefficient: linearDecay{b: defaultDecayB}}
}
