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

package reader

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/consts"
	malachitetypes "github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/malachite/types"
	metrictypes "github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/types"
)

const tolerationTime = 3 * time.Second

type MBData struct {
	MBBody     monitor.GroupMBStats
	UpdateTime int64
}

type MBReader interface {
	// GetMBData yields mb usage rate statistics
	GetMBData() (*MBData, error)
}

// metaServerMBReader converts counter usage into rate usage
type metaServerMBReader struct {
	metricFetcher interface {
		GetByStringIndex(metricName string) interface{}
	}

	currCounterData *malachitetypes.MBData
}

func isDataFresh(epocElapsed int64, now time.Time) bool {
	timestamp := time.Unix(epocElapsed, 0)
	return now.Before(timestamp.Add(tolerationTime))
}

func (m *metaServerMBReader) GetMBData() (*MBData, error) {
	return m.getMBData(time.Now())
}

func (m *metaServerMBReader) getMBData(now time.Time) (*MBData, error) {
	newCounterData, err := m.getCounterData(now)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get mb data")
	}

	var rate *MBData
	rate, err = calcRateData(newCounterData, m.currCounterData)
	if newCounterData != nil {
		m.currCounterData = newCounterData
	}
	return rate, err
}

// calcRateData derives the rate from two counters
func calcRateData(newCounter, oldCounter *malachitetypes.MBData) (*MBData, error) {
	if newCounter == nil || oldCounter == nil {
		return nil, errors.New("mb rate temporarily unavailable")
	}

	newTimeStamp := time.Unix(newCounter.UpdateTime, 0)
	oldTimeStamp := time.Unix(oldCounter.UpdateTime, 0)
	if !newTimeStamp.After(oldTimeStamp) {
		return nil, errors.New("mb data timestamp no change")
	}
	if newTimeStamp.After(oldTimeStamp.Add(tolerationTime)) {
		return nil, errors.New("mb data too stale to use")
	}

	elapsed := newTimeStamp.Sub(oldTimeStamp)
	stats, err := calcMBRate(newCounter.MBBody, oldCounter.MBBody, elapsed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calc rate data")
	}

	return &MBData{
		MBBody:     stats,
		UpdateTime: newTimeStamp.Unix(),
	}, nil
}

func calcMBRate(newCounter, oldCounter malachitetypes.MBGroupData, elapsed time.Duration) (monitor.GroupMBStats, error) {
	result := monitor.GroupMBStats{}

	msElapsed := elapsed.Milliseconds()

	// todo: more groups than / the root
	result["/"] = map[int]monitor.MBInfo{}

	oldCounterLookup := map[int]malachitetypes.MBCCDStat{}
	for _, ccdCounter := range oldCounter {
		oldCounterLookup[ccdCounter.CCDID] = ccdCounter
	}

	for _, ccdCounter := range newCounter {
		ccd := ccdCounter.CCDID
		oldCCDCounter, ok := oldCounterLookup[ccd]
		if !ok {
			return nil, fmt.Errorf("unknown ccd %d", ccd)
		}

		rateLocalMB := (ccdCounter.MBLocalCounter - oldCCDCounter.MBLocalCounter) * 1000 / msElapsed / 1024 / 1024
		rateTotalMB := (ccdCounter.MBTotalCounter - oldCCDCounter.MBTotalCounter) * 1000 / msElapsed / 1024 / 1024
		result["/"][ccd] = monitor.MBInfo{
			LocalMB:  int(rateLocalMB),
			RemoteMB: int(rateTotalMB - rateLocalMB),
			TotalMB:  int(rateTotalMB),
		}

	}
	return result, nil
}

func (m *metaServerMBReader) getCounterData(now time.Time) (*malachitetypes.MBData, error) {
	data := m.metricFetcher.GetByStringIndex(consts.MetricRealtimeMB)

	mbData, ok := data.(*malachitetypes.MBData)
	if !ok {
		return nil, errors.New("invalid data type from metric store")
	}

	if !isDataFresh(mbData.UpdateTime, now) {
		return nil, errors.New("stale mb data in metric store")
	}

	return mbData, nil
}

func New(metricFetcher metrictypes.MetricsFetcher) MBReader {
	return &metaServerMBReader{
		metricFetcher: metricFetcher,
	}
}
