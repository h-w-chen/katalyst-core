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
	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
)

type cappingInstruction struct {
	opCode         string
	opCurrentValue string
	opTargetValue  string
}

func (c cappingInstruction) ToListAndWatchResponse() *advisorsvc.ListAndWatchResponse {
	return &advisorsvc.ListAndWatchResponse{
		PodEntries: nil,
		ExtraEntries: []*advisorsvc.CalculationInfo{{
			CgroupPath: "",
			CalculationResult: &advisorsvc.CalculationResult{
				Values: map[string]string{
					"op-code":          c.opCode,
					"op-current-value": c.opCurrentValue,
					"op-target-value":  c.opTargetValue,
				},
			},
		}},
	}
}

func getCappingInstruction(info *advisorsvc.CalculationInfo) (*cappingInstruction, error) {
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

	opCurrValue := values["op-current-value"]
	opTargetValue := values["op-target-value"]

	return &cappingInstruction{
		opCode:         opCode,
		opCurrentValue: opCurrValue,
		opTargetValue:  opTargetValue,
	}, nil
}

func FromListAndWatchResponse(response *advisorsvc.ListAndWatchResponse) ([]*cappingInstruction, error) {
	if len(response.ExtraEntries) == 0 {
		return nil, errors.New("no valid data of no capping instruction")
	}

	count := len(response.ExtraEntries)
	cis := make([]*cappingInstruction, count)
	for i, calcInfo := range response.ExtraEntries {
		ci, err := getCappingInstruction(calcInfo)
		if err != nil {
			return nil, err
		}

		cis[i] = ci
	}

	return cis, nil
}
