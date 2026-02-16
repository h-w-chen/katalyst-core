package priority

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/advisor/resource"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

var (
	instance GroupPriority
	once     sync.Once
)

func GetInstance() GroupPriority {
	once.Do(func() {
		instance = make(GroupPriority, len(resctrlMajorGroupWeights))
		for key, value := range resctrlMajorGroupWeights {
			instance[key] = value
		}
	})
	return instance
}

type GroupPriority map[string]int

func (g GroupPriority) GetWeight(name string) int {
	baseWeight, ok := g[getMajor(name)]
	if !ok {
		return defaultWeight
	}

	return baseWeight + getSubWeight(name)

}

func (g GroupPriority) SortGroups(groups []string) []sets.String {
	sort.Slice(groups, func(i, j int) bool {
		return g.GetWeight(groups[i]) > g.GetWeight(groups[j])
	})

	return g.mergeGroupsByWeight(groups)
}

func (g GroupPriority) mergeGroupsByWeight(groups []string) []sets.String {
	var mergedGroups []sets.String
	for _, group := range groups {
		if len(mergedGroups) == 0 {
			mergedGroups = append(mergedGroups, sets.NewString(group))
			continue
		}

		lastGroup := mergedGroups[len(mergedGroups)-1]
		weightLastGroup, err := g.getWeightOfEquivGroup(lastGroup)
		if err != nil {
			general.Warningf("[mbm] failed to get allocation weight of group %v: %v", lastGroup, err)
			continue
		}

		if g.GetWeight(group) == weightLastGroup {
			lastGroup.Insert(group)
			continue
		}

		mergedGroups = append(mergedGroups, sets.NewString(group))
	}
	return mergedGroups
}

func (g GroupPriority) getWeightOfEquivGroup(equivGroups sets.String) (int, error) {
	repGroup, err := resource.GetGroupRepresentative(equivGroups)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("failed to get representative of groups %v", equivGroups))
	}

	return g.GetWeight(repGroup), nil
}

func (g GroupPriority) AddWeight(name string, weight int) {
	g[name] = weight
}
