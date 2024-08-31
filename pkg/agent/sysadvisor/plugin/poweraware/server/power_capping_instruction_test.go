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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
)

func Test_cappingInstruction_ToListAndWatchResponse(t *testing.T) {
	t.Parallel()

	type fields struct {
		opCode         string
		opCurrentValue string
		opTargetValue  string
	}
	tests := []struct {
		name   string
		fields fields
		want   *advisorsvc.ListAndWatchResponse
	}{
		{
			name: "happy path",
			fields: fields{
				opCode:         "4",
				opCurrentValue: "555",
				opTargetValue:  "500",
			},
			want: &advisorsvc.ListAndWatchResponse{
				ExtraEntries: []*advisorsvc.CalculationInfo{
					{
						CalculationResult: &advisorsvc.CalculationResult{
							Values: map[string]string{
								"op-code":          "4",
								"op-current-value": "555",
								"op-target-value":  "500",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := cappingInstruction{
				opCode:         tt.fields.opCode,
				opCurrentValue: tt.fields.opCurrentValue,
				opTargetValue:  tt.fields.opTargetValue,
			}
			eq := reflect.DeepEqual(tt.want, c.ToListAndWatchResponse())
			assert.Truef(t, eq, "should be equal")
		})
	}
}

func Test_getCappingInstruction(t *testing.T) {
	t.Parallel()

	type args struct {
		info *advisorsvc.CalculationInfo
	}
	tests := []struct {
		name    string
		args    args
		want    *cappingInstruction
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path no error",
			args: args{
				info: &advisorsvc.CalculationInfo{
					CalculationResult: &advisorsvc.CalculationResult{
						Values: map[string]string{
							"op-code":          "4",
							"op-current-value": "100",
							"op-target-value":  "80",
						},
					},
				},
			},
			want: &cappingInstruction{
				opCode:         "4",
				opCurrentValue: "100",
				opTargetValue:  "80",
			},
			wantErr: assert.NoError,
		},
		{
			name: "nil value map is invalid",
			args: args{
				info: &advisorsvc.CalculationInfo{
					CalculationResult: &advisorsvc.CalculationResult{
						Values: nil,
					},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := getCappingInstruction(tt.args.info)
			if !tt.wantErr(t, err, fmt.Sprintf("getCappingInstruction(%v)", tt.args.info)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getCappingInstruction(%v)", tt.args.info)
		})
	}
}

func TestFromListAndWatchResponse(t *testing.T) {
	t.Parallel()

	type args struct {
		response *advisorsvc.ListAndWatchResponse
	}
	tests := []struct {
		name    string
		args    args
		want    []*cappingInstruction
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path no error",
			args: args{
				response: &advisorsvc.ListAndWatchResponse{
					ExtraEntries: []*advisorsvc.CalculationInfo{
						{
							CalculationResult: &advisorsvc.CalculationResult{
								Values: map[string]string{
									"op-code":          "4",
									"op-current-value": "555",
									"op-target-value":  "500",
								},
							},
						},
						{
							CalculationResult: &advisorsvc.CalculationResult{
								Values: map[string]string{
									"op-code": "-1",
								},
							},
						},
					},
				},
			},
			want: []*cappingInstruction{
				{
					opCode:         "4",
					opCurrentValue: "555",
					opTargetValue:  "500",
				},
				{
					opCode: "-1",
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := FromListAndWatchResponse(tt.args.response)
			if !tt.wantErr(t, err, fmt.Sprintf("FromListAndWatchResponse(%v)", tt.args.response)) {
				return
			}
			assert.Equalf(t, tt.want, got, "FromListAndWatchResponse(%v)", tt.args.response)
		})
	}
}
