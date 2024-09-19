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

package mbm

import (
	"path"
	"sync"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/raw"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/task"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

// intervalTaskMBM is the interval to scan task mon_groups and calculate the MB
const intervalTaskMBM = time.Second * 1

type TaskMonitor interface {
	GetTaskMB(pid int) map[int]int
	AggregateNodeMB(node int) map[int]int
	skeleton.GenericPlugin
}

type taskMonitor struct {
	rwLock sync.RWMutex
	// taskMBs keeps each task's CCD-mb table
	// task id should be used to consult with task manager
	taskMBs     map[int]map[int]int
	taskManager task.TaskManager

	mbCalculator raw.MBCalculator

	// nodeCCDs helps to locate the CCDs of a task via the associated numa node
	nodeCCDs map[int][]int

	chStop chan struct{}
}

func (t *taskMonitor) Name() string {
	return "task_mb_manager"
}

func (t *taskMonitor) Start() error {
	general.Infof("mbm: resctrl task monitor started")
	go t.run()
	return nil
}

func (t *taskMonitor) Stop() error {
	general.Infof("mbm: resctrl task monitor requested to stop")
	t.chStop <- struct{}{}
	return nil
}

func (t *taskMonitor) GetTaskMB(pid int) map[int]int {
	t.rwLock.RLock()
	defer t.rwLock.RUnlock()
	return maps.Clone(t.taskMBs[pid])
}

func (t *taskMonitor) AggregateNodeMB(node int) map[int]int {
	result := make(map[int]int)

	t.rwLock.RLock()
	defer t.rwLock.RUnlock()
	for _, pid := range t.taskManager.GetTasks(node) {
		for ccd, mb := range t.taskMBs[pid] {
			result[ccd] += mb
		}
	}

	return result
}

func (t *taskMonitor) run() {
	general.Infof("mbm: resctrl task monitor main loop run")
	wait.Until(t.scan, intervalTaskMBM, t.chStop)
	general.Infof("mbm: resctrl task monitor run exited")
}

func (t *taskMonitor) scan() {
	t.scanFs(afero.NewOsFs())
}

func (t *taskMonitor) scanFs(fs afero.Fs) {
	fis, err := afero.ReadDir(fs, resctrl.MonGroupRoot)
	if err != nil {
		general.Errorf("mbm: failed to scan task mon groups: %v", err)
		return
	}

	for _, finfo := range fis {
		if !finfo.IsDir() {
			continue
		}
		// extract node/pid for the task, so mb could associate to the node aggregates
		basePath := finfo.Name()
		var node, pid int
		node, pid, err = task.GetTaskInfo(basePath)
		if err != nil {
			// todo: consider delete erred folder?
			continue
		}
		ccds := t.nodeCCDs[node]
		mbs := t.mbCalculator.CalcMB(path.Join(resctrl.MonGroupRoot, finfo.Name()), ccds)
		t.updateTaskMB(pid, mbs)
	}
}

func (t *taskMonitor) updateTaskMB(pid int, ccdMB map[int]int) {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.taskMBs[pid] = ccdMB
}

func NewTaskManager(nodeCCDs map[int][]int, taskManager task.TaskManager, mbCalculator raw.MBCalculator) (TaskMonitor, error) {
	return &taskMonitor{
		taskMBs:      make(map[int]map[int]int),
		taskManager:  taskManager,
		mbCalculator: mbCalculator,
		nodeCCDs:     nodeCCDs,
		chStop:       make(chan struct{}),
	}, nil
}
