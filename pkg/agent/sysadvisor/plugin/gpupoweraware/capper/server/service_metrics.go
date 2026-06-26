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
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	powermetric "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/metric"
)

func (g *powerCapService) emitGPUPowerCapInstruction(instruction *capper.Instruction) {
	//	powermetric.EmitGPUPowerCapInstruction(g.emitter, instruction.OpCode, instruction.RawTargetValue, instruction.RawCurrentValue, string(instruction.Level))
}

func (g *powerCapService) emitGPUPowerCapReset() {
	//	powermetric.EmitGPUPowerCapReset(g.emitter)
}

func (g *powerCapService) emitErrorCode(errorCause powermetric.ErrorCause) {
	powermetric.EmitErrorCode(g.emitter, errorCause)
}
