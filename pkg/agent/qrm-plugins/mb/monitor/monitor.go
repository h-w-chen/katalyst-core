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
)

type MBMonitor interface {
	GetMBQoSGroups() (map[task.QoSGroup]*MBQoSGroup, error)
}

func New(taskManager task.Manager, mbReader task.TaskMBReader) (MBMonitor, error) {
	return &mbMonitor{
		taskManager: taskManager,
		mbReader:    mbReader,
	}, nil
}

type mbMonitor struct {
	taskManager task.Manager
	mbReader    task.TaskMBReader
}

func (t mbMonitor) GetMBQoSGroups() (map[task.QoSGroup]*MBQoSGroup, error) {
	if err := t.refreshTasks(); err != nil {
		return nil, err
	}

	qosCCDMB, err := t.getQoSMBs()
	if err != nil {
		return nil, err
	}

	groups := make(map[task.QoSGroup]*MBQoSGroup)
	for qos, ccdMB := range qosCCDMB {
		groups[qos] = &MBQoSGroup{
			CCDMB: ccdMB,
		}
	}

	return groups, nil
}

func (t mbMonitor) getQoSMBs() (map[task.QoSGroup]map[int]int, error) {
	result := make(map[task.QoSGroup]map[int]int)

	// todo: read in parallel to speed up
	for _, pod := range t.taskManager.GetTasks() {
		ccdMB, err := t.mbReader.ReadMB(pod)
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

func (m mbMonitor) refreshTasks() error {
	return m.taskManager.RefreshTasks()
}
