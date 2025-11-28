package decorator

import (
	"sync"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/qosaware/resource/cpu/assembler/headroomassembler"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

var initializers sync.Map
var enabledDecorator string
var lock sync.RWMutex

type InitFunc func(inner headroomassembler.HeadroomAssembler,
	conf *config.Configuration, extraConf interface{},
	metaReader metacache.MetaReader, metaServer *metaserver.MetaServer, emitter metrics.MetricEmitter,
) headroomassembler.HeadroomAssembler

func RegisterInitializer(name string, initFunc InitFunc) {
	initializers.Store(name, initFunc)
}

func GetRegisteredInitializers() map[string]InitFunc {
	res := make(map[string]InitFunc)
	initializers.Range(func(key, value interface{}) bool {
		res[key.(string)] = value.(InitFunc)
		return true
	})
	return res
}

func EnablePlugin(name string) {
	lock.Lock()
	defer lock.Unlock()
	enabledDecorator = name
}

func GetEnabledPlugin() string {
	lock.RLock()
	defer lock.RUnlock()

	return enabledDecorator
}
