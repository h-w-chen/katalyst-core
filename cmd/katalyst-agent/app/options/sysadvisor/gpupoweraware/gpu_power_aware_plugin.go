package gpupoweraware

import (
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/sysadvisor/gpupoweraware"
)

type PowerAwarePluginOptions struct {
	GPUPowerCappingAdvisorSocketAbsPath string
}

func (p *PowerAwarePluginOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("gpu-power-aware-plugin")
	fs.StringVar(&p.GPUPowerCappingAdvisorSocketAbsPath,
		"gpu-power-capping-advisor-sock-abs-path",
		p.GPUPowerCappingAdvisorSocketAbsPath,
		"absolute path of unix socket file for power capping advisor served in sys-advisor")
}

func (p *PowerAwarePluginOptions) ApplyTo(o *gpupoweraware.GPUPowerAwarePluginConfiguration) error {
	o.GPUPowerCappingAdvisorSocketAbsPath = p.GPUPowerCappingAdvisorSocketAbsPath
	return nil
}

func NewPowerAwarePluginOptions() *PowerAwarePluginOptions {
	return &PowerAwarePluginOptions{
		GPUPowerCappingAdvisorSocketAbsPath: "/tmp/gpu-x.socket",
	}
}
