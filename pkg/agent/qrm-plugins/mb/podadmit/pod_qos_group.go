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
	"encoding/json"
	"fmt"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	qosutil "github.com/kubewharf/katalyst-core/pkg/util/qos"
)

const templateSharedSubgroup = "shared-%d"

// PodGrouper determines QoS related properties of the pod in admitting request
type PodGrouper struct {
	poolToSharedSubgroup  map[string]int
	defaultSharedSubgroup int
}

func getSharedSubgroup(val int) string {
	return fmt.Sprintf(templateSharedSubgroup, val)
}

func (p *PodGrouper) getSharedSubgroupByPool(pool string) string {
	if v, ok := p.poolToSharedSubgroup[pool]; ok {
		return getSharedSubgroup(v)
	}
	return getSharedSubgroup(p.defaultSharedSubgroup)
}

func IsDecdicatedCoresNumaExclusive(qosLevel string, annotations map[string]string) bool {
	if apiconsts.PodAnnotationQoSLevelDedicatedCores != qosLevel {
		return false
	}
	return qosutil.AnnotationsIndicateNUMAExclusive(annotations)
}

func (p *PodGrouper) IsShared30(qosLevel string, annotations map[string]string) bool {
	if subgroup, err := p.GetQoSGroup(qosLevel, annotations); err == nil {
		return subgroup == "shared-30"
	}

	return false
}

func identifyCPUSetPool(annoInReq map[string]string) string {
	if pool, ok := annoInReq[apiconsts.PodAnnotationCPUEnhancementCPUSet]; ok {
		return pool
	}

	// fall back to original composite (not flattened) form
	enhancementValue, ok := annoInReq[apiconsts.PodAnnotationCPUEnhancementKey]
	if !ok {
		return ""
	}

	flattenedEnhancements := map[string]string{}
	err := json.Unmarshal([]byte(enhancementValue), &flattenedEnhancements)
	if err != nil {
		return ""
	}
	return identifyCPUSetPool(flattenedEnhancements)
}

// GetQoSGroup returns qos group based on inputs of qos level and relevant annotation, e.g.
// input "dedicated_cores", ...                   => "dedicated"
// input "shared_cores", {"cpuset_pool": "batch"} => "shared-30" // should there be a valid cpuset pool mapping
func (p *PodGrouper) GetQoSGroup(qosLevel string, annotations map[string]string) (string, error) {
	switch qosLevel {
	case apiconsts.PodAnnotationQoSLevelDedicatedCores:
		return string(task.QoSGroupDedicated), nil
	case apiconsts.PodAnnotationQoSLevelSystemCores:
		return string(task.QoSGroupSystem), nil
	case apiconsts.PodAnnotationQoSLevelReclaimedCores:
		return string(task.QoSGroupReclaimed), nil
	}

	if qosLevel != apiconsts.PodAnnotationQoSLevelSharedCores {
		return "", fmt.Errorf("unrecognized qos level %s", qosLevel)
	}

	pool := identifyCPUSetPool(annotations)
	return p.getSharedSubgroupByPool(pool), nil
}

func NewPodGrouper(poolToSharedSubgroup map[string]int, defaultSharedSubgroup int) *PodGrouper {
	return &PodGrouper{
		poolToSharedSubgroup:  poolToSharedSubgroup,
		defaultSharedSubgroup: defaultSharedSubgroup,
	}
}
