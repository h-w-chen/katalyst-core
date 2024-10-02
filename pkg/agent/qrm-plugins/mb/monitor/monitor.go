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

package monitor

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/writemb"
)

type MBMonitor interface {
	GetMBQoSGroups() (map[task.QoSGroup]*MBQoSGroup, error)
}

func New(taskManager task.Manager, rmbReader task.TaskMBReader, wmbReader writemb.WriteMBReader) (MBMonitor, error) {
	return &mbMonitor{
		taskManager: taskManager,
		rmbReader:   rmbReader,
		wmbReader:   wmbReader,
	}, nil
}

type mbMonitor struct {
	taskManager task.Manager
	rmbReader   task.TaskMBReader
	wmbReader   writemb.WriteMBReader
}

func (t mbMonitor) GetMBQoSGroups() (map[task.QoSGroup]*MBQoSGroup, error) {
	if err := t.refreshTasks(); err != nil {
		return nil, err
	}

	rQoSCCDMB, err := t.getReadsMBs()
	if err != nil {
		return nil, err
	}

	wQoSCCDMB, err := t.getWritesMBs(getCCDQoSGroups(rQoSCCDMB))
	if err != nil {
		return nil, err
	}

	groups := make(map[task.QoSGroup]*MBQoSGroup)
	for qos, ccdMB := range rQoSCCDMB {
		groups[qos] = &MBQoSGroup{
			CCDMB: ccdMB,
		}
	}
	for qos, ccdMB := range wQoSCCDMB {
		if qroup, ok := groups[qos]; !ok {
			groups[qos] = &MBQoSGroup{
				CCDMB: ccdMB,
			}
		} else {
			for ccd, mb := range ccdMB {
				qroup.CCDMB[ccd] += mb
			}
		}
	}

	return groups, nil
}

func getCCDQoSGroups(qosMBs map[task.QoSGroup]map[int]int) map[int][]task.QoSGroup {
	result := make(map[int][]task.QoSGroup)
	for qos, ccdmb := range qosMBs {
		for ccd, _ := range ccdmb {
			result[ccd] = append(result[ccd], qos)
		}
	}
	return result
}

func (t mbMonitor) getReadsMBs() (map[task.QoSGroup]map[int]int, error) {
	result := make(map[task.QoSGroup]map[int]int)

	// todo: read in parallel to speed up
	for _, pod := range t.taskManager.GetTasks() {
		ccdMB, err := t.rmbReader.ReadMB(pod)
		if err != nil {
			return nil, err
		}

		if _, ok := result[pod.QoSGroup]; !ok {
			result[pod.QoSGroup] = make(map[int]int)
		}
		for ccd, mb := range ccdMB {
			result[pod.QoSGroup][ccd] += mb
		}
	}

	return result, nil
}

func (t mbMonitor) getWritesMBs(ccdQoSGroup map[int][]task.QoSGroup) (map[task.QoSGroup]map[int]int, error) {
	result := make(map[task.QoSGroup]map[int]int)
	for ccd, groups := range ccdQoSGroup {
		mb, err := t.wmbReader.GetMB(ccd)
		if err != nil {
			return nil, err
		}
		// there may have more than one qos ctrl group binding to a specific ccd
		// for now it is fine to duplicate mb usages among them (as in POC shared_30 groups are exclusive)
		// todo: figure out proper distributions of mb among qos ctrl groups binding to given ccd
		for _, qos := range groups {
			if _, ok := result[qos]; !ok {
				result[qos] = make(map[int]int)
			}
			result[qos][ccd] = mb
		}
	}

	return result, nil
}

func (m mbMonitor) refreshTasks() error {
	return m.taskManager.RefreshTasks()
}
