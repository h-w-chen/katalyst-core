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

package mbdomain

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type MBDomain struct {
	ID        int
	NumaNodes []int
	CCDNode   map[int]int
	NodeCCDs  map[int][]int
	CCDs      []int

	rwLock sync.RWMutex
	// numa nodes that will be assigned to dedicated pods that still are in Admit state
	PreemptyNodes sets.Int
	ccdIncubated  IncubatedCCDs

	incubationInterval time.Duration
}

func (m *MBDomain) String() string {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	var sb strings.Builder
	sb.WriteString("----- mb domain summary -----\n")
	sb.WriteString(fmt.Sprintf("    id: %d\n", m.ID))
	for _, node := range m.NumaNodes {
		sb.WriteString(fmt.Sprintf("    numa node: %d\n", node))
		for _, ccd := range m.NodeCCDs[node] {
			sb.WriteString(fmt.Sprintf("      ccd %d\n", ccd))
		}
	}

	return sb.String()
}

func (m *MBDomain) startIncubation(ccds sets.Int) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	for _, ccd := range m.CCDs {
		if ccds.Has(ccd) {
			m.ccdIncubated[ccd] = time.Now().Add(m.incubationInterval)
		}
	}
}

func (m *MBDomain) PreemptNodes(nodes []int) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	m.PreemptyNodes.Insert(nodes...)
}

func (m *MBDomain) UnpreemptNodes(nodes []int) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	for _, node := range nodes {
		delete(m.PreemptyNodes, node)
	}
}

func (m *MBDomain) GetPreemptingNodes() []int {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	return m.PreemptyNodes.List()
}

func (m *MBDomain) CleanseIncubates() {
	m.rwLock.Lock()
	m.rwLock.Unlock()

	for ccd, v := range m.ccdIncubated {
		if isIncubated(v) {
			delete(m.ccdIncubated, ccd)
		}
	}
}

func (m *MBDomain) CloneIncubates() IncubatedCCDs {
	m.rwLock.RLock()
	m.rwLock.RUnlock()

	clone := make(IncubatedCCDs)
	for ccd, v := range m.ccdIncubated {
		clone[ccd] = v
	}

	return clone
}

type MBDomainManager struct {
	Domains map[int]*MBDomain
}

func (m *MBDomainManager) StartIncubation(ccds []int) {
	dict := sets.NewInt(ccds...)
	for _, domain := range m.Domains {
		domain.startIncubation(dict)
	}
}

func NewMBDomainManager(dieTopology *machine.DieTopology, incubationInterval time.Duration) *MBDomainManager {
	manager := &MBDomainManager{
		Domains: make(map[int]*MBDomain),
	}

	for packageID := 0; packageID < dieTopology.Packages; packageID++ {
		mbDomain := &MBDomain{
			ID:                 packageID,
			NumaNodes:          dieTopology.NUMAsInPackage[packageID],
			CCDNode:            make(map[int]int),
			NodeCCDs:           make(map[int][]int),
			PreemptyNodes:      make(sets.Int),
			ccdIncubated:       make(IncubatedCCDs),
			incubationInterval: incubationInterval,
		}

		for node, ccds := range dieTopology.DiesInNuma {
			for ccd, _ := range ccds {
				mbDomain.CCDNode[ccd] = node
				mbDomain.NodeCCDs[node] = append(mbDomain.NodeCCDs[node], ccd)
				mbDomain.CCDs = append(mbDomain.CCDs, ccd)
			}
			sort.Slice(mbDomain.NodeCCDs[node], func(i, j int) bool {
				return mbDomain.NodeCCDs[node][i] < mbDomain.NodeCCDs[node][j]
			})
		}

		sort.Slice(mbDomain.CCDs, func(i, j int) bool {
			return mbDomain.CCDs[i] < mbDomain.CCDs[j]
		})

		manager.Domains[packageID] = mbDomain
		general.InfofV(6, "mbm: %s", mbDomain.String())
	}

	return manager
}
