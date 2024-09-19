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

package raw

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
)

const InvalidMB = -1

type MBCalculator interface {
	CalcMB(monGroup string, dies []int) map[int]int
}

func NewMBCalculator() (MBCalculator, error) {
	return &mbCalculator{
		rawDataKeeper: make(rawDataKeeper),
	}, nil
}

type rawData struct {
	value    int64
	readTime time.Time
}

type rawDataKeeper map[string]rawData

type mbCalculator struct {
	rawDataKeeper rawDataKeeper
}

func (c mbCalculator) CalcMB(monGroup string, dies []int) map[int]int {
	return getMB(afero.NewOsFs(), monGroup, dies, time.Now(), c.rawDataKeeper)
}

func getMB(fs afero.Fs, monGroup string, dies []int, ts time.Time, dataKeeper rawDataKeeper) map[int]int {
	result := make(map[int]int)
	for _, die := range dies {
		basePath := fmt.Sprintf("mon_L3_%02d", die)
		monPath := path.Join(monGroup, "mon_data", basePath, resctrl.MBRawFile)
		v := readRawData(fs, monPath)

		mb := InvalidMB
		if lastRecord, ok := dataKeeper[monPath]; ok {
			avgMB, err := calcAverageMBinMBps(v, ts, lastRecord.value, lastRecord.readTime)
			if err == nil {
				mb = avgMB
			}
		}
		result[die] = mb

		// replace the last read raw record
		dataKeeper[monPath] = rawData{
			value:    v,
			readTime: ts,
		}
	}

	return result
}

func readRawData(fs afero.Fs, path string) int64 {
	buffer, err := afero.ReadFile(fs, path)
	if err != nil {
		return InvalidMB
	}

	if string(buffer) == "Unavailable" {
		return InvalidMB
	}

	v, err := strconv.ParseInt(string(buffer), 10, 64)
	if err != nil {
		return InvalidMB
	}

	return v
}

func calcAverageMBinMBps(currV int64, nowTime time.Time, lastV int64, lastTime time.Time) (int, error) {
	if currV == InvalidMB || lastV == InvalidMB || currV < lastV {
		return InvalidMB, errors.New("invalid MB should ignore")
	}

	elapsed := nowTime.Sub(lastTime)
	mbInMB := (currV - lastV) / elapsed.Microseconds()
	return int(mbInMB), nil
}
