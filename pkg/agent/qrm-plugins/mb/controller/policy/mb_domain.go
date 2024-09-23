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

package policy

import (
	"sort"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

const (
	domainTotalMB         = 120_000 //120 GBps in one mb sharing domain
	maxMBDedicatedPerNuma = 60_000  // if a socket pod assigned to one numa node, its max mb is 60 GB
	loungeMB              = 6_000   // lounge zone MB earmarked to dedicated qos is 6 GBps
)

type MBDomain struct {
	ID        int
	NumaNodes []int
	ccdNode   map[int]int
	nodeCCDs  map[int][]int
	CCDs      []int

	rwLock sync.RWMutex
	// numa nodes that will be assigned to dedicated pods that still are in Admit state
	preemptyNodes sets.Int
}

func (m *MBDomain) PreemptNodes(nodes []int) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	m.preemptyNodes.Insert(nodes...)
}

func (m *MBDomain) UnpreemptNodes(nodes []int) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	for _, node := range nodes {
		delete(m.preemptyNodes, node)
	}
}

type MBDomainManager struct {
	Domains map[int]*MBDomain
}

func NewMBDomainManager(dieTopology machine.DieTopology) *MBDomainManager {
	manager := &MBDomainManager{
		Domains: make(map[int]*MBDomain),
	}

	for packageID := 0; packageID < dieTopology.Packages; packageID++ {
		mbDomain := &MBDomain{
			ID:            packageID,
			NumaNodes:     dieTopology.NUMAsInPackage[packageID],
			ccdNode:       make(map[int]int),
			nodeCCDs:      make(map[int][]int),
			preemptyNodes: make(sets.Int),
		}

		for node, ccds := range dieTopology.DiesInNuma {
			for ccd, _ := range ccds {
				mbDomain.ccdNode[ccd] = node
				mbDomain.nodeCCDs[node] = append(mbDomain.nodeCCDs[node], ccd)
				mbDomain.CCDs = append(mbDomain.CCDs, ccd)
			}
			sort.Slice(mbDomain.nodeCCDs[node], func(i, j int) bool {
				return mbDomain.nodeCCDs[node][i] < mbDomain.nodeCCDs[node][j]
			})
		}

		sort.Slice(mbDomain.CCDs, func(i, j int) bool {
			return mbDomain.CCDs[i] < mbDomain.CCDs[j]
		})

		manager.Domains[packageID] = mbDomain
	}

	return manager
}
