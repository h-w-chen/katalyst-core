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
	"fmt"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	"path"
	"strings"
)

const tmplProcTaskFolder = "/proc/%d/task"

// todo: support pod across numa nodes

type Task struct {
	numaNode  int
	pod       *v1.Pod
	idProcess int
	idThreads []string
}

func cleanupFolder(fs afero.Fs, folder string) error {
	return fs.Remove(folder)
}

func getMonCtrlGroupFolder(node, pid int) string {
	monGroup := fmt.Sprintf("%s%d_pid_%d", resctrl.NumaFolderPrefix, node, pid)
	return path.Join(resctrl.MonGroupRoot, monGroup)
}

func (t Task) OnReady(fs afero.Fs) error {
	monGroupFullPath := getMonCtrlGroupFolder(t.numaNode, t.idProcess)
	if err := fs.Mkdir(monGroupFullPath, resctrl.FolderPerm); err != nil {
		return err
	}

	taskFilePath := path.Join(monGroupFullPath, resctrl.TasksFile)
	threads := strings.Join(t.idThreads, "\n")
	return afero.WriteFile(fs, taskFilePath, []byte(threads), resctrl.FilePerm)
}

func (t Task) OnTerminate(fs afero.Fs) error {
	monGroupFullPath := getMonCtrlGroupFolder(t.numaNode, t.idProcess)
	return cleanupFolder(fs, monGroupFullPath)
}

func newTask(fs afero.Fs, pod *v1.Pod, node, pid int) (*Task, error) {
	task := &Task{
		numaNode:  node,
		pod:       pod,
		idProcess: pid,
	}

	var err error
	if task.idThreads, err = getThreads(fs, pid); err != nil {
		return nil, err
	}

	return task, nil
}
