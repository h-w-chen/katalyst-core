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

type Manager struct {
	packages []PoolsPackage
}

func (m Manager) GetPackage(packageID int) PoolsPackage {
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
	panic("impl")
}

func (m Manager) DeleteAppPool(pool AppPool) error {
	panic("impl")
}

// New creates an app pool/package manager
func New(numPackage int) *Manager {
	packages := make([]PoolsPackage, numPackage)
	for i := range packages {
		packages[i] = newPackage(i)
	}

	return &Manager{
		packages: packages,
	}
}
