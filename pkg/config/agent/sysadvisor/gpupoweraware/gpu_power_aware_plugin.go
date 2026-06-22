package gpupoweraware

type GPUPowerAwarePluginConfiguration struct {
	GPUPowerCappingAdvisorSocketAbsPath string
}

func NewPowerAwarePluginConfiguration() *GPUPowerAwarePluginConfiguration {
	return &GPUPowerAwarePluginConfiguration{}
}
