//go:build integration_test
// +build integration_test

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

package podadmit

import (
	"context"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/process"
	watcherapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	pluginapi "k8s.io/kubelet/pkg/apis/resourceplugin/v1alpha1"
)

func Test_Pod_Admit_Service_Integration(t *testing.T) {
	t.Logf("lengthy integration test; not intended as part of check-in tests")

	////lis := bufconn.Listen(1024 * 1024)
	//lis, err := net.Listen("unix", "/tmp/qrm_mb_plugin.sock")
	//if err != nil {
	//	t.Errorf("test error : %v", err)
	//	return
	//}
	//s := grpc.NewServer()
	//manager := &admitter{
	//	qosConfig:     generic.NewQoSConfiguration(),
	//	domainManager: &mbdomain.MBDomainManager{},
	//}
	//pluginapi.RegisterResourcePluginServer(s, manager)
	//go func() {
	//	if err := s.Serve(lis); err != nil {
	//		t.Errorf("failed to serve: %v", err)
	//	}
	//}()
	//
	dummyQoSConfig := generic.NewQoSConfiguration()
	dummyDomainManager := &mbdomain.MBDomainManager{}
	stubController, _ := controller.New(nil, nil, dummyDomainManager, nil)
	svc, err := NewPodAdmitService(dummyQoSConfig, dummyDomainManager, stubController, nil, []string{"/tmp"})
	if err != nil {
		t.Errorf("test error : %v", err)
		return
	}
	_ = svc.Start()
	defer svc.Stop()

	//conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
	//	return lis.Dial()
	//}), grpc.WithInsecure())
	conn, err := process.Dial("/tmp/qrm_mb_plugin.sock", time.Second*3)
	if err != nil {
		t.Errorf("dial: %v", err)
		return
	}

	regClient := watcherapi.NewRegistrationClient(conn)
	r, err := regClient.GetInfo(context.TODO(), &watcherapi.InfoRequest{})
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	t.Logf("registration: %v", r)

	client := pluginapi.NewResourcePluginClient(conn)
	resp, err := client.Allocate(context.Background(), &pluginapi.ResourceRequest{
		PodUid:       "pod-123-4567",
		PodNamespace: "ns-test",
		PodName:      "pod-test",
		PodRole:      "",
		PodType:      "",
		ResourceName: "",
		Hint: &pluginapi.TopologyHint{
			Nodes:     []uint64{2},
			Preferred: true,
		},
		ResourceRequests: nil,
		Labels:           nil,
		Annotations: map[string]string{
			"katalyst.kubewharf.io/qos_level": "dedicated_cores",
		},
	})

	assert.NoError(t, err)
	t.Logf("response: %#v", resp)
}
