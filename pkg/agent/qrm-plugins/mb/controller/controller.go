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

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/allocator"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

const intervalMBController = time.Second * 1

type Controller struct {
	mbMonitor   monitor.Monitor
	mbAllocator allocator.Allocator
}

func (c Controller) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, c.run, intervalMBController)
}

func (c Controller) run(ctx context.Context) {
	panic("impl")
}

func New() *Controller {
	return &Controller{
		mbMonitor:   nil,
		mbAllocator: nil,
	}
}
