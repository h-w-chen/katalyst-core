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

package resctrl

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/spf13/afero"
)

const InvalidMB = -1

type CCDMBCalculator interface {
	CalcMB(monGroup string, ccd int) int
	RemoveMonGroup(monGroup string)
}

// rawData keeps the last seen state for one specific ccd mon raw value
type rawData struct {
	value    int64
	readTime time.Time
}

type mbCalculator struct {
	rwLock sync.RWMutex
	// keeping the last seen states of all processed ccd mon raw data
	prevRawData map[string]rawData
}

func (c *mbCalculator) RemoveMonGroup(monGroup string) {
	panic("impl")
}

func (c *mbCalculator) CalcMB(monGroup string, ccd int) int {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	return calcMB(afero.NewOsFs(), monGroup, ccd, time.Now(), c.prevRawData)
}

func NewCCDMBCalculator() (CCDMBCalculator, error) {
	return &mbCalculator{
		prevRawData: make(map[string]rawData),
	}, nil
}

func calcMB(fs afero.Fs, monGroup string, ccd int, tsCurr time.Time, lastSeenData map[string]rawData) int {
	ccdMon := fmt.Sprintf(TmplCCDMonFolder, ccd)
	monPath := path.Join(monGroup, MonData, ccdMon, MBRawFile)
	valueCurr := readRawData(fs, monPath)

	mb := InvalidMB
	if lastSeenRawData, ok := lastSeenData[monPath]; ok {
		mb = calcAverageInMBps(valueCurr, tsCurr, lastSeenRawData.value, lastSeenRawData.readTime)
	}

	// always refresh the last seen raw record needed for future calc
	lastSeenData[monPath] = rawData{
		value:    valueCurr,
		readTime: tsCurr,
	}

	return mb
}

func calcAverageInMBps(currV int64, nowTime time.Time, lastV int64, lastTime time.Time) int {
	if currV == InvalidMB || lastV == InvalidMB || currV < lastV {
		return InvalidMB
	}

	elapsed := nowTime.Sub(lastTime)
	mbInMB := (currV - lastV) / elapsed.Microseconds()
	return int(mbInMB)
}
