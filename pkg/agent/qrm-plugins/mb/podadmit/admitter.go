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
	"fmt"
	v1 "k8s.io/api/core/v1"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/cpu/dynamicpolicy/state"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/util"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type admitter struct {
	pluginapi.UnimplementedResourcePluginServer
	qosConfig     *generic.QoSConfiguration
	domainManager *mbdomain.MBDomainManager
	mbController  *controller.Controller
	taskManager   task.Manager
}

func (m admitter) GetTopologyAwareResources(ctx context.Context, req *pluginapi.GetTopologyAwareResourcesRequest) (*pluginapi.GetTopologyAwareResourcesResponse, error) {
	general.InfofV(6, "mbm: pod admit is enquired with topology aware resource, pod uid %v, container %v", req.PodUid, req.ContainerName)
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

func (m admitter) GetTopologyAwareAllocatableResources(ctx context.Context, request *pluginapi.GetTopologyAwareAllocatableResourcesRequest) (*pluginapi.GetTopologyAwareAllocatableResourcesResponse, error) {
	general.InfofV(6, "mbm: pod admit is enquired with allocatable resources: %v", request.String())
	return &pluginapi.GetTopologyAwareAllocatableResourcesResponse{
		AllocatableResources: map[string]*pluginapi.AllocatableTopologyAwareResource{
			string(v1.ResourceMemory): {
				IsNodeResource:   false,
				IsScalarResource: true,
			},
		},
	}, nil
}

// todo: find better way to check for offline shared_cores, e.g.
// leveraging KatalystQoSLevelAnnotationKey     = "katalyst.kubewharf.io/qos_level"
//            KatalystNumaBindingAnnotationKey  = "katalyst.kubewharf.io/numa_binding"
func cloneAugmentedAnnotation(qosLevel string, anno map[string]string) map[string]string {
	//	enhancementKVs := c.GetQoSEnhancementKVs(pod, expandedAnnotations, )
	clone := general.DeepCopyMap(anno)
	if qosLevel != apiconsts.PodAnnotationQoSLevelSharedCores {
		return clone
	}

	if enhancementValue, ok := anno[apiconsts.PodAnnotationCPUEnhancementKey]; ok {
		flattenedEnhancements := map[string]string{}
		err := json.Unmarshal([]byte(enhancementValue), &flattenedEnhancements)
		if err != nil {
			return clone
		}
		if pool := state.GetSpecifiedPoolName(qosLevel, flattenedEnhancements[apiconsts.PodAnnotationCPUEnhancementCPUSet]); pool == "batch" {
			clone["rdt.resources.beta.kubernetes.io/pod"] = "shared-30"
		}
	}
	return clone
}

func (m admitter) Allocate(ctx context.Context, req *pluginapi.ResourceRequest) (*pluginapi.ResourceAllocationResponse, error) {
	general.InfofV(6, "mbm: resource allocate - pod admitting %s/%s, uid %s", req.PodNamespace, req.PodName, req.PodUid)
	qosLevel, err := m.qosConfig.GetQoSLevel(nil, req.Annotations)
	if err != nil {
		return nil, err
	}

	if req.ContainerType == pluginapi.ContainerType_SIDECAR {
		// sidecar container admit after main container
		general.InfofV(6, "mbm: resource allocate sidecar container - pod admitting %s/%s, uid %s", req.PodNamespace, req.PodName, req.PodUid)
	} else if qosLevel == apiconsts.PodAnnotationQoSLevelDedicatedCores {
		if req.Hint != nil {
			if len(req.Hint.Nodes) == 0 {
				return nil, fmt.Errorf("hint is empty")
			}

			general.InfofV(6, "mbm: identified socket pod %s/%s", req.PodNamespace, req.PodName)

			var nodesToPreempt []int
			// check numa nodes' in-use state; only preempt those not-in-use yet
			inUses := m.taskManager.GetNumaNodesInUse()
			for _, node := range req.Hint.Nodes {
				if inUses.Has(int(node)) {
					continue
				}

				nodesToPreempt = append(nodesToPreempt, int(node))
			}

			if len(nodesToPreempt) > 0 {
				if m.domainManager.PreemptNodes(nodesToPreempt) {
					// requests to adjust mb ASAP for new preemption if there are any changes
					m.mbController.ReqToAdjustMB()
				}
			}
		}
	}

	// 0 on error; no need to handle error explicitly
	reqInt, _, _ := util.GetQuantityFromResourceReq(req)

	resp := &pluginapi.ResourceAllocationResponse{
		PodUid:       req.PodUid,
		PodNamespace: req.PodNamespace,
		PodName:      req.PodName,
		PodRole:      req.PodRole,
		PodType:      req.PodType,
		ResourceName: "memory",
		AllocationResult: &pluginapi.ResourceAllocation{
			ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
				"memory": {
					IsNodeResource:    false,
					IsScalarResource:  true,
					Annotations:       cloneAugmentedAnnotation(qosLevel, req.Annotations),
					AllocatedQuantity: float64(reqInt),
					//ResourceHints: &pluginapi.ListOfTopologyHints{
					//	Hints: []*pluginapi.TopologyHint{
					//		req.Hint,
					//	},
					//},
				},
			},
		},
		Labels:      general.DeepCopyMap(req.Labels),
		Annotations: cloneAugmentedAnnotation(qosLevel, req.Annotations),
	}

	return resp, nil
}

func (m admitter) GetResourcePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.ResourcePluginOptions, error) {
	general.InfofV(6, "mbm: pod admit is enquired with options")
	return &pluginapi.ResourcePluginOptions{
		PreStartRequired:      false,
		WithTopologyAlignment: false,
		NeedReconcile:         false,
	}, nil
}

var _ pluginapi.ResourcePluginServer = (*admitter)(nil)
