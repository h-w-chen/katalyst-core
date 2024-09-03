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

package externalpolicy

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/agent"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/server"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

// todo: consider to have a configuration arg for this interval
const intervalRetryToConnect = time.Second * 10

type powerCappingComponent struct {
	client      advisorsvc.AdvisorServiceClient
	localCapper server.NodePowerCapper
}

func (r powerCappingComponent) Run(ctx context.Context) {
	if err := r.localCapper.Init(); err != nil {
		general.Errorf("local power capping failed to initialized: %v", err)
		return
	}

	wait.UntilWithContext(ctx, r.run, intervalRetryToConnect)
}

// run is ok to exit, as the remote server may be disconnected or even unavailable
// however, it is important being able to get hold of the remote server when it
// comes online in "hot pluggable" style
func (r powerCappingComponent) run(ctx context.Context) {
	stream, err := r.client.ListAndWatch(ctx, &advisorsvc.Empty{})
	if err != nil {
		// ok fail here; will retry in next run
		return
	}

	for {
		lwResp, err := stream.Recv()
		if err != nil {
			general.Errorf("failed to receive messages from stream: %v", err)
			return
		}

		capInsts, err := server.FromListAndWatchResponse(lwResp)
		if err != nil {
			general.Errorf("failed to receive power capping messages from stream: %v", err)
			return
		}

		for _, ci := range capInsts {
			if err := r.process(ctx, ci); err != nil {
				general.Errorf("failed to process capping req: %v", err)
			}
		}
	}
}

func (r powerCappingComponent) process(ctx context.Context, ci *server.CappingInstruction) error {
	op, targetValue, currentValue := server.ToCappingRequest(ci)
	switch op {
	case server.OpReset:
		r.localCapper.Reset()
		return nil
	case server.OpCap:
		r.localCapper.Cap(ctx, targetValue, currentValue)
		return nil
	default:
		return fmt.Errorf("unexpected op code: %v", op)
	}
}

func NewPowerCappingComponent(client advisorsvc.AdvisorServiceClient, capper server.NodePowerCapper) (agent.Component, error) {
	return &powerCappingComponent{
		client:      client,
		localCapper: capper,
	}, nil
}
