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

package qosgroup

import (
	"encoding/json"
	"fmt"

	"github.com/kubewharf/katalyst-api/pkg/consts"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/config"
)

// PodGrouper determines QoS related properties of the pod in admitting request
type PodGrouper struct {
	poolToSharedSubgroup  map[string]int
	defaultSharedSubgroup int
	enabledQos            sets.String
}

func identifyCPUSetPool(annoInReq map[string]string) string {
	if pool, ok := annoInReq[consts.PodAnnotationCPUEnhancementCPUSet]; ok {
		return pool
	}

	// fall back to original composite (not flattened) form
	enhancementValue, ok := annoInReq[consts.PodAnnotationCPUEnhancementKey]
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
func (p *PodGrouper) GetQoSGroup(qosLevel string, annotations map[string]string) (QoSGroup, error) {
	if !p.enabledQos.Has(qosLevel) {
		return "", fmt.Errorf("qos %s not enabled", qosLevel)
	}

	qos := QoS{
		Level: consts.QoSLevel(qosLevel),
	}

	if qos.Level == consts.QoSLevelSharedCores {
		pool := identifyCPUSetPool(annotations)
		subGroup, ok := p.poolToSharedSubgroup[pool]
		if !ok {
			subGroup = p.defaultSharedSubgroup
		}
		qos.SubLevel = subGroup
	}

	return qos.ToCtrlGroup()
}

func NewPodGrouper(conf *config.Configuration) *PodGrouper {
	defaultSubgroup, ok := conf.CPUSetPoolToSharedSubgroup["share"]
	if !ok {
		defaultSubgroup = defaultSharedSubgroup
	}

	return &PodGrouper{
		poolToSharedSubgroup:  conf.CPUSetPoolToSharedSubgroup,
		defaultSharedSubgroup: defaultSubgroup,
		enabledQos:            sets.NewString(conf.QoSGroupEnabledQoS...),
	}
}
