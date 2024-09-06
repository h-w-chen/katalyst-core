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

package endpoint

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pluginapi "github.com/kubewharf/katalyst-api/pkg/protocol/evictionplugin/v1alpha1"
)

type dummyRecover struct {
	Endpoint
}

func (d *dummyRecover) GetTopEvictionPods(c context.Context, request *pluginapi.GetTopEvictionPodsRequest) (*pluginapi.GetTopEvictionPodsResponse, error) {
	return &pluginapi.GetTopEvictionPodsResponse{
		TargetPods: []*v1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				UID: "123321",
			},
		}},
		DeletionOptions: nil,
	}, nil
}

func (d *dummyRecover) Recover() (Endpoint, error) {
	return d, nil
}

func TestRecoverableEndpoint_GetTopEvictionPods(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyRecover{},
	}

	resp, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.TargetPods))
	assert.Equal(t, "123321", string(resp.TargetPods[0].UID))
}

type dummyUnrecover struct{}

func (d *dummyUnrecover) Recover() (Endpoint, error) {
	return nil, errors.New("test unrecoverable")
}

func TestRecoverableEndpoint_GetTopEvictionPods_Unrecovered(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyUnrecover{},
	}

	resp, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.TargetPods))
}

type dummyFaultyRecover struct {
	Endpoint
}

func (d *dummyFaultyRecover) GetTopEvictionPods(c context.Context, request *pluginapi.GetTopEvictionPodsRequest) (*pluginapi.GetTopEvictionPodsResponse, error) {
	return nil, errors.New("test endpoint error")
}

func (d *dummyFaultyRecover) Recover() (Endpoint, error) {
	return d, nil
}

func TestRecoverableEndpoint_GetTopEvictionPods_FaultyReturn(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyFaultyRecover{},
	}

	_, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.Error(t, err)
	assert.Equal(t, "test endpoint error", err.Error())
	assert.Nilf(t, rep.endpoint, "faulty method call should clear endpoint connection")
}
