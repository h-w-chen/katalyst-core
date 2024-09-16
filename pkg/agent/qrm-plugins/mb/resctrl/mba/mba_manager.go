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
	if err := m.CleanupResctrlLayout(fs); err != nil {
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

func (m MBAManager) CleanupResctrlLayout(fs afero.Fs) error {
	for _, mbas := range m.mbasByPackage {
		for numaNode, _ := range mbas {
			nodeCtrlGroup := getNodeMBAFolder(numaNode)
			if _, err := fs.Stat(nodeCtrlGroup); err != nil {
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

func New(packageByNode map[int]int, diesByNode map[int]sets.Int, cpusByDie map[int][]int) (*MBAManager, error) {
	manager := &MBAManager{
		mbasByPackage: make(MBAPackage),
	}

	for node, dies := range diesByNode {
		packageID, ok := packageByNode[node]
		if !ok {
			return nil, errors.Errorf("invalid mba data: failed to locate package for numa node %d", node)
		}

		var nodeCPUS []int
		for die, _ := range dies {
			nodeCPUS = append(nodeCPUS, cpusByDie[die]...)
		}

		mba := &MBA{
			numaNode:       node,
			cpus:           nodeCPUS,
			sharingPackage: packageID,
		}

		if _, ok := manager.mbasByPackage[packageID]; !ok {
			manager.mbasByPackage[packageID] = make(map[int]*MBA)
		}
		manager.mbasByPackage[packageID][node] = mba
	}

	manager.packages = len(manager.mbasByPackage)

	return manager, nil
}
