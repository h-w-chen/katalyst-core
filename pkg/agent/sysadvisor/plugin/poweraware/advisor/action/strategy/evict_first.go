package strategy

import (
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/advisor/action"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"github.com/kubewharf/katalyst-core/pkg/consts"
	metrictypes "github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/types"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	// the limit of dvfs effect a voluntary dvfs plan is allowed
	voluntaryDVFSEffectMaximum = 10
	// threshold of cpu usgae that allows voluntary dvfs
	voluntaryDVFSCPUUsageThreshold = 0.45
)

type EvictableProber interface {
	HasEvictablePods() bool
}

// dvfsTracker keeps track and accumulates the effect lowering power by means of dvfs
type dvfsTracker struct {
	dvfsAccumEffect int

	// to keep track of current dvfs effect
	inDVFS    bool
	prevPower int
}

func (d *dvfsTracker) update(actualWatt, desiredWatt int) {
	// only accumulate if dvfs is engaged
	if d.prevPower >= 0 && d.inDVFS {
		// if actual power is more than previous, likely previous round dvfs took no effect; not to take into account
		if actualWatt < d.prevPower {
			dvfsEffect := (d.prevPower - actualWatt) * 100 / d.prevPower
			d.dvfsAccumEffect += dvfsEffect
		}
	}

	d.prevPower = actualWatt
}

func (d *dvfsTracker) dvfsEnter() {
	d.inDVFS = true
}

func (d *dvfsTracker) dvfsExit() {
	d.inDVFS = false
}

func (d *dvfsTracker) clear() {
	d.dvfsAccumEffect = 0
	d.prevPower = 0
	d.inDVFS = false
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
	dvfsTracker     dvfsTracker
	metricsReader   metrictypes.MetricsReader
}

func (e *evictFirstStrategy) DVFSReset() {
	e.dvfsTracker.clear()
}

func (e *evictFirstStrategy) allowVoluntaryFreqCap() bool {
	// todo: consider cpu frequency
	if e.metricsReader != nil {
		if cpuUsage, err := e.metricsReader.GetNodeMetric(consts.MetricCPUUsageRatio); err == nil {
			if cpuUsage.Value <= voluntaryDVFSCPUUsageThreshold {
				return false
			}
		}
	}

	return e.dvfsTracker.dvfsAccumEffect < voluntaryDVFSEffectMaximum
}

func (e *evictFirstStrategy) recommendOpForAlertP0(ttl time.Duration) spec.InternalOp {
	// always prefer eviction over dvfs if possible
	if e.evictableProber.HasEvictablePods() {
		return spec.InternalOpEvict
	}

	if e.allowVoluntaryFreqCap() {
		return spec.InternalOpFreqCap
	}

	return spec.InternalOpNoop
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

func (e *evictFirstStrategy) calConstraintDVFSTarget(actualWatt, desiredWatt int) (int, error) {
	leftPercentage := voluntaryDVFSEffectMaximum - e.dvfsTracker.dvfsAccumEffect
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
	e.dvfsTracker.update(actualWatt, desiredWatt)
	general.InfofV(6, "pap: dvfs effect: %d", e.dvfsTracker.dvfsAccumEffect)

	if actualWatt <= desiredWatt {
		e.dvfsTracker.dvfsExit()
		return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
	}

	op := e.recommendOp(alert, internalOp, ttl)
	switch op {
	case spec.InternalOpFreqCap:
		e.dvfsTracker.dvfsEnter()
		// try to conduct freq capping within the allowed limit if not set for hard dvfs
		if internalOp != spec.InternalOpFreqCap && !(alert == spec.PowerAlertS0 && internalOp == spec.InternalOpAuto) {
			var err error
			desiredWatt, err = e.calConstraintDVFSTarget(actualWatt, desiredWatt)
			if err != nil {
				return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
			}
		}
		return action.PowerAction{Op: spec.InternalOpFreqCap, Arg: desiredWatt}
	case spec.InternalOpEvict:
		e.dvfsTracker.dvfsExit()
		return action.PowerAction{
			Op:  spec.InternalOpEvict,
			Arg: e.coefficient.calcExcessiveInPercent(desiredWatt, actualWatt, ttl),
		}
	default:
		e.dvfsTracker.dvfsExit()
		return action.PowerAction{Op: spec.InternalOpNoop, Arg: 0}
	}
}

func NewEvictFirstStrategy(prober EvictableProber, metricsReader metrictypes.MetricsReader) PowerActionStrategy {
	return &evictFirstStrategy{
		coefficient:     exponentialDecay{b: defaultDecayB},
		evictableProber: prober,
		dvfsTracker:     dvfsTracker{dvfsAccumEffect: 0},
		metricsReader:   metricsReader,
	}
}
