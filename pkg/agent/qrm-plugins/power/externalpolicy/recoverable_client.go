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
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/advisorsvc"
	"github.com/kubewharf/katalyst-core/pkg/util/process"
)

// recoverableAdvisorSvcClient re-establishes the session with server from fresh on the streaming LW method call
type recoverableAdvisorSvcClient struct {
	advisorSocketAbsPath string
	conn                 *grpc.ClientConn
}

func (r *recoverableAdvisorSvcClient) AddContainer(ctx context.Context, in *advisorsvc.ContainerMetadata, opts ...grpc.CallOption) (*advisorsvc.AddContainerResponse, error) {
	return nil, errors.New("not supported")
}

func (r *recoverableAdvisorSvcClient) RemovePod(ctx context.Context, in *advisorsvc.RemovePodRequest, opts ...grpc.CallOption) (*advisorsvc.RemovePodResponse, error) {
	return nil, errors.New("not supported")
}

func (r *recoverableAdvisorSvcClient) ListAndWatch(ctx context.Context, in *advisorsvc.Empty, opts ...grpc.CallOption) (advisorsvc.AdvisorService_ListAndWatchClient, error) {
	// always starts a fresh connection on each invocation
	if r.conn != nil {
		_ = r.conn.Close()
		r.conn = nil
	}
	conn, err := process.Dial(r.advisorSocketAbsPath, 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to service")
	}

	// keep conn for future cleanup
	r.conn = conn

	client := advisorsvc.NewAdvisorServiceClient(conn)
	return client.ListAndWatch(ctx, in, opts...)
}

func NewRecoverableAdvisorSvcClient(advisorSocketPath string) advisorsvc.AdvisorServiceClient {
	return &recoverableAdvisorSvcClient{
		advisorSocketAbsPath: advisorSocketPath,
	}
}
