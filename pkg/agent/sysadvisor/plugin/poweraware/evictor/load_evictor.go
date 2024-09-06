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

package evictor

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
)

// LoadEvictor is the interface used in advisor policy
type LoadEvictor interface {
	Evict(ctx context.Context, targetPercent int)
}

type loadEvictor struct {
	qosConfig  *generic.QoSConfiguration
	podFetcher pod.PodFetcher
	podEvictor PodEvictor
}

func (l loadEvictor) isBE(pod *v1.Pod) bool {
	qosLevel, err := l.qosConfig.GetQoSLevelForPod(pod)
	if err != nil {
		// unknown, not BE anyway
		return false
	}

	return qosLevel == apiconsts.PodAnnotationQoSLevelReclaimedCores
}

func (l loadEvictor) Evict(ctx context.Context, targetPercent int) {
	pods, err := l.podFetcher.GetPodList(ctx, l.isBE)
	if err != nil {
		klog.Errorf("pap: evict: failed to get pods: %v", err)
		return
	}
	klog.V(6).Infof("pap: evict: there are %d BE pods, %d%%", len(pods), targetPercent)

	// discard pending requests not handled yet; we will provide a new sleet of evict requests anyway
	l.podEvictor.Reset(ctx)

	// todo: replace this FIFO evict policy
	countToLive := len(pods) * targetPercent / 100
	for _, p := range pods[:countToLive] {
		// not care much for returned error as power alert eviction is the best effort by design
		_ = l.podEvictor.Evict(ctx, p)
	}
}

func NewPowerEvictServer(qosConfig *generic.QoSConfiguration,
	podFetcher pod.PodFetcher,
	podEvictor PodEvictor,
) LoadEvictor {
	return &loadEvictor{
		qosConfig:  qosConfig,
		podFetcher: podFetcher,
		podEvictor: podEvictor,
	}
}

// noopPodEvictor does not really evict any pod other than counting the invocations;
// used in unit test, or when eviction feature is disabled
type noopPodEvictor struct {
	called int
}

func (d *noopPodEvictor) Reset(ctx context.Context) {}

func (d *noopPodEvictor) Evict(ctx context.Context, pod *v1.Pod) error {
	// dummy does no op, besides recording called times
	d.called += 1
	return nil
}

func NewNoopPodEvictor() PodEvictor {
	return &noopPodEvictor{}
}
