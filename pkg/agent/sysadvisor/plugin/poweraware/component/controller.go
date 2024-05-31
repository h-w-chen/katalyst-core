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
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/types"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/node"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
	"github.com/kubewharf/katalyst-core/pkg/util/external/rapl"
)

// 8 seconds between actions since RAPL capping needs 4-6 seconds to stablize itself
const intervalSpecFetch = time.Second * 8

type PowerAwareController interface {
	Run(ctx context.Context)
}

type powerAwareController struct {
	specFetcher            SpecFetcher
	powerReader            PowerReader
	reconciler             PowerReconciler
	powerLimitInitResetter rapl.InitResetter
}

func (p powerAwareController) Run(ctx context.Context) {
	if err := p.powerReader.Init(); err != nil {
		klog.Errorf("pap: failed to initialize power reader: %v; exited", err)
		return
	}
	if err := p.powerLimitInitResetter.Init(); err != nil {
		klog.Errorf("pap: failed to initialize power capping: %v; exited", err)
		return
	}

	wait.Until(func() { p.run(ctx) }, intervalSpecFetch, ctx.Done())

	p.powerReader.Cleanup()
	p.powerLimitInitResetter.Reset()
}

func (p powerAwareController) run(ctx context.Context) {
	spec, err := p.specFetcher.GetPowerSpec(ctx)
	if err != nil {
		klog.Errorf("pap: getting power spec failed: %#v", err)
		return
	}

	// remove power capping limit if any, on NONE alert
	if spec.Alert == types.PowerAlertOK {
		p.powerLimitInitResetter.Reset()
		return
	}

	if types.InternalOpPause == spec.InternalOp {
		return
	}

	currentWatts, err := p.powerReader.Get(ctx)
	if err != nil {
		klog.Errorf("pap: reading power failed: %#v", err)
		return
	}
	p.reconciler.Reconcile(ctx, spec, currentWatts)
}

func NewController(dryRun bool, nodeFetcher node.NodeFetcher,
	podFetcher pod.PodFetcher, qosConfig *generic.QoSConfiguration,
	limiter rapl.RAPLLimiter,
) PowerAwareController {
	return &powerAwareController{
		specFetcher: &specFetcherByNodeAnnotation{nodeFetcher: nodeFetcher},
		powerReader: &ipmiPowerReader{},
		reconciler: &powerReconciler{
			dryRun:      dryRun,
			priorAction: PowerAction{},
			evictor: &loadEvictor{
				qosConfig:  qosConfig,
				podFetcher: podFetcher,
				podKiller:  &dummyPodKiller{},
			},
			capper:   &raplCapper{raplLimiter: limiter},
			strategy: &ruleBasedPowerStrategy{coefficient: linearDecay{b: defaultDecayB}},
		},
		powerLimitInitResetter: limiter,
	}
}
