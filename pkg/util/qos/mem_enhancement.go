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

package qos

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
)

func ParseMemoryEnhancement(qosConf *generic.QoSConfiguration, pod *v1.Pod) map[string]string {
	if pod == nil || qosConf == nil {
		return nil
	}
	return qosConf.GetQoSEnhancementKVs(pod, map[string]string{}, apiconsts.PodAnnotationMemoryEnhancementKey)
}

// IsPodNumaBinding checks whether the pod needs numa-binding
func IsPodNumaBinding(qosConf *generic.QoSConfiguration, pod *v1.Pod) bool {
	memoryEnhancement := ParseMemoryEnhancement(qosConf, pod)
	return AnnotationsIndicateNUMABinding(memoryEnhancement)
}

// IsPodNumaExclusive checks whether the pod needs numa-exclusive
func IsPodNumaExclusive(qosConf *generic.QoSConfiguration, pod *v1.Pod) bool {
	isPodNumaBinding := IsPodNumaBinding(qosConf, pod)
	if !isPodNumaBinding {
		return false
	}

	isDedicatedPod, err := qosConf.CheckDedicatedQoS(pod, map[string]string{})
	if err != nil || !isDedicatedPod {
		return false
	}

	memoryEnhancement := ParseMemoryEnhancement(qosConf, pod)
	return AnnotationsIndicateNUMAExclusive(memoryEnhancement)
}

func AnnotationsIndicateNUMABinding(annotations map[string]string) bool {
	return annotations[apiconsts.PodAnnotationMemoryEnhancementNumaBinding] ==
		apiconsts.PodAnnotationMemoryEnhancementNumaBindingEnable
}

func AnnotationsIndicateNUMAExclusive(annotations map[string]string) bool {
	return AnnotationsIndicateNUMABinding(annotations) &&
		annotations[apiconsts.PodAnnotationMemoryEnhancementNumaExclusive] ==
			apiconsts.PodAnnotationMemoryEnhancementNumaExclusiveEnable
}

// GetRSSOverUseEvictThreshold parse the user specified threshold and checks if it's valid
func GetRSSOverUseEvictThreshold(qosConf *generic.QoSConfiguration, pod *v1.Pod) (threshold *float64, invalid bool) {
	memoryEnhancement := ParseMemoryEnhancement(qosConf, pod)
	thresholdStr, ok := memoryEnhancement[apiconsts.PodAnnotationMemoryEnhancementRssOverUseThreshold]
	if !ok {
		return
	}

	parsedThreshold, parseErr := strconv.ParseFloat(thresholdStr, 64)
	if parseErr != nil {
		invalid = true
		return
	}

	if !isValidRatioThreshold(parsedThreshold) {
		invalid = true
		return
	}

	threshold = &parsedThreshold
	return
}

func isValidRatioThreshold(threshold float64) bool {
	return threshold > 0
}

// GetOOMPriority parse the user specified oom priority from memory enhancement annotation
func GetOOMPriority(qosConf *generic.QoSConfiguration, pod *v1.Pod) (priority *int, invalid bool) {
	memoryEnhancement := ParseMemoryEnhancement(qosConf, pod)
	oomPriorityStr, ok := memoryEnhancement[apiconsts.PodAnnotationMemoryEnhancementOOMPriority]
	if !ok {
		return
	}

	parsedOOMPriority, parseErr := strconv.Atoi(oomPriorityStr)
	if parseErr != nil {
		invalid = true
		return
	}

	priority = &parsedOOMPriority
	return
}

// GetActualNUMABindingResult parse the user specified numa binding result from pod annotation
// if the annotation is not found, return -1
func GetActualNUMABindingResult(qosConf *generic.QoSConfiguration, pod *v1.Pod) (int, error) {
	if pod == nil {
		return -1, fmt.Errorf("pod cannot be nil")
	}

	if result, ok := pod.Annotations[apiconsts.PodAnnotationNUMABindResultKey]; ok {
		return strconv.Atoi(result)
	} else if IsPodNumaBinding(qosConf, pod) {
		return -1, fmt.Errorf("numa binding result not found")
	}

	return -1, nil
}
