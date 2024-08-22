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

	pluginapi "github.com/kubewharf/katalyst-api/pkg/protocol/evictionplugin/v1alpha1"
)

type EndpointRecover interface {
	Recover() (Endpoint, error)
}

type remoteEndpointRecover struct {
	socketPath string
	pluginName string
}

func (r remoteEndpointRecover) Recover() (Endpoint, error) {
	return NewRemoteEndpointImpl(r.socketPath, r.pluginName)
}

// RecoverableEndpoint wraps up an endpoint to remote service which might stop or resume working anytime
type RecoverableEndpoint struct {
	pluginName string
	endpoint   Endpoint
	recover    EndpointRecover
}

func (r *RecoverableEndpoint) Name() string {
	return r.pluginName
}

func (r *RecoverableEndpoint) tryRecover() {
	if r.endpoint != nil {
		return
	}

	var err error
	if r.endpoint, err = r.recover.Recover(); err != nil {
		// ok unable to re-establish the connection
		r.endpoint = nil
	}
}

func (r *RecoverableEndpoint) ThresholdMet(c context.Context) (*pluginapi.ThresholdMetResponse, error) {
	r.tryRecover()
	if r.endpoint == nil {
		return &pluginapi.ThresholdMetResponse{}, nil
	}

	resp, err := r.endpoint.ThresholdMet(c)
	if err != nil {
		r.endpoint = nil
	}

	return resp, err
}

func (r *RecoverableEndpoint) GetTopEvictionPods(c context.Context, request *pluginapi.GetTopEvictionPodsRequest) (*pluginapi.GetTopEvictionPodsResponse, error) {
	r.tryRecover()
	if r.endpoint == nil {
		return &pluginapi.GetTopEvictionPodsResponse{}, nil
	}

	resp, err := r.endpoint.GetTopEvictionPods(c, request)
	if err != nil {
		r.endpoint = nil
	}

	return resp, err
}

func (r *RecoverableEndpoint) GetEvictPods(c context.Context, request *pluginapi.GetEvictPodsRequest) (*pluginapi.GetEvictPodsResponse, error) {
	r.tryRecover()
	if r.endpoint == nil {
		return &pluginapi.GetEvictPodsResponse{}, nil
	}

	resp, err := r.endpoint.GetEvictPods(c, request)
	if err != nil {
		r.endpoint = nil
	}

	return resp, err
}

func (r *RecoverableEndpoint) Start() {
	if r.endpoint != nil {
		r.endpoint.Start()
	}
}

func (r *RecoverableEndpoint) Stop() {
	if r.endpoint != nil {
		r.endpoint.Stop()
	}
}

func (r *RecoverableEndpoint) IsStopped() bool {
	if r.endpoint == nil {
		return true
	}

	return r.endpoint.IsStopped()
}

func (r *RecoverableEndpoint) StopGracePeriodExpired() bool {
	if r.endpoint == nil {
		return false
	}

	return r.endpoint.StopGracePeriodExpired()
}

func NewRecoverableRemoteEndpoint(socketPath, pluginName string) *RecoverableEndpoint {
	return &RecoverableEndpoint{
		pluginName: pluginName,
		recover: &remoteEndpointRecover{
			socketPath: socketPath,
			pluginName: pluginName,
		},
	}
}
