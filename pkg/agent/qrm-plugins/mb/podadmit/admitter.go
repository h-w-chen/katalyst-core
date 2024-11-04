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
	"fmt"

	v1 "k8s.io/api/core/v1"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
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

func (m admitter) GetTopologyAwareResources(ctx context.Context, request *pluginapi.GetTopologyAwareResourcesRequest) (*pluginapi.GetTopologyAwareResourcesResponse, error) {
	general.InfofV(6, "mbm: pod admit is enquired with topology aware resource")
	return &pluginapi.GetTopologyAwareResourcesResponse{}, nil
}

func (m admitter) GetTopologyAwareAllocatableResources(ctx context.Context, request *pluginapi.GetTopologyAwareAllocatableResourcesRequest) (*pluginapi.GetTopologyAwareAllocatableResourcesResponse, error) {
	general.InfofV(6, "mbm: pod admit is enquired with allocatable resources")
	return &pluginapi.GetTopologyAwareAllocatableResourcesResponse{
		AllocatableResources: map[string]*pluginapi.AllocatableTopologyAwareResource{
			string(v1.ResourceMemory): {},
		},
	}, nil
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

	resp := &pluginapi.ResourceAllocationResponse{
		//PodUid:           request.PodUid,
		//PodNamespace:     request.PodNamespace,
		//PodName:          request.PodName,
		//PodRole:          request.PodRole,
		//PodType:          request.PodType,
		//ResourceName:     "mb-pod-admit",
		//AllocationResult: nil,
		//Labels:           general.DeepCopyMap(request.Labels),
		//Annotations:      general.DeepCopyMap(request.Annotations),
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
