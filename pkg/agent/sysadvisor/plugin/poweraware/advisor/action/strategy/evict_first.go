package strategy

import (
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor/action"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
)

const dvfsLimit = 10

type EvictableProber interface {
	HasEvictablePods() bool
}

// evictFirstStrategy always attempts to evict low priority pods if any; only after all are exhausted will it resort to DVFS means.
// besides, it will continue to try the best to meet the alert spec, regardless of the alert update time.
// alert level has the following meanings in this strategy:
// P1 - eviction only;
// P0 - evict if applicable; otherwise conduct DVFS once if needed (DVFS is limited to 10%);
// S0 - DVFS in urgency (no limit on DVFS)
type evictFirstStrategy struct {
	coefficient     exponentialDecay
	evictableProber EvictableProber
	dvfsUsed        int
}

func (e *evictFirstStrategy) DVFSReset() {
	e.dvfsUsed = 0
}

func (e *evictFirstStrategy) recommendOpForAlertP0(ttl time.Duration) spec.InternalOp {
	// always prefer eviction over dvfs if possible
	if !e.evictableProber.HasEvictablePods() && e.dvfsUsed < dvfsLimit {
		return spec.InternalOpFreqCap
	}

	return spec.InternalOpEvict
}

func (e *evictFirstStrategy) recommendOp(alert spec.PowerAlert, internalOp spec.InternalOp, ttl time.Duration) spec.InternalOp {
	if internalOp != spec.InternalOpAuto {
		return internalOp
	}

	switch alert {
	case spec.PowerAlertS0:
		return spec.InternalOpFreqCap
	case spec.PowerAlertP0:
		return e.recommendOpForAlertP0(ttl)
	case spec.PowerAlertP1:
		return spec.InternalOpEvict
	default:
		return spec.InternalOpNoop
	}
}

func estimateDVFSStrength(actualWatt, desiredWatt int) int {
	return (actualWatt - desiredWatt) * 100 / actualWatt
}

func (e *evictFirstStrategy) calConstraintDVFSTarget(actualWatt, desiredWatt int) (int, error) {
	leftPercentage := dvfsLimit - e.dvfsUsed
	if leftPercentage <= 0 {
		return 0, errors.New("no room for dvfs")
	}

	lowerLimit := (100 - leftPercentage) * actualWatt / 100
	if lowerLimit > desiredWatt {
		return lowerLimit, nil
	}
	return desiredWatt, nil
}

func (e *evictFirstStrategy) RecommendAction(actualWatt int, desiredWatt int, alert spec.PowerAlert, internalOp spec.InternalOp, ttl time.Duration) action.PowerAction {
	if actualWatt <= desiredWatt {
		return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
	}

	op := e.recommendOp(alert, internalOp, ttl)
	switch op {
	case spec.InternalOpFreqCap:
		// try to conduct freq capping within the allowed limit if not set for hard dvfs
		if internalOp != spec.InternalOpFreqCap && !(alert == spec.PowerAlertS0 && internalOp == spec.InternalOpAuto) {
			var err error
			desiredWatt, err = e.calConstraintDVFSTarget(actualWatt, desiredWatt)
			if err != nil {
				return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
			}
		}
		e.dvfsUsed += estimateDVFSStrength(actualWatt, desiredWatt)
		return action.PowerAction{Op: spec.InternalOpFreqCap, Arg: desiredWatt}
	case spec.InternalOpEvict:
		return action.PowerAction{
			Op:  spec.InternalOpEvict,
			Arg: e.coefficient.calcExcessiveInPercent(desiredWatt, actualWatt, ttl),
		}
	default:
		return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
	}
}

func NewEvictFirstStrategy(prober EvictableProber) PowerActionStrategy {
	return &evictFirstStrategy{
		coefficient:     exponentialDecay{b: defaultDecayB},
		evictableProber: prober,
		dvfsUsed:        0,
	}
}
