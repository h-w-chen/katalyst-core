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

package dynamicpolicy

import (
	"context"
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/metaserver/external"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

func TestMBMController_Run_Stop(t *testing.T) {
	t.Parallel()
	mc := NewMBMController(&metrics.DummyMetrics{}, &external.DummyExternalManager{}, nil, 80, time.Second*2, topologySummary{})

	controller, ok := mc.(*mbmController)
	if !ok {
		t.Fatalf("unexpected type created")
	}

	mc.Run(context.TODO())
	if controller.mbmControllerCancel == nil {
		t.Errorf("expected cancel func after started; got nil")
	}

	mc.Stop()
	if controller.mbmControllerCancel != nil {
		t.Errorf("expected cancel func cleared after stop; got non-nil")
	}
}

func TestGetTopologySummary(t *testing.T) {
	t.Parallel()

	/*	hard code for unit test - mock the case of 2 physical NUMA and 3 fake NUMA each */
	testSiblingNumaMap := map[int]sets.Int{
		0:  sets.Int{1: {}, 2: {}},
		1:  sets.Int{0: {}, 2: {}},
		2:  sets.Int{0: {}, 1: {}},
		3:  sets.Int{4: {}, 5: {}},
		4:  sets.Int{3: {}, 5: {}},
		5:  sets.Int{3: {}, 4: {}},
		6:  sets.Int{7: {}, 8: {}},
		7:  sets.Int{6: {}, 8: {}},
		8:  sets.Int{6: {}, 7: {}},
		9:  sets.Int{10: {}, 11: {}},
		10: sets.Int{9: {}, 11: {}},
		11: sets.Int{9: {}, 10: {}},
	}
	testInfo := &machine.KatalystMachineInfo{
		CPUTopology: &machine.CPUTopology{NumNUMANodes: 12},
		ExtraTopologyInfo: &machine.ExtraTopologyInfo{
			SiblingNumaInfo: &machine.SiblingNumaInfo{
				SiblingNumaMap: testSiblingNumaMap,
			},
		},
	}

	expectedMap := map[int][]int{
		0: []int{0, 1, 2},
		1: []int{3, 4, 5},
		2: []int{6, 7, 8},
		3: []int{9, 10, 11},
	}

	topo := getTopologySummary(testInfo)

	if 4 != topo.numPackages {
		t.Errorf("expected 4 packages; got %d", topo.numPackages)
	}
	if 12 != topo.numNUMAs {
		t.Errorf("expected 12 fake numa nodes; got %d", topo.numNUMAs)
	}
	if !reflect.DeepEqual(expectedMap, topo.PackageNUMAs) {
		t.Errorf("got unexpected package-numa mapping: %v", topo.PackageNUMAs)
	}
}
