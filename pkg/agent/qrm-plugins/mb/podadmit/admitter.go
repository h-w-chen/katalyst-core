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

package podadmit

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/cpu/dynamicpolicy/state"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/util"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type admitter struct {
	pluginapi.UnimplementedResourcePluginServer
	qosConfig *generic.QoSConfiguration
	//domainManager *mbdomain.MBDomainManager
	//mbController  *controller.Controller
	//taskManager   task.Manager

	nodePreempter *NodePreempter
}

func (m *admitter) GetTopologyAwareResources(ctx context.Context, req *pluginapi.GetTopologyAwareResourcesRequest) (*pluginapi.GetTopologyAwareResourcesResponse, error) {
	//general.InfofV(6, "mbm: pod admit is enquired with topology aware resource, pod uid %v, container %v", req.PodUid, req.ContainerName)
	return &pluginapi.GetTopologyAwareResourcesResponse{
		PodUid: req.PodUid,
		ContainerTopologyAwareResources: &pluginapi.ContainerTopologyAwareResources{
			ContainerName: req.ContainerName,
			AllocatedResources: map[string]*pluginapi.TopologyAwareResource{
				string(v1.ResourceMemory): {
					IsNodeResource:   false,
					IsScalarResource: true,
				},
			},
		},
	}, nil
}

func (m *admitter) GetTopologyAwareAllocatableResources(ctx context.Context, request *pluginapi.GetTopologyAwareAllocatableResourcesRequest) (*pluginapi.GetTopologyAwareAllocatableResourcesResponse, error) {
	//general.InfofV(6, "mbm: pod admit is enquired with allocatable resources: %v", request.String())
	return &pluginapi.GetTopologyAwareAllocatableResourcesResponse{
		AllocatableResources: map[string]*pluginapi.AllocatableTopologyAwareResource{
			string(v1.ResourceMemory): {
				IsNodeResource:   false,
				IsScalarResource: true,
			},
		},
	}, nil
}

func (m *admitter) RemovePod(ctx context.Context, req *pluginapi.RemovePodRequest) (*pluginapi.RemovePodResponse, error) {
	general.InfofV(6, "mbm: resource RemovePod - pod uid %s", req.PodUid)
	return &pluginapi.RemovePodResponse{}, nil
}

// todo: generalize to check for any shared_xx
func IsBatchPod(qosLevel string, anno map[string]string) bool {
	if qosLevel != apiconsts.PodAnnotationQoSLevelSharedCores {
		return false
	}

	// shared_xx 优先级是跟 pool name对齐
	if enhancementValue, ok := anno[apiconsts.PodAnnotationCPUEnhancementKey]; ok {
		flattenedEnhancements := map[string]string{}
		err := json.Unmarshal([]byte(enhancementValue), &flattenedEnhancements)
		if err != nil {
			return false
		}

		if pool := state.GetSpecifiedPoolName(qosLevel, flattenedEnhancements[apiconsts.PodAnnotationCPUEnhancementCPUSet]); pool == "shared-30" {
			return true
		}
	}

	return false
}

func IsSocketPod(qosLevel string, annotations map[string]string) bool {
	if v, ok := annotations["instance-model"]; ok {
		return qosLevel == apiconsts.PodAnnotationQoSLevelDedicatedCores && len(v) > 0
	}

	return false
}

//func (m *admitter) getNotInUseNodes(nodes []uint64) []int {
//	var notInUses []int
//	inUses := m.taskManager.GetNumaNodesInUse()
//	for _, node := range nodes {
//		if inUses.Has(int(node)) {
//			continue
//		}
//
//		notInUses = append(notInUses, int(node))
//	}
//
//	return notInUses
//}

//func (m *admitter) PreemptNodes(req *pluginapi.ResourceRequest) error {
//	general.InfofV(6, "mbm: preempt nodes for pod %s/%s", req.PodNamespace, req.PodName)
//
//	if req.Hint != nil {
//		if len(req.Hint.Nodes) == 0 {
//			return fmt.Errorf("hint is empty")
//		}
//	}
//
//	// check numa nodes' in-use state; only preempt those not-in-use yet
//	nodesToPreempt := m.getNotInUseNodes(req.Hint.Nodes)
//	if len(nodesToPreempt) > 0 {
//		if m.domainManager.PreemptNodes(nodesToPreempt) {
//			// requests to adjust mb ASAP for new preemption if there are any changes
//			m.mbController.ReqToAdjustMB()
//		}
//	}
//
//	return nil
//}

func (m *admitter) Allocate(ctx context.Context, req *pluginapi.ResourceRequest) (*pluginapi.ResourceAllocationResponse, error) {
	general.InfofV(6, "mbm: resource allocate - pod admitting %s/%s, uid %s", req.PodNamespace, req.PodName, req.PodUid)
	qosLevel, err := m.qosConfig.GetQoSLevel(nil, req.Annotations)
	if err != nil {
		return nil, err
	}

	if IsSocketPod(qosLevel, req.Annotations) {
		general.InfofV(6, "mbm: resource allocate - identified socket pod %s/%s", req.PodNamespace, req.PodName)

		if err := m.nodePreempter.PreemptNodes(req); err != nil {
			return nil, errors.Wrap(err, "failed to preempt socket pod request")
		}
	}

	// 0 on error; no need to handle error explicitly
	reqInt, _, _ := util.GetQuantityFromResourceReq(req)
	allocateInfo := &pluginapi.ResourceAllocationInfo{
		IsNodeResource:    false,
		IsScalarResource:  true,
		AllocatedQuantity: float64(reqInt),
		Annotations:       general.DeepCopyMap(req.Annotations),
	}

	if req.Hint != nil {
		allocateInfo.ResourceHints = &pluginapi.ListOfTopologyHints{
			Hints: []*pluginapi.TopologyHint{
				req.Hint,
			},
		}
	}

	if IsBatchPod(qosLevel, req.Annotations) {
		general.InfofV(6, "mbm: resource allocate - pod admitting %s/%s, shared-30", req.PodNamespace, req.PodName)
		if allocateInfo.Annotations == nil {
			allocateInfo.Annotations = make(map[string]string)
		}
		allocateInfo.Annotations["rdt.resources.beta.kubernetes.io/pod"] = "shared-30"
	}

	resp := &pluginapi.ResourceAllocationResponse{
		PodUid:       req.PodUid,
		PodNamespace: req.PodNamespace,
		PodName:      req.PodName,
		PodRole:      req.PodRole,
		PodType:      req.PodType,
		ResourceName: "memory",
		AllocationResult: &pluginapi.ResourceAllocation{
			ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
				"memory": allocateInfo,
			},
		},
		Labels:      general.DeepCopyMap(req.Labels),
		Annotations: general.DeepCopyMap(req.Annotations),
	}

	return resp, nil
}

func (m *admitter) GetResourcePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.ResourcePluginOptions, error) {
	general.InfofV(6, "mbm: pod admit is enquired with options")
	return &pluginapi.ResourcePluginOptions{
		PreStartRequired:      false,
		WithTopologyAlignment: false,
		NeedReconcile:         false,
	}, nil
}

var _ pluginapi.ResourcePluginServer = (*admitter)(nil)
