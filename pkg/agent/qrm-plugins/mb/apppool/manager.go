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

package apppool

import (
	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type Manager struct {
	packages []PoolsPackage
}

func (m Manager) getPackage(packageID int) PoolsPackage {
	for _, p := range m.packages {
		if p.GetID() == packageID {
			return p
		}
	}
	return nil
}

func (m Manager) GetPackages() []PoolsPackage {
	return m.packages
}

func (m Manager) AddAppPool(nodes []int) (AppPool, error) {
	for _, p := range m.packages {
		if ap, err := p.AddAppPool(nodes); err == nil {
			return ap, nil
		}
	}

	return nil, errors.New("unable to add app pool")
}

func (m Manager) DeleteAppPool(pool AppPool) error {
	packageID := pool.GetPackageID()
	p := m.getPackage(packageID)
	return p.DeleteAppPool(pool)
}

// New creates an app pool/package manager
func New(dieTopology *machine.DieTopology) *Manager {
	packages := make([]PoolsPackage, dieTopology.Packages)
	for i := range packages {
		packages[i] = newPackage(i, dieTopology.NUMAsInPackage[i], dieTopology.DiesInNuma)
	}

	return &Manager{
		packages: packages,
	}
}
