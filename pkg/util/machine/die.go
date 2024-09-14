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

package machine

import "k8s.io/apimachinery/pkg/util/sets"

// DieTopology keeps the relationship of dies(CCDs), numa, package, and socket
type DieTopology struct {
	// "fake" numa is OS made sub-numa, access across some fake numa nodes could be as efficient as inside,
	// is they are really in one "real" numa domain (package)
	FakeNUMAEnabled bool // if the fake NUMA is configured on this server

	// number of CPU sockets on whole machine
	CPUSockets int

	// package is concept of "real" numa, within which the mem access is equally efficient. and outside much more inefficient
	// e.g. NPS2 makes 2 PACKAGES per CPU socket, NPS1 1 PACKAGES per socket,
	// whileas the (fake) numa nodes could be 4 or 8 per socket
	PackagesPerSocket int // number of pakcage ("real" numa") on one CPU socket, e.g. NPS1 it is 1
	Packages          int // number of "physical" NUMA domains on whole machine. it is CPUSockets * PackagesPerSocket
	PackagesInSocket  map[int][]int

	NUMAs          int           // os made sub numa node number
	NUMAsInPackage map[int][]int // mapping from Package to Numa nodes

	Dies       int           // number of die(CCD)s on whole machine
	DiesInNuma map[int][]int // mapping from Numa to CCDs

	DieSize   int           // how many cpu on a die(CCD)
	CPUsInDie map[int][]int // mapping from CCD to cpus
}

func NewDieTopology(siblingMap map[int]sets.Int) *DieTopology {
	topo := &DieTopology{}
	topo.NUMAsInPackage = GetNUMAsInPackage(siblingMap)

	// todo: stuff more info needed for mbm-poc

	return topo
}
