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

// AppPool is logical unit of mb allocation inside a package
// typical pool is one numa node hosting multiple pods or devoted to single Socket pod,
// sometimes it could be multiple numa nodes (at most 2 in POC phase)
// app pool is named for it hosts a pool of applications
// attention: not yet support the special case that socket pod occupying numa nodes cross packages
type AppPool interface {
	GetPackageID() int
	GetTaskType() TaskType
	GetLifeCyclePhase() UnitPhase
	GetMode() MBAllocationMode
	GetNUMANodes() []int
	GetCCDs() map[int][]int

	SetPhase(reserved UnitPhase)
}

// todo: revisit for concurrency potential?
type appPool struct {
	packageID int
	numaNodes []int
	ccds      map[int][]int

	appType TaskType
	mode    MBAllocationMode
	phase   UnitPhase
}

func (a appPool) GetPackageID() int {
	return a.packageID
}

func (a appPool) GetTaskType() TaskType {
	return a.appType
}

func (a appPool) GetLifeCyclePhase() UnitPhase {
	return a.phase
}

func (a appPool) GetMode() MBAllocationMode {
	return a.mode
}

func (a appPool) GetNUMANodes() []int {
	return a.numaNodes
}

func (a appPool) GetCCDs() map[int][]int {
	return a.ccds
}

func (a appPool) SetPhase(phase UnitPhase) {
	a.phase = phase
}

var _ AppPool = &appPool{}
