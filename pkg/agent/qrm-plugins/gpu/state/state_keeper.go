package state

import (
	gpuconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/gpu/consts"
	v1 "k8s.io/api/core/v1"
)

func GetReadOnlyState() AllocationResourcesMap {
	return gAllocationResourcesMap
}

func GetGPUAllocations() map[string]string {
	// returns {'GPU-fef8089b-4820-abfc-e83e-94318197576e':'default/pod-name-1'}

	gpuDeviceState := gAllocationResourcesMap[v1.ResourceName(gpuconsts.GPUDeviceType)]
	result := map[string]string{}
	for gpuID, allocState := range gpuDeviceState {
		if allocState == nil {
			continue
		}

		// todo: how to handle multiple pod allocated to one gpu?
		for podId, _ := range allocState.PodEntries {
			result[gpuID] = podId
			break
		}
	}

	return result
}
