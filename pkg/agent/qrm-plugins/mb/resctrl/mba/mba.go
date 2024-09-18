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

package mba

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

// todo: support pod across numa nodes

const mbUnitAMD = 1_000 / 8 // AMD specific value of mb unit is 1/8th GBps

// MBA manages one numa node's MB
type MBA struct {
	numaNode       int
	cpus           []int
	sharingPackage int
}

func (m MBA) CreateResctrlControlGroup(fs afero.Fs) error {
	nodeCtrlGroup := getNodeMBAFolder(m.numaNode)
	if err := fs.Mkdir(nodeCtrlGroup, resctrl.FolderPerm); err != nil {
		return err
	}

	cpulistFilePath := path.Join(nodeCtrlGroup, resctrl.CPUListFile)
	cpus := intsTostrs(m.cpus)
	cpuslist := strings.Join(cpus, ",")
	general.InfofV(6, "mbm: node %d cpus: %s", m.numaNode, cpuslist)
	return afero.WriteFile(fs, cpulistFilePath, []byte(cpuslist), resctrl.FilePerm)
}

func toSchmataInst(mbCCD map[int]int) string {
	if len(mbCCD) == 0 {
		return ""
	}

	//echo "MB:2=32;3=32;" > mba1/schemata
	var sb strings.Builder
	sb.WriteString("MB:")
	for ccd, mb := range mbCCD {
		v := (mb + mbUnitAMD - 1) / mbUnitAMD
		sb.WriteString(fmt.Sprintf("%d=%d;", ccd, v))
	}
	return sb.String()
}

func (m MBA) SetSchemataMBs(mbCCD map[int]int) error {
	return m.setSchemataMBs(afero.NewOsFs(), mbCCD)
}

func (m MBA) setSchemataMBs(fs afero.Fs, mbCCD map[int]int) error {
	nodeCtrlGroup := getNodeMBAFolder(m.numaNode)
	schemataPath := path.Join(nodeCtrlGroup, resctrl.SchemataFile)
	return afero.WriteFile(fs, schemataPath, []byte(toSchmataInst(mbCCD)), resctrl.FilePerm)
}

func intsTostrs(ints []int) []string {
	strs := make([]string, len(ints))

	for i, v := range ints {
		strs[i] = strconv.Itoa(v)
	}

	return strs
}

func getNodeMBAFolder(node int) string {
	nodeBasePath := fmt.Sprintf("%s%d", resctrl.NumaFolderPrefix, node)
	return path.Join(resctrl.FsRoot, nodeBasePath)
}
