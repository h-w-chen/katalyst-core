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

package component

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	apiconsts "github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/server"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
)

type LoadEvictor interface {
	Evict(ctx context.Context, targetPercent int)
}

type loadEvictor struct {
	qosConfig  *generic.QoSConfiguration
	podFetcher pod.PodFetcher
	podEvictor server.PodEvictor
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
	// todo: verify pod fetcher able to get pods indeed
	all, _ := l.podFetcher.GetPodList(ctx, func(v *v1.Pod) bool {
		return true
	})
	klog.V(6).Infof("pap: evict: get %d pods", len(all))

	pods, err := l.podFetcher.GetPodList(ctx, l.isBE)
	if err != nil {
		klog.Errorf("pap: evict: failed to get pods: %v", err)
		return
	}
	klog.V(6).Infof("pap: evict: there are %d BE pods", len(pods))

	// discard pending requests not handled yet; we will provide a new sleet of evict requests anyway
	l.podEvictor.Reset(ctx)

	countToLive := len(pods) * targetPercent / 100
	for _, p := range pods[:countToLive] {
		// not care much for returned error as power alert eviction is the best effort by design
		_ = l.podEvictor.Evict(ctx, p)
	}

	// todo: remove! this is for internal validatioon only. MUST remove before code is finalized
	_ = l.podEvictor.Evict(ctx, &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-pod",
			UID:  "111-222-333",
		},
	})
}

// NoopPodEvictor does not really evict any pod other than counting the invocations;
// used in unit test, or when eviction feature is disabled
type NoopPodEvictor struct {
	called int
}

func (d *NoopPodEvictor) Reset(ctx context.Context) {}

func (d *NoopPodEvictor) Evict(ctx context.Context, pod *v1.Pod) error {
	// dummy does no op, besides recording called times
	d.called += 1
	return nil
}

var _ server.PodEvictor = &NoopPodEvictor{}
