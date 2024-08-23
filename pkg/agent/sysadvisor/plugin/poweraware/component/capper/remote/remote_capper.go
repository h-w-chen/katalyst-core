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

package remote

import (
	"context"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/component/capper"
	"github.com/kubewharf/katalyst-core/pkg/util/external/power"
)

type RemoteCapper struct {
}

func (r RemoteCapper) Init() error {
	// todo: if not enabled, return error unavailable
	panic("implement me")
}

func (r RemoteCapper) Reset() {
	//TODO implement me
	panic("implement me")
}

func (r RemoteCapper) Cap(ctx context.Context, targetWatts, currWatt int) {
	//TODO implement me
	panic("implement me")
}

func New() *RemoteCapper {
	return &RemoteCapper{}
}

var _ capper.PowerCapper = &RemoteCapper{}
var _ power.InitResetter = &RemoteCapper{}
