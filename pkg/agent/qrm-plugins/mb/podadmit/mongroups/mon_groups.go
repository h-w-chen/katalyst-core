package mongroups

import (
	"encoding/json"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/util"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

type Manager struct {
	podGrouper *qosgroup.PodGrouper
	policy     *policy
}

type policy struct {
	EnabledClosIDs []string `json:"enabled-closids,omitempty"`
}

func NewManager(conf *config.Configuration, podGrouper *qosgroup.PodGrouper) *Manager {
	mgr := &Manager{
		policy:     &policy{},
		podGrouper: podGrouper,
	}
	if conf.MonGroupsPolicy != "" {
		if err := json.Unmarshal([]byte(conf.MonGroupsPolicy), mgr.policy); err != nil {
			general.Errorf("unmarshal mon_groups policy %s error: %v", conf.MonGroupsPolicy, err)
		}
	}
	return mgr
}

func (m *Manager) PostProcessAllocate(req *pluginapi.ResourceRequest, resp *pluginapi.ResourceAllocationResponse, qosLevel string, origReqAnno map[string]string) {
	allocInfo := resp.AllocationResult.ResourceAllocation[string(v1.ResourceMemory)]
	if allocInfo == nil {
		return
	}
	if allocInfo.Annotations == nil {
		allocInfo.Annotations = make(map[string]string)
	}
	if !m.needPodMonGroups(qosLevel, origReqAnno) {
		allocInfo.Annotations[util.AnnotationRdtNeedPodMonGroups] = strconv.FormatBool(false)
		general.InfofV(6, "mbm: pod %s/%s qos %s not need pod mon_groups", req.PodNamespace, req.PodName, qosLevel)
	}
}

// needPodMonGroups returns whether the pod needs to create pod-level mon_groups. Default true
func (m *Manager) needPodMonGroups(qosLevel string, annotations map[string]string) bool {
	qosGroup, err := m.podGrouper.GetQoSGroup(qosLevel, annotations)
	if err != nil {
		return true
	}
	closID := string(qosGroup)

	// if EnabledClosIDs specified, but not hit, return false
	if len(m.policy.EnabledClosIDs) > 0 && !sets.NewString(m.policy.EnabledClosIDs...).Has(closID) {
		return false
	}

	return true
}
