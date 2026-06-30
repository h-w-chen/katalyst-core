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
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"

	gpuconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/gpu/consts"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type PodFetcher interface {
	GetPod(ctx context.Context, podUID string) (*v1.Pod, error)
}

// gpuPluginState is an in-memory implementation of State;
// everytime we want to read or write states, those requests will always
// go to in-memory State, and then go to disk State, i.e. in write-back mode
type gpuPluginState struct {
	sync.RWMutex

	qrmConf                        *qrm.QRMPluginsConfiguration
	defaultResourceStateGenerators *DefaultResourceStateGeneratorRegistry
	podFetcher                     PodFetcher

	machineState       AllocationResourcesMap
	podResourceEntries PodResourceEntries

	machineStateSyncNotifiers []func()
}

func NewGPUPluginState(
	conf *qrm.QRMPluginsConfiguration,
	resourceStateGeneratorRegistry *DefaultResourceStateGeneratorRegistry,
	podFetchers ...PodFetcher,
) (State, error) {
	generalLog.InfoS("initializing new gpu plugin in-memory state store")

	defaultMachineState, err := GenerateMachineState(resourceStateGeneratorRegistry)
	if err != nil {
		return nil, fmt.Errorf("GenerateMachineState failed with error: %w", err)
	}

	return &gpuPluginState{
		qrmConf:                        conf,
		machineState:                   defaultMachineState,
		defaultResourceStateGenerators: resourceStateGeneratorRegistry,
		podFetcher:                     firstPodFetcher(podFetchers),
		podResourceEntries:             make(PodResourceEntries),
	}, nil
}

func firstPodFetcher(podFetchers []PodFetcher) PodFetcher {
	for _, podFetcher := range podFetchers {
		if podFetcher != nil {
			return podFetcher
		}
	}
	return nil
}

func (s *gpuPluginState) AddMachineStateSyncNotifier(notifier func()) {
	s.Lock()
	defer s.Unlock()
	s.machineStateSyncNotifiers = append(s.machineStateSyncNotifiers, notifier)
}

func (s *gpuPluginState) SetMachineState(allocationResourcesMap AllocationResourcesMap, _ bool) {
	s.Lock()
	s.machineState = allocationResourcesMap.Clone()
	generalLog.InfoS("updated gpu plugin machine state",
		"GPUMap", allocationResourcesMap.String())

	general.Infof("chw-debug: gpu state: set machine state")
	gpuDevState, ok := s.machineState[gpuconsts.GPUDeviceType]
	if !ok {
		general.Infof("chw-debug: gpu state: no gpu device state")
	}
	for devID, allocState := range gpuDevState {
		podEntries := allocState.PodEntries
		for podID, contEntries := range podEntries {
			if s.podFetcher != nil {
				pod, err := s.podFetcher.GetPod(context.Background(), podID)
				if err != nil {
					general.Warningf("chw-debug: get pod failed: pod %v, err=%v", podID, err)
				} else if pod != nil {
					general.Infof("chw-debug: pod labels: pod %v, labels=%v, annotaions=%v", podID, pod.Labels, pod.Annotations)
				}
			}

			for contID, allocInfo := range contEntries {
				general.Infof("chw-debug: gpu state: dev %v, pod %v, container %v, role=%v, labels=%v, annotaions=%v", devID, podID, contID,
					allocInfo.AllocationMeta.PodRole,
					allocInfo.AllocationMeta.Labels,
					allocInfo.AllocationMeta.Annotations)
			}
		}
	}

	var notifiers []func()
	notifiers = append(notifiers, s.machineStateSyncNotifiers...)
	s.Unlock()

	// Invoke notifiers outside the lock to avoid potential deadlocks
	for _, notifier := range notifiers {
		notifier()
	}
}

func (s *gpuPluginState) SetResourceState(resourceName v1.ResourceName, allocationMap AllocationMap, _ bool) {
	s.Lock()
	s.machineState[resourceName] = allocationMap.Clone()
	generalLog.InfoS("updated gpu plugin resource state",
		"resourceName", resourceName,
		"allocationMap", allocationMap.String())

	general.Infof("chw-debug: SetResourceState called, resourceName=%v", resourceName)
	for devID, allocState := range allocationMap {
		general.Infof("chw-debug: SetResourceState dev=%v, allocatable=%v, podCount=%v",
			devID, allocState.Allocatable, len(allocState.PodEntries))
		for podUID, contEntries := range allocState.PodEntries {
			// try to fetch pod labels/annotations
			if s.podFetcher != nil {
				pod, err := s.podFetcher.GetPod(context.Background(), podUID)
				if err != nil {
					general.Warningf("chw-debug: SetResourceState get pod failed: pod=%v, err=%v", podUID, err)
				} else if pod != nil {
					general.Infof("chw-debug: SetResourceState pod=%v, labels=%v, annotations=%v", podUID, pod.Labels, pod.Annotations)
				} else {
					general.Warningf("chw-debug: SetResourceState pod not found: pod=%v", podUID)
				}
			}
			for contName, allocInfo := range contEntries {
				general.Infof("chw-debug: SetResourceState dev=%v, pod=%v, container=%v, deviceName=%v, quantity=%v, NUMA=%v, role=%v, labels=%v, annotations=%v, topologyAware=%v",
					devID, podUID, contName,
					allocInfo.DeviceName,
					allocInfo.AllocatedAllocation.Quantity,
					allocInfo.AllocatedAllocation.NUMANodes,
					allocInfo.AllocationMeta.PodRole,
					allocInfo.AllocationMeta.Labels,
					allocInfo.AllocationMeta.Annotations,
					len(allocInfo.TopologyAwareAllocations))
			}
		}
	}

	var notifiers []func()
	notifiers = append(notifiers, s.machineStateSyncNotifiers...)
	s.Unlock()

	// Invoke notifiers outside the lock to avoid potential deadlocks
	for _, notifier := range notifiers {
		notifier()
	}
}

