package monitor

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	utilmetric "github.com/kubewharf/katalyst-core/pkg/util/metric"
)

const mbMetricToleranceInterval = 2 * time.Second

type metricStoreMBReader struct {
	metricStore *utilmetric.MetricStore
}

func isStale(timestamp *time.Time, now *time.Time) bool {
	return now.After(timestamp.Add(mbMetricToleranceInterval))
}

func toMBQoSGroup(ccdMetrics map[int]utilmetric.MetricData, now time.Time) map[int]int {
	result := make(map[int]int)
	for ccd, metric := range ccdMetrics {
		if isStale(metric.Time, &now) {
			continue
		}
		result[ccd] = int(metric.Value)
	}
	return result
}

func (m *metricStoreMBReader) GetMBQoSGroups() (map[task.QoSGroup]*MBQoSGroup, error) {
	return m.getMBQoSGroups(time.Now())
}

func (m *metricStoreMBReader) getMBQoSGroups(now time.Time) (map[task.QoSGroup]*MBQoSGroup, error) {
	rmbs, err := m.getMBQoSGroupsByMetricName("rmb", now)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve read-mb metrics")
	}
	wmbs, err := m.getMBQoSGroupsByMetricName("wmb", now)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve write-mb metrics")
	}

	groupCCDMBs := getGroupCCDMBs(rmbs, wmbs)
	groups := make(map[task.QoSGroup]*MBQoSGroup)
	for qos, ccdMB := range groupCCDMBs {
		groups[qos] = newMBQoSGroup(ccdMB)
	}

	return groups, nil
}

func (m *metricStoreMBReader) getMBQoSGroupsByMetricName(metricName string, now time.Time) (map[task.QoSGroup]map[int]int, error) {
	val := m.metricStore.GetByStringIndex(metricName)
	if val == nil {
		return nil, fmt.Errorf("index %s not found", metricName)
	}

	resctrlCCDMetricMap, ok := val.(map[string]map[int]utilmetric.MetricData)
	if !ok {
		return nil, fmt.Errorf("unexpected inner metric map type for metric %s", metricName)
	}

	result := make(map[task.QoSGroup]map[int]int)
	for ctlGroup, ccdMetrics := range resctrlCCDMetricMap {
		result[task.QoSGroup(ctlGroup)] = toMBQoSGroup(ccdMetrics, now)
	}

	return result, nil
}

func NewMetricStoreMBReader(metricStore *utilmetric.MetricStore) MBMonitor {
	return &metricStoreMBReader{
		metricStore: metricStore,
	}
}
