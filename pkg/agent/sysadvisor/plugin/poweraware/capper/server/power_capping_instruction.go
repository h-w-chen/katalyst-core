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
	"fmt"
	"github.com/pkg/errors"
	"strconv"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
)

type powerCappingOpCode string

const (
	keyOpCode         = "op-code"
	keyOpCurrentValue = "op-current-value"
	keyOpTargetValue  = "op-target-value"

	OpCap     powerCappingOpCode = "4"
	OpReset   powerCappingOpCode = "-1"
	OpUnknown powerCappingOpCode = "-2"
)

var powerCappingReset = &CappingInstruction{
	OpCode: OpReset,
}

type CappingInstruction struct {
	OpCode         powerCappingOpCode
	OpCurrentValue string
	OpTargetValue  string
}

func (c CappingInstruction) ToListAndWatchResponse() *advisorsvc.ListAndWatchResponse {
	return &advisorsvc.ListAndWatchResponse{
		PodEntries: nil,
		ExtraEntries: []*advisorsvc.CalculationInfo{{
			CgroupPath: "",
			CalculationResult: &advisorsvc.CalculationResult{
				Values: map[string]string{
					keyOpCode:         string(c.OpCode),
					keyOpCurrentValue: c.OpCurrentValue,
					keyOpTargetValue:  c.OpTargetValue,
				},
			},
		}},
	}
}

func getCappingInstruction(info *advisorsvc.CalculationInfo) (*CappingInstruction, error) {
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

	opCode, ok := values[keyOpCode]
	if !ok {
		return nil, errors.New("op-code not found")
	}

	opCurrValue := values[keyOpCurrentValue]
	opTargetValue := values[keyOpTargetValue]

	return &CappingInstruction{
		OpCode:         powerCappingOpCode(opCode),
		OpCurrentValue: opCurrValue,
		OpTargetValue:  opTargetValue,
	}, nil
}

func FromListAndWatchResponse(response *advisorsvc.ListAndWatchResponse) ([]*CappingInstruction, error) {
	if len(response.ExtraEntries) == 0 {
		return nil, errors.New("no valid data of no capping instruction")
	}

	count := len(response.ExtraEntries)
	cis := make([]*CappingInstruction, count)
	for i, calcInfo := range response.ExtraEntries {
		ci, err := getCappingInstruction(calcInfo)
		if err != nil {
			return nil, err
		}

		cis[i] = ci
	}

	return cis, nil
}

func capToMessage(targetWatts, currWatt int) (*CappingInstruction, error) {
	if targetWatts >= currWatt {
		return nil, errors.New("invalid power cap request")
	}

	return &CappingInstruction{
		OpCode:         OpCap,
		OpCurrentValue: fmt.Sprintf("%d", currWatt),
		OpTargetValue:  fmt.Sprintf("%d", targetWatts),
	}, nil
}

func ToCappingRequest(ci *CappingInstruction) (opCode powerCappingOpCode, targetValue, currentValue int) {
	if ci == nil {
		return OpUnknown, 0, 0
	}

	opCode = ci.OpCode
	if OpReset == opCode {
		return
	}

	var err error
	if targetValue, err = strconv.Atoi(ci.OpTargetValue); err != nil {
		return OpUnknown, 0, 0
	}

	if currentValue, err = strconv.Atoi(ci.OpCurrentValue); err != nil {
		return OpUnknown, 0, 0
	}

	return
}
