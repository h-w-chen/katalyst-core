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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/util/sets"
)

// MBAManager manages MBA (memory bandwidth allocation) control-groups
type MBAManager struct {
	packages      int
	mbasByPackage MBAPackage
}

// CreateResctrlLayout ensures resctrl file system layout in line with MBAs
func (m MBAManager) CreateResctrlLayout(fs afero.Fs) error {
	if err := m.cleanupResctrlLayout(fs); err != nil {
		return errors.Wrap(err, "failed to clean up resctrl mba folder")
	}

	for _, mbas := range m.mbasByPackage {
		for _, mba := range mbas {
			if err := mba.CreateResctrlControlGroup(fs); err != nil {
				return errors.Wrap(err, "failed to create ctrl group")
			}
		}
	}

	return nil
}

func (m MBAManager) GetMBA(node int) (*MBA, error) {
	for _, mbas := range m.mbasByPackage {
		for _, mba := range mbas {
			if mba.numaNode == node {
				return mba, nil
			}
		}
	}

	return nil, errors.Errorf("MBA of node %d not found", node)
}

func (m MBAManager) cleanupResctrlLayout(fs afero.Fs) error {
	for _, mbas := range m.mbasByPackage {
		for numaNode, _ := range mbas {
			nodeCtrlGroup := resctrl.GetNodeMBAFolder(numaNode)
			if _, err := fs.Stat(nodeCtrlGroup); err != nil {
				// assuming folder not exist
				// todo: more stringent error checking
				continue
			}
			if err := fs.Remove(nodeCtrlGroup); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m MBAManager) Cleanup() {
	_ = m.cleanupResctrlLayout(afero.NewOsFs())
}

func getMBA(packageID, nodeID int, dies sets.Int, cpusByDie map[int][]int) (*MBA, error) {
	var nodeCPUS []int
	for die, _ := range dies {
		dieCPUS, ok := cpusByDie[die]
		if !ok {
			return nil, errors.Errorf("invalid mba data: failed to locate cpus for die %d", die)
		}
		nodeCPUS = append(nodeCPUS, dieCPUS...)
	}

	return &MBA{
		numaNode:       nodeID,
		cpus:           nodeCPUS,
		sharingPackage: packageID,
	}, nil
}

func New(packageByNode map[int]int, diesByNode map[int]sets.Int, cpusByDie map[int][]int) (*MBAManager, error) {
	manager := &MBAManager{
		mbasByPackage: make(MBAPackage),
	}

	for node, dies := range diesByNode {
		packageID, ok := packageByNode[node]
		if !ok {
			return nil, errors.Errorf("invalid mba data: failed to locate package for numa node %d", node)
		}

		mba, err := getMBA(packageID, node, dies, cpusByDie)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get numa node mba")
		}

		if _, ok := manager.mbasByPackage[packageID]; !ok {
			manager.mbasByPackage[packageID] = make(map[int]*MBA)
		}
		manager.mbasByPackage[packageID][node] = mba
	}

	manager.packages = len(manager.mbasByPackage)

	return manager, nil
}
