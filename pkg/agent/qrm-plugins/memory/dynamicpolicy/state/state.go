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

package state

import (
	"encoding/json"
	"fmt"
	"sync"

	info "github.com/google/cadvisor/info/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/commonstate"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/util"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type AllocationInfo struct {
	commonstate.AllocationMeta `json:",inline"`

	AggregatedQuantity   uint64         `json:"aggregated_quantity"`
	NumaAllocationResult machine.CPUSet `json:"numa_allocation_result,omitempty"`

	// keyed by numa node id, value is assignment for the pod in corresponding NUMA node
	TopologyAwareAllocations map[int]uint64 `json:"topology_aware_allocations"`

	// keyed by control knob names referred in memory advisor package
	ExtraControlKnobInfo map[string]commonstate.ControlKnobInfo `json:"extra_control_knob_info"`
}

type (
	ContainerEntries   map[string]*AllocationInfo     // Keyed by container name
	PodEntries         map[string]ContainerEntries    // Keyed by pod UID
	PodResourceEntries map[v1.ResourceName]PodEntries // Keyed by resource name
)

// NUMANodeState records the amount of memory per numa node (in bytes)
type NUMANodeState struct {
	TotalMemSize   uint64     `json:"total"`
	SystemReserved uint64     `json:"systemReserved"`
	Allocatable    uint64     `json:"allocatable"`
	Allocated      uint64     `json:"Allocated"`
	Free           uint64     `json:"free"`
	PodEntries     PodEntries `json:"pod_entries"`
}

type (
	NUMANodeMap          map[int]*NUMANodeState          // keyed by numa node id
	NUMANodeResourcesMap map[v1.ResourceName]NUMANodeMap // keyed by resource name
)

func (ai *AllocationInfo) String() string {
	if ai == nil {
		return ""
	}

	contentBytes, err := json.Marshal(ai)
	if err != nil {
		klog.Errorf("[AllocationInfo.String] marshal AllocationInfo failed with error: %v", err)
		return ""
	}
	return string(contentBytes)
}

func (ai *AllocationInfo) Clone() *AllocationInfo {
	if ai == nil {
		return nil
	}

	clone := &AllocationInfo{
		AllocationMeta:       *ai.AllocationMeta.Clone(),
		AggregatedQuantity:   ai.AggregatedQuantity,
		NumaAllocationResult: ai.NumaAllocationResult.Clone(),
	}

	if ai.TopologyAwareAllocations != nil {
		clone.TopologyAwareAllocations = make(map[int]uint64)

		for node, quantity := range ai.TopologyAwareAllocations {
			clone.TopologyAwareAllocations[node] = quantity
		}
	}

	if ai.ExtraControlKnobInfo != nil {
		clone.ExtraControlKnobInfo = make(map[string]commonstate.ControlKnobInfo)

		for name := range ai.ExtraControlKnobInfo {
			clone.ExtraControlKnobInfo[name] = ai.ExtraControlKnobInfo[name]
		}
	}

	return clone
}

// GetResourceAllocation transforms resource allocation information into *pluginapi.ResourceAllocation
func (ai *AllocationInfo) GetResourceAllocation() (*pluginapi.ResourceAllocation, error) {
	if ai == nil {
		return nil, fmt.Errorf("GetResourceAllocation of nil AllocationInfo")
	}

	// deal with main resource
	resourceAllocation := &pluginapi.ResourceAllocation{
		ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
			string(v1.ResourceMemory): {
				OciPropertyName:   util.OCIPropertyNameCPUSetMems,
				IsNodeResource:    false,
				IsScalarResource:  true,
				AllocatedQuantity: float64(ai.AggregatedQuantity),
				AllocationResult:  ai.NumaAllocationResult.String(),
			},
		},
	}

	// deal with accompanying resources
	for name, entry := range ai.ExtraControlKnobInfo {
		if entry.OciPropertyName == "" {
			continue
		}

		if resourceAllocation.ResourceAllocation[name] != nil {
			return nil, fmt.Errorf("name: %s meets conflict", name)
		}

		resourceAllocation.ResourceAllocation[name] = &pluginapi.ResourceAllocationInfo{
			OciPropertyName:  entry.OciPropertyName,
			AllocationResult: entry.ControlKnobValue,
		}
	}

	return resourceAllocation, nil
}

