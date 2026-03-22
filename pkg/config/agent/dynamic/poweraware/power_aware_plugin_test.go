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

package poweraware

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubewharf/katalyst-api/pkg/apis/config/v1alpha1"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic/crd"
)

func TestNewPowerAwareConfiguration(t *testing.T) {
	t.Run("should create config with default values", func(t *testing.T) {
		config := NewPowerAwareConfiguration()

		assert.NotNil(t, config)
		assert.False(t, config.DryRun)
		assert.False(t, config.DisablePowerCapping)
		assert.False(t, config.DisablePowerPressureEvict)
	})
}

func TestPowerAwareConfiguration_ApplyConfiguration(t *testing.T) {
	t.Run("should not modify config when CRD is nil", func(t *testing.T) {
		config := NewPowerAwareConfiguration()
		originalDryRun := config.DryRun

		config.ApplyConfiguration(nil)

		assert.Equal(t, originalDryRun, config.DryRun)
	})

	t.Run("should not modify config when PowerAwareConfiguration is nil", func(t *testing.T) {
		config := NewPowerAwareConfiguration()
		originalDryRun := config.DryRun

		conf := &crd.DynamicConfigCRD{}
		config.ApplyConfiguration(conf)

		assert.Equal(t, originalDryRun, config.DryRun)
	})

	t.Run("should apply valid configuration", func(t *testing.T) {
		config := NewPowerAwareConfiguration()

		dryRun := true
		disablePowerCapping := true
		disablePowerPressureEvict := true

		conf := &crd.DynamicConfigCRD{
			PowerAwareConfiguration: &v1alpha1.PowerAwareConfiguration{
				Spec: v1alpha1.PowerAwareConfigurationSpec{
					Config: v1alpha1.PowerAwareConfig{
						DryRun:                    &dryRun,
						DisablePowerCapping:       &disablePowerCapping,
						DisablePowerPressureEvict: &disablePowerPressureEvict,
					},
				},
			},
		}

		config.ApplyConfiguration(conf)

		assert.True(t, config.DryRun)
		assert.True(t, config.DisablePowerCapping)
		assert.True(t, config.DisablePowerPressureEvict)
	})

	t.Run("should apply partial configuration", func(t *testing.T) {
		config := NewPowerAwareConfiguration()

		dryRun := true

		conf := &crd.DynamicConfigCRD{
			PowerAwareConfiguration: &v1alpha1.PowerAwareConfiguration{
				Spec: v1alpha1.PowerAwareConfigurationSpec{
					Config: v1alpha1.PowerAwareConfig{
						DryRun: &dryRun,
					},
				},
			},
		}

		config.ApplyConfiguration(conf)

		assert.True(t, config.DryRun)
		assert.False(t, config.DisablePowerCapping)
		assert.False(t, config.DisablePowerPressureEvict)
	})
}
