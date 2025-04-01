package mongroups

import (
	"testing"

	"github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/stretchr/testify/assert"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/util"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/config/agent"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/qrm"
)

func TestManager_Allocate(t *testing.T) {
	cases := []struct {
		name        string
		policy      string
		qosLevel    string
		annotations map[string]string
		expectResp  *pluginapi.ResourceAllocationResponse
	}{
		{
			name:     "empty policy",
			policy:   "",
			qosLevel: "dedicated_cores",
			expectResp: &pluginapi.ResourceAllocationResponse{
				AllocationResult: &pluginapi.ResourceAllocation{
					ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
						"memory": {
							Annotations: map[string]string{},
						},
					},
				},
			},
		}, {
			name:     "need mon_groups",
			qosLevel: "dedicated_cores",
			policy:   `{"enabled-closids": ["dedicated"]}`,
			expectResp: &pluginapi.ResourceAllocationResponse{
				AllocationResult: &pluginapi.ResourceAllocation{
					ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
						"memory": {
							Annotations: map[string]string{},
						},
					},
				},
			},
		}, {
			name:     "not need mon_groups",
			qosLevel: "shared_cores",
			policy:   `{"enabled-closids": ["dedicated"]}`,
			expectResp: &pluginapi.ResourceAllocationResponse{
				AllocationResult: &pluginapi.ResourceAllocation{
					ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
						"memory": {
							Annotations: map[string]string{
								util.AnnotationRdtNeedPodMonGroups: "false",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config.Configuration{
				AgentConfiguration: &agent.AgentConfiguration{
					StaticAgentConfiguration: &agent.StaticAgentConfiguration{
						QRMPluginsConfiguration: &qrm.QRMPluginsConfiguration{
							MBQRMPluginConfig: &qrm.MBQRMPluginConfig{
								MonGroupsPolicy: tt.policy,
								QoSGroupEnabledQoS: []string{
									string(consts.QoSLevelSharedCores),
									string(consts.QoSLevelDedicatedCores),
								},
							},
						},
					},
				},
			}
			grouper := qosgroup.NewPodGrouper(conf)
			mgr := NewManager(conf, grouper)
			req := &pluginapi.ResourceRequest{
				PodName: "test",
			}
			resp := &pluginapi.ResourceAllocationResponse{
				AllocationResult: &pluginapi.ResourceAllocation{
					ResourceAllocation: map[string]*pluginapi.ResourceAllocationInfo{
						"memory": {
							Annotations: map[string]string{},
						},
					},
				},
			}
			mgr.PostProcessAllocate(req, resp, tt.qosLevel, tt.annotations)
			assert.Equal(t, tt.expectResp, resp, "allocate resp not equal")
		})
	}
}
