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

package numapackage

type MBAllocationMode string

type TaskType string

type UnitPhase string

const (
	MBAllocationModeHardPreempt MBAllocationMode = "hard-preempt"
	MBAllocationModeSoftAdjust  MBAllocationMode = "soft-adjust"

	TaskTypeSOCKET      = "socket"
	TaskTypeLowPriority = "low-priority"

	UnitPhaseAdmitted    = "admitted"
	UnitPhaseReserved    = "reserved"
	UnitPhaseRunning     = "running"
	UnitPhaseTerminating = "terminating"
)

type MBPackage interface {
	GetID() int
	GetMode() MBAllocationMode
	GetUnits() []MBUnit
}

// MBUnit is logical unit of mb allocation inside a package
// typical unit is one numa node, sometimes it could be multiple num nodes (at most 2 in POC phase)
type MBUnit interface {
	GetTaskType() TaskType
	GetLifeCyclePhase() UnitPhase
	GetMode() MBAllocationMode
	GetNUMANodes() []int
	GetCCDs() []int

	SetPhase(reserved UnitPhase)
}
