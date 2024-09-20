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
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type TaskManager interface {
	GetTasks(node int) []int
	NewTask(pod *v1.Pod) (*Task, error)
	RemoveTask(task *Task)
}

type taskManager struct {
	podResource PodResource

	nodeTaskIDs map[int]sets.Int
}

func (t taskManager) GetTasks(node int) []int {
	tasks, ok := t.nodeTaskIDs[node]
	if !ok {
		return nil
	}

	pids := make([]int, len(tasks))
	i := 0
	for pid, _ := range tasks {
		pids[i] = pid
		i++
	}

	return pids
}

func (t taskManager) NewTask(pod *v1.Pod) (*Task, error) {
	return t.newTask(afero.NewOsFs(), pod)
}

func (t taskManager) newTask(fgs afero.Fs, pod *v1.Pod) (*Task, error) {
	node, err := t.podResource.GetNumaNode(pod)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get numa node of pod")
	}

	pid, err := t.podResource.GetPid(pod)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pid of pod")
	}

	var task *Task
	task, err = newTask(afero.NewOsFs(), pod, node, pid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create task instance")
	}

	t.nodeTaskIDs[node].Insert(pid)
	return task, nil
}

func (t taskManager) RemoveTask(task *Task) {
	t.nodeTaskIDs[task.numaNode].Delete(task.idProcess)
}

func New(podResource PodResource) (TaskManager, error) {
	return &taskManager{
		podResource: podResource,
		nodeTaskIDs: make(map[int]sets.Int),
	}, nil
}
