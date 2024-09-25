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
	"fmt"
	"path"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-api/pkg/consts"
	resctrlconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/consts"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task/cgnode"
)

type QoSLevel = consts.QoSLevel

const (
	QoSLevelReclaimedCores QoSLevel = consts.QoSLevelReclaimedCores
	QoSLevelSharedCores    QoSLevel = consts.QoSLevelSharedCores
	QoSLevelDedicatedCores QoSLevel = consts.QoSLevelDedicatedCores
	QoSLevelSystemCores    QoSLevel = consts.QoSLevelSystemCores
)

var qosFolderLookup = map[QoSLevel]string{
	QoSLevelDedicatedCores: resctrlconsts.GroupDedicated,
	QoSLevelSharedCores:    resctrlconsts.GroupSharedCore,
	QoSLevelReclaimedCores: resctrlconsts.GroupReclaimed,
	QoSLevelSystemCores:    resctrlconsts.GroupSystem,
}

var qosToCgroupv1GroupFolder = map[QoSLevel]string{
	QoSLevelDedicatedCores: "burstable",
	QoSLevelSharedCores:    "burstable",
	QoSLevelReclaimedCores: "offline-besteffort",
	QoSLevelSystemCores:    "burstable",
}

type Task struct {
	QoSLevel QoSLevel
	PodUID   string

	pid   int
	spids []int

	NumaNode []int
	nodeCCDs map[int]sets.Int
}

func (t Task) GetID() string {
	return t.PodUID
}

func GetResctrlCtrlGroupFolder(qos QoSLevel) (string, error) {
	qosFolder, ok := qosFolderLookup[qos]
	if !ok {
		return "", errors.New("invalid qos level of task")
	}

	return path.Join(resctrlconsts.FsRoot, qosFolder), nil
}

func (t Task) GetResctrlCtrlGroup() (string, error) {
	return GetResctrlCtrlGroupFolder(t.QoSLevel)
}

func (t Task) GetResctrlMonGroup() (string, error) {
	taskCtrlGroup, err := t.GetResctrlCtrlGroup()
	if err != nil {
		return "", err
	}

	taskFolder := fmt.Sprintf(resctrlconsts.TmplTaskFolder, t.PodUID)
	return path.Join(taskCtrlGroup, resctrlconsts.SubGroupMonRoot, taskFolder), nil
}

func (t Task) GetCCDs() []int {
	ccds := make([]int, 0)
	for _, node := range t.NumaNode {
		for ccd, _ := range t.nodeCCDs[node] {
			ccds = append(ccds, ccd)
		}
	}
	return ccds
}

func getCgroupCPUSetPath(podUID string, qos QoSLevel) string {
	// todo: support cgroup v2
	// below assumes cgroup v1
	return path.Join("/sys/fs/cgroup/cpuset/kubepods/", qosToCgroupv1GroupFolder[qos], "pod"+podUID)
}

func getNumaNodes(podUID string, qos QoSLevel) ([]int, error) {
	return cgnode.GetNumaNodes(getCgroupCPUSetPath(podUID, qos))
}