func (pe PodEntries) Clone() PodEntries {
	if pe == nil {
		return nil
	}

	clone := make(PodEntries)
	for podUID, containerEntries := range pe {
		if containerEntries == nil {
			continue
		}

		clone[podUID] = make(ContainerEntries)
		for containerName, allocationInfo := range containerEntries {
			clone[podUID][containerName] = allocationInfo.Clone()
		}
	}
	return clone
}

// GetMainContainerAllocation returns AllocationInfo that belongs
// the main container for this pod
func (pe PodEntries) GetMainContainerAllocation(podUID string) (*AllocationInfo, bool) {
	for _, allocationInfo := range pe[podUID] {
		if allocationInfo.CheckMainContainer() {
			return allocationInfo, true
		}
	}
	return nil, false
}

func (pre PodResourceEntries) String() string {
	if pre == nil {
		return ""
	}

	contentBytes, err := json.Marshal(pre)
	if err != nil {
		klog.Errorf("[PodResourceEntries.String] marshal PodResourceEntries failed with error: %v", err)
		return ""
	}
	return string(contentBytes)
}

func (pre PodResourceEntries) Clone() PodResourceEntries {
	if pre == nil {
		return nil
	}

	clone := make(PodResourceEntries)
	for resourceName, podEntries := range pre {
		clone[resourceName] = podEntries.Clone()
	}
	return clone
}

func (ns *NUMANodeState) String() string {
	if ns == nil {
		return ""
	}

	contentBytes, err := json.Marshal(ns)
	if err != nil {
		klog.Errorf("[NUMANodeState.String] marshal NUMANodeState failed with error: %v", err)
		return ""
	}
	return string(contentBytes)
}

func (ns *NUMANodeState) Clone() *NUMANodeState {
	if ns == nil {
		return nil
	}

	return &NUMANodeState{
		TotalMemSize:   ns.TotalMemSize,
		SystemReserved: ns.SystemReserved,
		Allocatable:    ns.Allocatable,
		Allocated:      ns.Allocated,
		Free:           ns.Free,
		PodEntries:     ns.PodEntries.Clone(),
	}
}

// HasSharedOrDedicatedNUMABindingPods returns true if any AllocationInfo in this NUMANodeState is for shared or dedicated numa-binding
func (ns *NUMANodeState) HasSharedOrDedicatedNUMABindingPods() bool {
	if ns == nil {
		return false
	}

	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if allocationInfo != nil && allocationInfo.CheckSharedOrDedicatedNUMABinding() {
				return true
			}
		}
	}
	return false
}

// HasDedicatedNUMABindingAndNUMAExclusivePods returns true if any AllocationInfo in this NUMANodeState is for dedicated with numa-binding and
// numa-exclusive
func (ns *NUMANodeState) HasDedicatedNUMABindingAndNUMAExclusivePods() bool {
	if ns == nil {
		return false
	}

	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if allocationInfo != nil && allocationInfo.CheckDedicatedNUMABinding() &&
				allocationInfo.CheckNumaExclusive() {
				return true
			}
		}
	}
	return false
}

// HasReclaimedActualNUMABindingPods returns true if any AllocationInfo in this NUMANodeState is for reclaimed actual numa-binding
func (ns *NUMANodeState) HasReclaimedActualNUMABindingPods() bool {
	if ns == nil {
		return false
	}

	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if allocationInfo != nil && allocationInfo.CheckReclaimedActualNUMABinding() {
				return true
			}
		}
	}
	return false
}

// HasReclaimedNonActualNUMABindingPods returns true if any AllocationInfo in this NUMANodeState is for reclaimed non-actual numa-binding
func (ns *NUMANodeState) HasReclaimedNonActualNUMABindingPods() bool {
	if ns == nil {
		return false
	}

	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if allocationInfo != nil && allocationInfo.CheckReclaimedNonActualNUMABinding() {
				return true
			}
		}
	}
	return false
}

