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

package capper

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper"
)

// Level represents the GPU workload level to cap.
type Level string

const (
	LevelInfer Level = "infer"
	LevelTrain Level = "train"
	LevelAll   Level = "all"

	keyOpLevel = "op-level"
)

var PowerCapReset = &Instruction{
	OpCode: capper.OpReset,
}

// Instruction extends capper.CapInstruction with a GPU workload level.
type Instruction struct {
	OpCode capper.PowerCapOpCode

	OpCurrentValue string
	OpTargetValue  string

	RawTargetValue  int
	RawCurrentValue int

	Level Level
}

func (g Instruction) ToAdviceResponse() *advisorsvc.GetAdviceResponse {
	return &advisorsvc.GetAdviceResponse{
		PodEntries:   nil,
		ExtraEntries: []*advisorsvc.CalculationInfo{wrapGPUInst(g)},
	}
}

func (g Instruction) ToListAndWatchResponse() *advisorsvc.ListAndWatchResponse {
	return &advisorsvc.ListAndWatchResponse{
		PodEntries:   nil,
		ExtraEntries: []*advisorsvc.CalculationInfo{wrapGPUInst(g)},
	}
}

func wrapGPUInst(g Instruction) *advisorsvc.CalculationInfo {
	values := map[string]string{
		"op-code":       string(g.OpCode),
		"current-value": g.OpCurrentValue,
		"target-value":  g.OpTargetValue,
	}
	if g.Level != "" {
		values[keyOpLevel] = string(g.Level)
	}
	return &advisorsvc.CalculationInfo{
		CgroupPath: "",
		CalculationResult: &advisorsvc.CalculationResult{
			Values: values,
		},
	}
}

func getGPUInstructionFromCalcInfo(info *advisorsvc.CalculationInfo) (*Instruction, error) {
	if info == nil {
		return nil, errors.New("invalid data of nil CalculationInfo")
	}

	calcRes := info.CalculationResult
	if calcRes == nil {
		return nil, errors.New("invalid data of nil CalculationResult")
	}

	values := calcRes.GetValues()
	if len(values) == 0 {
		return nil, errors.New("invalid data of empty Values map")
	}

	opCode, ok := values["op-code"]
	if !ok {
		return nil, errors.New("op-code not found")
	}

	opCurrValue := values["current-value"]
	opTargetValue := values["target-value"]

	var err error
	currValue := 0
	if len(opCurrValue) > 0 {
		currValue, err = strconv.Atoi(opCurrValue)
		if err != nil {
			return nil, errors.New("current value format error")
		}
	}

	targetValue := 0
	if len(opTargetValue) > 0 {
		targetValue, err = strconv.Atoi(opTargetValue)
		if err != nil {
			return nil, errors.New("target value format error")
		}
	}

	level := Level(values[keyOpLevel])

	return &Instruction{
		OpCode:          capper.PowerCapOpCode(opCode),
		OpCurrentValue:  opCurrValue,
		OpTargetValue:   opTargetValue,
		RawCurrentValue: currValue,
		RawTargetValue:  targetValue,
		Level:           level,
	}, nil
}

func NewInstruction(targetWatts, currWatt int, level Level) (*Instruction, error) {
	if targetWatts >= currWatt {
		return nil, errors.New("invalid gpu power cap request")
	}

	return &Instruction{
		OpCode:          capper.OpCap,
		OpCurrentValue:  fmt.Sprintf("%d", currWatt),
		OpTargetValue:   fmt.Sprintf("%d", targetWatts),
		RawTargetValue:  targetWatts,
		RawCurrentValue: currWatt,
		Level:           level,
	}, nil
}
