package util

import (
	"encoding/json"
	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	qosutil "github.com/kubewharf/katalyst-core/pkg/util/qos"
)

func IsDecdicatedCoresNumaExclusive(qosLevel string, annotations map[string]string) bool {
	if apiconsts.PodAnnotationQoSLevelDedicatedCores != qosLevel {
		return false
	}
	return isNumaExclusive(annotations)
}

func isNumaExclusive(annotations map[string]string) bool {
	// simplify the check logic by only looking at memory aspect,
	// which seems a reasonable condition for Socket Pods
	const enhancementKey = apiconsts.PodAnnotationMemoryEnhancementKey
	enhancementValue, ok := annotations[enhancementKey]
	if !ok {
		return false
	}

	flattenedEnhancements := map[string]string{}
	err := json.Unmarshal([]byte(enhancementValue), &flattenedEnhancements)
	if err != nil {
		general.Errorf("parse enhancement %s failed: %v", enhancementKey, err)
		return false
	}

	return qosutil.AnnotationsIndicateNUMAExclusive(flattenedEnhancements)
}
