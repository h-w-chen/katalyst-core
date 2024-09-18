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

package policydeliver

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/allocator"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
)

type Deliver interface {
	DeliverMBAllocs(mbAllocs []policy.MBAlloc) error
}

type deliver struct {
	mbMonitor   monitor.Monitor
	mbAllocator allocator.Allocator
}

func (d deliver) DeliverMBAllocs(mbAllocs []policy.MBAlloc) error {
	for _, alloc := range mbAllocs {
		if err := d.deliverMBAlloc(alloc); err != nil {
			return err
		}
	}
	return nil
}

func (d deliver) deliverMBAlloc(alloc policy.MBAlloc) error {
	for _, node := range alloc.AppPool.GetNUMANodes() {
		ccdCurrs := d.mbMonitor.GetMB(node)
		ccdAllocs := distributeCCDMBs(alloc.MBUpperBound, ccdCurrs)
		if err := d.mbAllocator.AllocateMB(node, ccdAllocs); err != nil {
			return err
		}
	}

	return nil
}

func New(mbMonitor monitor.Monitor, mbAllocator allocator.Allocator) Deliver {
	return &deliver{
		mbMonitor:   mbMonitor,
		mbAllocator: mbAllocator,
	}
}