func (s *gpuPluginState) SetPodResourceEntries(podResourceEntries PodResourceEntries, _ bool) {
	s.Lock()
	defer s.Unlock()
	s.podResourceEntries = podResourceEntries.Clone()
}

func (s *gpuPluginState) SetAllocationInfo(
	resourceName v1.ResourceName, podUID, containerName string, allocationInfo *AllocationInfo, _ bool,
) {
	s.Lock()
	defer s.Unlock()

	isNewPod := false
	isNewContainer := false
	if _, ok := s.podResourceEntries[resourceName]; !ok {
		s.podResourceEntries[resourceName] = make(PodEntries)
	}
	if _, ok := s.podResourceEntries[resourceName][podUID]; !ok {
		s.podResourceEntries[resourceName][podUID] = make(ContainerEntries)
		isNewPod = true
	} else if _, ok := s.podResourceEntries[resourceName][podUID][containerName]; !ok {
		isNewContainer = true
	}

	s.podResourceEntries[resourceName][podUID][containerName] = allocationInfo.Clone()
	generalLog.InfoS("updated gpu plugin pod resource entries",
		"podUID", podUID,
		"containerName", containerName,
		"allocationInfo", allocationInfo.String())

	general.Infof("chw-debug: SetAllocationInfo called, resourceName=%v, podUID=%v, container=%v, isNewPod=%v, isNewContainer=%v",
		resourceName, podUID, containerName, isNewPod, isNewContainer)
	general.Infof("chw-debug: SetAllocationInfo deviceName=%v, quantity=%v, NUMA=%v, role=%v, labels=%v, annotations=%v, topologyAwareCount=%v",
		allocationInfo.DeviceName,
		allocationInfo.AllocatedAllocation.Quantity,
		allocationInfo.AllocatedAllocation.NUMANodes,
		allocationInfo.AllocationMeta.PodRole,
		allocationInfo.AllocationMeta.Labels,
		allocationInfo.AllocationMeta.Annotations,
		len(allocationInfo.TopologyAwareAllocations))
	for topoKey, topoAlloc := range allocationInfo.TopologyAwareAllocations {
		general.Infof("chw-debug: SetAllocationInfo topologyAware[%v]: quantity=%v, NUMA=%v",
			topoKey, topoAlloc.Quantity, topoAlloc.NUMANodes)
	}

	// try to fetch real pod labels/annotations from MetaServer
	if s.podFetcher != nil {
		pod, err := s.podFetcher.GetPod(context.Background(), podUID)
		if err != nil {
			general.Warningf("chw-debug: SetAllocationInfo get pod failed: pod=%v, err=%v", podUID, err)
		} else if pod != nil {
			general.Infof("chw-debug: SetAllocationInfo real pod labels=%v, annotations=%v", pod.Labels, pod.Annotations)
		} else {
			general.Warningf("chw-debug: SetAllocationInfo pod not found: pod=%v", podUID)
		}
	}
}

func (s *gpuPluginState) Delete(resourceName v1.ResourceName, podUID, containerName string, _ bool) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.podResourceEntries[resourceName]; !ok {
		return
	}

	if _, ok := s.podResourceEntries[resourceName][podUID]; !ok {
		return
	}

	delete(s.podResourceEntries[resourceName][podUID], containerName)
	if len(s.podResourceEntries[resourceName][podUID]) == 0 {
		delete(s.podResourceEntries[resourceName], podUID)
	}

	generalLog.InfoS("deleted container entry", "podUID", podUID, "containerName", containerName)
}

func (s *gpuPluginState) ClearState() {
	s.Lock()

	machineState, err := GenerateMachineState(s.defaultResourceStateGenerators)
	if err != nil {
		generalLog.ErrorS(err, "failed to generate machine state")
	}
	s.machineState = machineState
	s.podResourceEntries = make(PodResourceEntries)

	generalLog.InfoS("cleared state")

	var notifiers []func()
	notifiers = append(notifiers, s.machineStateSyncNotifiers...)
	s.Unlock()

	// Invoke notifiers outside the lock to avoid potential deadlocks
	for _, notifier := range notifiers {
		notifier()
	}
}

func (s *gpuPluginState) StoreState() error {
	// nothing to do
	return nil
}

func (s *gpuPluginState) GetMachineState() AllocationResourcesMap {
	s.RLock()
	defer s.RUnlock()

	return s.machineState.Clone()
}

func (s *gpuPluginState) GetPodResourceEntries() PodResourceEntries {
	s.RLock()
	defer s.RUnlock()

	return s.podResourceEntries.Clone()
}

func (s *gpuPluginState) GetPodEntries(resourceName v1.ResourceName) PodEntries {
	s.RLock()
	defer s.RUnlock()

	return s.podResourceEntries[resourceName].Clone()
}

func (s *gpuPluginState) GetAllocationInfo(resourceName v1.ResourceName, podUID, containerName string) *AllocationInfo {
	s.RLock()
	defer s.RUnlock()

	if res, ok := s.podResourceEntries[resourceName][podUID][containerName]; ok {
		return res.Clone()
	}

	return nil
}
