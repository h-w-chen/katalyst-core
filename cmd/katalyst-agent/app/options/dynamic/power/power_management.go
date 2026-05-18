package power

import (
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic/power"
)

type PowerOptions struct {
	DisablePowerAdvisor bool
	DisablePowerCapping bool

	// PowerReductionRatio is the percentage ratio the power usage is allowed to reduce
	PowerReductionRatio int
}

func NewPowerOptions() *PowerOptions {
	return &PowerOptions{
		DisablePowerAdvisor: true,
		DisablePowerCapping: true,
		PowerReductionRatio: 10,
	}
}

func (o *PowerOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("power-management")
	fs.BoolVar(&o.DisablePowerCapping, "disable-power-capping", o.DisablePowerCapping, "disable power capping")
	fs.BoolVar(&o.DisablePowerAdvisor, "disable-power-advisor", o.DisablePowerAdvisor, "disable power advisor")
	fs.IntVar(&o.PowerReductionRatio, "power-reduction-ratio", o.PowerReductionRatio, "allowed power reduction percentage")
}

func (o *PowerOptions) ApplyTo(c *power.PowerManagementConfiguration) error {
	c.DisablePowerCapping = o.DisablePowerCapping
	c.DisablePowerAdvisor = o.DisablePowerAdvisor
	c.PowerReductionRatio = o.PowerReductionRatio
	return nil
}