func (ns *NUMANodeState) GetNonActualNUMABindingAvailableHeadroom(numaHeadroom int64) int64 {
	res := numaHeadroom
	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if allocationInfo.CheckReclaimedActualNUMABinding() {
				return 0
			}
		}
	}
	return res
}

// ExistMatchedAllocationInfo returns true if the stated predicate holds true for some pods of this numa else it returns false.
func (ns *NUMANodeState) ExistMatchedAllocationInfo(f func(ai *AllocationInfo) bool) bool {
	for _, containerEntries := range ns.PodEntries {
		for _, allocationInfo := range containerEntries {
			if f(allocationInfo) {
				return true
			}
		}
	}

	return false
}

// SetAllocationInfo adds a new AllocationInfo (for pod/container pairs) into the given NUMANodeState
func (ns *NUMANodeState) SetAllocationInfo(podUID string, containerName string, allocationInfo *AllocationInfo) {
	if ns == nil {
		return
	}

	if ns.PodEntries == nil {
		ns.PodEntries = make(PodEntries)
	}

	if _, ok := ns.PodEntries[podUID]; !ok {
		ns.PodEntries[podUID] = make(ContainerEntries)
	}

	ns.PodEntries[podUID][containerName] = allocationInfo.Clone()
}

func (nm NUMANodeMap) Clone() NUMANodeMap {
	clone := make(NUMANodeMap)
	for node, ns := range nm {
		clone[node] = ns.Clone()
	}
	return clone
}

// BytesPerNUMA is a helper function to parse memory capacity at per numa level
func (nm NUMANodeMap) BytesPerNUMA() (uint64, error) {
	if len(nm) == 0 {
		return 0, fmt.Errorf("getBytesPerNUMAFromMachineState got nil numaMap")
	}

	var maxNUMAAllocatable uint64
	for _, numaState := range nm {
		if numaState != nil {
			maxNUMAAllocatable = general.MaxUInt64(maxNUMAAllocatable, numaState.Allocatable)
		}
	}

	if maxNUMAAllocatable > 0 {
		return maxNUMAAllocatable, nil
	}

	return 0, fmt.Errorf("getBytesPerNUMAFromMachineState doesn't get valid numaState")
}

// GetNUMANodesWithoutSharedOrDedicatedNUMABindingPods returns a set of numa nodes; for
// those numa nodes, they all don't contain shared or dedicated numa-binding pods
func (nm NUMANodeMap) GetNUMANodesWithoutSharedOrDedicatedNUMABindingPods() machine.CPUSet {
	res := machine.NewCPUSet()
	for numaId, numaNodeState := range nm {
		if numaNodeState != nil && !numaNodeState.HasSharedOrDedicatedNUMABindingPods() {
			res = res.Union(machine.NewCPUSet(numaId))
		}
	}
	return res
}

// GetNUMANodesWithoutDedicatedNUMABindingAndNUMAExclusivePods returns a set of numa nodes; for
// those numa nodes, they all don't contain dedicated with numa-binding and numa-exclusive pods
func (nm NUMANodeMap) GetNUMANodesWithoutDedicatedNUMABindingAndNUMAExclusivePods() machine.CPUSet {
	res := machine.NewCPUSet()
	for numaId, numaNodeState := range nm {
		if numaNodeState != nil && !numaNodeState.HasDedicatedNUMABindingAndNUMAExclusivePods() {
			res = res.Union(machine.NewCPUSet(numaId))
		}
	}
	return res
}

// GetNUMANodesWithoutReclaimedActualNUMABindingPods returns a set of numa nodes; for
// those numa nodes, they all don't contain reclaimed actual numa binding pods
func (nm NUMANodeMap) GetNUMANodesWithoutReclaimedActualNUMABindingPods() machine.CPUSet {
	res := machine.NewCPUSet()
	for numaId, numaNodeState := range nm {
		if numaNodeState != nil && !numaNodeState.HasReclaimedActualNUMABindingPods() {
			res = res.Union(machine.NewCPUSet(numaId))
		}
	}
	return res
}

