package mb

import (
	"time"

	"github.com/pkg/errors"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller"
	"github.com/kubewharf/katalyst-core/pkg/config/generic"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type plugin struct {
	qosConfig              *generic.QoSConfiguration
	pluginRegistrationDirs []string

	dieTopology        *machine.DieTopology
	incubationInterval time.Duration

	mbController *controller.Controller
}

func (p *plugin) Name() string {
	return "qrm_mb_plugin"
}

func (p *plugin) Start() error {
	general.InfofV(6, "mbm: plugin component starting ....")
	general.InfofV(6, "mbm: mb incubation interval %v", p.incubationInterval)
	general.InfofV(6, "mbm: numa-CCD-cpu topology: \n%s", p.dieTopology)

	// todo: NOT to return error (to crash explicitly); consider downgrade service
	if !p.dieTopology.FakeNUMAEnabled {
		return errors.New("mbm: not virtual numa; no need to dynamically manage the memory bandwidth")
	}

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				general.Errorf("mbm: background run exited, due to error: %v", err)
			}
		}()

		p.mbController.Run()
	}()

	return nil
}

func (p *plugin) Stop() error {
	general.Infof("mbm: mb plugin is stopping...")
	return p.mbController.Stop()
}
