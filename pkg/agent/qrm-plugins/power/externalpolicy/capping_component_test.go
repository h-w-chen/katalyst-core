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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/server"
)

type dummyCapper struct {
	server.NodePowerCapper

	resetCalled bool

	capCalled             bool
	capTarget, capCurrent int
}

func (d *dummyCapper) Reset() {
	d.resetCalled = true
}

func (d *dummyCapper) Cap(ctx context.Context, targetWatts, currWatt int) {
	d.capCalled = true
	d.capCurrent = currWatt
	d.capTarget = targetWatts
}

func Test_powerCappingComponent_process(t *testing.T) {
	t.Parallel()

	type fields struct {
		client      advisorsvc.AdvisorServiceClient
		localCapper *dummyCapper
	}
	type args struct {
		ctx context.Context
		ci  *server.CappingInstruction
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantReset  bool
		wantCap    bool
		capTarget  int
		capCurrent int
	}{
		{
			name: "-1 to reset power capping",
			fields: fields{
				client:      nil,
				localCapper: &dummyCapper{},
			},
			args: args{
				ctx: nil,
				ci: &server.CappingInstruction{
					OpCode: "-1",
				},
			},
			wantErr:   false,
			wantReset: true,
		},
		{
			name: "4 to cap the power",
			fields: fields{
				client:      nil,
				localCapper: &dummyCapper{},
			},
			args: args{
				ctx: nil,
				ci: &server.CappingInstruction{
					OpCode:         "4",
					OpCurrentValue: "400",
					OpTargetValue:  "380",
				},
			},
			wantErr:   false,
			wantReset: false,
		},
		{
			name: "unknown op code is error",
			fields: fields{
				client:      nil,
				localCapper: &dummyCapper{},
			},
			args: args{
				ctx: nil,
				ci: &server.CappingInstruction{
					OpCode:         "99",
					OpCurrentValue: "99",
					OpTargetValue:  "99",
				},
			},
			wantErr:    true,
			wantReset:  false,
			wantCap:    false,
			capTarget:  0,
			capCurrent: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := powerCappingComponent{
				client:      tt.fields.client,
				localCapper: tt.fields.localCapper,
			}
			assert.Falsef(t, tt.fields.localCapper.resetCalled, "expected capper not reset yet")

			if err := r.process(tt.args.ctx, tt.args.ci); (err != nil) != tt.wantErr {
				t.Errorf("process() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantReset {
				assert.Truef(t, tt.fields.localCapper.resetCalled, "expected capper to reset")
			}

			if tt.wantCap {
				assert.Equal(t, tt.capTarget, tt.fields.localCapper.capTarget)
				assert.Equal(t, tt.capCurrent, tt.fields.localCapper.capCurrent)
			}
		})
	}
}

type mockCapper struct {
	mock.Mock
	server.NodePowerCapper
}

func (m *mockCapper) Init() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockCapper) Cap(ctx context.Context, targetWatts, currWatt int) {
	m.Called(ctx, targetWatts, currWatt)
}

type mockClient struct {
	mock.Mock
	advisorsvc.AdvisorServiceClient
}

func (m *mockClient) ListAndWatch(ctx context.Context, in *advisorsvc.Empty, opts ...grpc.CallOption) (advisorsvc.AdvisorService_ListAndWatchClient, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(advisorsvc.AdvisorService_ListAndWatchClient), args.Error(1)
}

type mockStream struct {
	mock.Mock
	calledCount int
	advisorsvc.AdvisorService_ListAndWatchClient
}

func (m *mockStream) Recv() (*advisorsvc.ListAndWatchResponse, error) {
	m.calledCount += 1
	if m.calledCount > 1 {
		return nil, errors.New("test fake EOF")
	}

	return &advisorsvc.ListAndWatchResponse{
		ExtraEntries: []*advisorsvc.CalculationInfo{{
			CgroupPath: "",
			CalculationResult: &advisorsvc.CalculationResult{
				Values: map[string]string{
					"op-code":          "4",
					"op-current-value": "111",
					"op-target-value":  "98",
				},
			},
		}},
	}, nil
}

var _ advisorsvc.AdvisorService_ListAndWatchClient = &mockStream{}

func Test_powerCappingComponent_Run_premature_exit_on_capper_init_error(t *testing.T) {
	t.Parallel()

	capper := new(mockCapper)
	capper.On("Init").Return(errors.New("test error"))

	pc, _ := NewPowerCappingComponent(nil, capper)
	pc.Run(context.TODO())

	capper.AssertExpectations(t)
}

func Test_powerCappingComponent_run(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	in := &advisorsvc.Empty{}

	fakeStream := new(mockStream)

	client := new(mockClient)
	client.On("ListAndWatch", ctx, in).Return(fakeStream, nil)

	capper := new(mockCapper)
	capper.On("Cap", ctx, 98, 111).Once()

	pc := &powerCappingComponent{
		client:      client,
		localCapper: capper,
	}

	pc.run(ctx)

	capper.AssertExpectations(t)
	client.AssertExpectations(t)
	fakeStream.AssertExpectations(t)
}