// GetNUMANodesWithoutReclaimedNonActualNUMABindingPods returns a set of numa nodes; for
// those numa nodes, they all don't contain reclaimed non-actual numa binding pods
func (nm NUMANodeMap) GetNUMANodesWithoutReclaimedNonActualNUMABindingPods() machine.CPUSet {
	res := machine.NewCPUSet()
	for numaId, numaNodeState := range nm {
		if numaNodeState != nil && !numaNodeState.HasReclaimedNonActualNUMABindingPods() {
			res = res.Union(machine.NewCPUSet(numaId))
		}
	}
	return res
}

func (nm NUMANodeMap) GetNonActualNUMABindingAvailableHeadroom(numaHeadroom map[int]int64) int64 {
	res := int64(0)
	for id, numaNodeState := range nm {
		res += numaNodeState.GetNonActualNUMABindingAvailableHeadroom(numaHeadroom[id])
	}
	return res
}

func (nrm NUMANodeResourcesMap) String() string {
	if nrm == nil {
		return ""
	}

	contentBytes, err := json.Marshal(nrm)
	if err != nil {
		klog.Errorf("[NUMANodeResourcesMap.String] marshal NUMANodeResourcesMap failed with error: %v", err)
		return ""
	}
	return string(contentBytes)
}

func (nrm NUMANodeResourcesMap) Clone() NUMANodeResourcesMap {
	clone := make(NUMANodeResourcesMap)
	for resourceName, nm := range nrm {
		clone[resourceName] = nm.Clone()
	}
	return clone
}

// reader is used to get information from local states
type reader interface {
	GetMachineState() NUMANodeResourcesMap
	GetNUMAHeadroom() map[int]int64
	GetPodResourceEntries() PodResourceEntries
	GetAllocationInfo(resourceName v1.ResourceName, podUID, containerName string) *AllocationInfo
}

// writer is used to store information into local states,
// and it also provides functionality to maintain the local files
type writer interface {
	SetMachineState(numaNodeResourcesMap NUMANodeResourcesMap, persist bool)
	SetNUMAHeadroom(m map[int]int64, persist bool)
	SetPodResourceEntries(podResourceEntries PodResourceEntries, persist bool)
	SetAllocationInfo(resourceName v1.ResourceName, podUID, containerName string, allocationInfo *AllocationInfo, persist bool)

	Delete(resourceName v1.ResourceName, podUID, containerName string, persist bool)
	ClearState()
	StoreState() error
}

// ReadonlyState interface only provides methods for tracking pod assignments
type ReadonlyState interface {
	reader

	GetMachineInfo() *info.MachineInfo
	GetReservedMemory() map[v1.ResourceName]map[int]uint64
}

// State interface provides methods for tracking and setting pod assignments
type State interface {
	writer
	ReadonlyState
}

var (
	readonlyStateLock sync.RWMutex
	readonlyState     ReadonlyState
)

// GetReadonlyState retrieves the readonlyState in a thread-safe manner.
// Returns an error if readonlyState is not set.
func GetReadonlyState() (ReadonlyState, error) {
	readonlyStateLock.RLock()
	defer readonlyStateLock.RUnlock()

	if readonlyState == nil {
		return nil, fmt.Errorf("readonlyState isn't setted")
	}
	return readonlyState, nil
}

// SetReadonlyState updates the readonlyState in a thread-safe manner.
func SetReadonlyState(state ReadonlyState) {
	readonlyStateLock.Lock()
	defer readonlyStateLock.Unlock()

	readonlyState = state
}

var (
	readWriteStateLock sync.RWMutex
	readWriteState     State
)

// GetReadWriteState retrieves the readWriteState in a thread-safe manner.
// Returns an error if readWriteState is not set.
func GetReadWriteState() (State, error) {
	readWriteStateLock.RLock()
	defer readWriteStateLock.RUnlock()

	if readWriteState == nil {
		return nil, fmt.Errorf("readWriteState isn't set")
	}
	return readWriteState, nil
}

// SetReadWriteState updates the readWriteState in a thread-safe manner.
func SetReadWriteState(state State) {
	readWriteStateLock.Lock()
	defer readWriteStateLock.Unlock()

	readWriteState = state
}
