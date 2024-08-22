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

package power

import (
	"fmt"
	"path"

	"k8s.io/client-go/tools/events"

	"github.com/kubewharf/katalyst-core/pkg/agent/evictionmanager/endpoint"
	"github.com/kubewharf/katalyst-core/pkg/agent/evictionmanager/plugin"
	paplugin "github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/plugin"
	"github.com/kubewharf/katalyst-core/pkg/client"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

func NewPowerPressureEndpoint(_ *client.GenericClientSet, _ events.EventRecorder,
	_ *metaserver.MetaServer, _ metrics.MetricEmitter, conf *config.Configuration,
) plugin.EvictionPlugin {
	name := paplugin.EvictionPluginNameNodePowerPressure

	// the expected unix socket path is {conf.PluginRegistrationDir}/{plugin-name}.sock
	socketPath := path.Join(conf.PluginRegistrationDir, fmt.Sprintf("%s.sock", name))

	return endpoint.NewRecoverableRemoteEndpoint(socketPath, name)
}
