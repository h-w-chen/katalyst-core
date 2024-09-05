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

package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_powerCapAdvisorPluginServer_Reset(t *testing.T) {
	t.Parallel()

	pcs := newpPowerCapAdvisorPluginServer()
	pcs.Reset()

	assert.Equal(t,
		&CappingInstruction{OpCode: "-1"},
		pcs.latestCappingInst,
		"the latest power capping instruction should be RESET",
	)
}

func Test_powerCapAdvisorPluginServer_Cap(t *testing.T) {
	t.Parallel()

	pcs := newpPowerCapAdvisorPluginServer()
	pcs.Cap(context.TODO(), 111, 123)

	assert.Equal(t,
		&CappingInstruction{
			OpCode:         "4",
			OpCurrentValue: "123",
			OpTargetValue:  "111",
		},
		pcs.latestCappingInst,
		"the latest power capping instruction should be what was just to Cap",
	)
}