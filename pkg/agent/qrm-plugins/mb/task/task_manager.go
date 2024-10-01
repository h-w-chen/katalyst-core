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

package task

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/util/sets"

	resctrlconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/consts"
	resctrlfile "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/file"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/state"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task/cgutil"
)

type Manager interface {
	GetTasks() []*Task
	NewTask(podID string, qos QoSGroup) (*Task, error)
	FindTask(id string) (*Task, error)
	DeleteTask(task *Task)
	RefreshTasks() error
}

func New(nodeCCDs map[int]sets.Int, cpusInCCD map[int][]int, cleaner state.MBRawDataCleaner) (Manager, error) {
	cpuCCD := make(map[int]int)
	for ccd, cpus := range cpusInCCD {
		for _, cpu := range cpus {
			cpuCCD[cpu] = ccd
		}
	}

	return &manager{
		rawStateCleaner: cleaner,
		nodeCCDs:        nodeCCDs,
		cpuCCD:          cpuCCD,
		fs:              afero.NewOsFs(),
	}, nil
}

type manager struct {
	rawStateCleaner state.MBRawDataCleaner
	nodeCCDs        map[int]sets.Int
	cpuCCD          map[int]int

	rwLock  sync.RWMutex
	tasks   map[string]*Task
	taskQoS map[string]QoSGroup

	fs afero.Fs
}

func (m *manager) RefreshTasks() error {
	monGroupPathRefeshed, err := resctrlfile.GetResctrlMonGroups(m.fs)
	if err != nil {
		return err
	}

	// ensure task manager's tasks in line with mon groups identified
	tasksToDelete, err := m.identifyTaskToDelete(monGroupPathRefeshed)
	if err != nil {
		return err
	}
	for _, tasksToDelete := range tasksToDelete {
		m.DeleteTask(tasksToDelete)
	}

	newMonGroups, err := m.identifyNewMonGroups(monGroupPathRefeshed)
	if err != nil {
		return err
	}
	for _, newMonGroup := range newMonGroups {
		qos, podUID, err := ParseMonGroup(newMonGroup)
		if err != nil {
			return err
		}
		_, err = m.NewTask(podUID, qos)
		if err != nil {
			return err
		}
	}

	return nil
}

// todo: remove it if not really needed
func ctrlGroupToQoSLevel(ctrlGroup string) (QoSGroup, error) {
	return QoSGroup(ctrlGroup), nil
}

func ParseMonGroup(path string) (QoSGroup, string, error) {
	stem := strings.TrimPrefix(path, resctrlconsts.FsRoot)
	stem = strings.Trim(stem, "/")
	segs := strings.Split(stem, "/")
	if len(segs) != 3 {
		return "", "", fmt.Errorf("invalid mon group path: %s", path)
	}

	ctrlGroup, monGroup := segs[0], segs[2]
	qos, err := ctrlGroupToQoSLevel(ctrlGroup)
	if err != nil {
		return "", "", err
	}
	return qos, monGroup, nil
}

func (m *manager) getTaskMonGroups() (map[string]*Task, error) {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	result := make(map[string]*Task)
	for _, task := range m.tasks {
		monGroup, err := task.GetResctrlMonGroup()
		if err != nil {
			return nil, err
		}
		result[monGroup] = task
	}
	return result, nil
}

func (m *manager) identifyTaskToDelete(monGroups []string) ([]*Task, error) {
	refreshed := make(map[string]struct{})
	for _, monGroup := range monGroups {
		refreshed[monGroup] = struct{}{}
	}

	result := make([]*Task, 0)

	for _, task := range m.tasks {
		taskMonGroup, err := task.GetResctrlMonGroup()
		if err != nil {
			return nil, err
		}
		if _, ok := refreshed[taskMonGroup]; !ok {
			result = append(result, task)
		}
	}

	return result, nil
}

func (m *manager) identifyNewMonGroups(monGroups []string) ([]string, error) {
	result := make([]string, 0)

	existentMonGroups, err := m.getTaskMonGroups()
	if err != nil {
		return nil, err
	}

	for _, monGroup := range monGroups {
		if _, ok := existentMonGroups[monGroup]; !ok {
			result = append(result, monGroup)
		}
	}

	return result, nil
}

func (m *manager) DeleteTask(task *Task) {
	monGroup, _ := task.GetResctrlMonGroup()
	m.rawStateCleaner.Cleanup(monGroup)

	m.rwLock.Lock()
	m.rwLock.Unlock()
	delete(m.taskQoS, task.GetID())
	delete(m.tasks, task.GetID())
}

func (m *manager) FindTask(id string) (*Task, error) {
	m.rwLock.RLock()
	m.rwLock.RUnlock()
	task, ok := m.tasks[id]
	if !ok {
		return nil, errors.New("no task by the id")
	}

	return task, nil
}

func (m *manager) getNumaNodes(podUID string, qos QoSGroup) ([]int, error) {
	cgPath, err := getCgroupCPUSetPath(podUID, qos)
	if err != nil {
		return nil, err
	}
	return cgutil.GetNumaNodes(m.fs, cgPath)
}

func (m *manager) getBoundCPUs(podUID string, qos QoSGroup) ([]int, error) {
	cgPath, err := getCgroupCPUSetPath(podUID, qos)
	if err != nil {
		return nil, err
	}
	return cgutil.GetCPUs(m.fs, cgPath)
}

func (m *manager) NewTask(podID string, qos QoSGroup) (*Task, error) {
	nodes, err := m.getNumaNodes(podID, qos)
	if err != nil {
		return nil, err
	}

	cpus, err := m.getBoundCPUs(podID, qos)
	if err != nil {
		return nil, err
	}

	task := &Task{
		QoSGroup: qos,
		PodUID:   podID,
		NumaNode: nodes,
		nodeCCDs: m.nodeCCDs,
		CPUs:     cpus,
		cpuCCD:   m.cpuCCD,
	}

	m.addTask(task)
	return task, nil
}

func (m *manager) addTask(task *Task) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	m.taskQoS[task.GetID()] = task.QoSGroup
	m.tasks[task.GetID()] = task
}

func (m *manager) GetTasks() []*Task {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	result := make([]*Task, len(m.tasks))
	i := 0
	for _, task := range m.tasks {
		result[i] = task
		i++
	}

	return result
}
