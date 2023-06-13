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

package eviction

import (
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/dynamic/eviction"
)

// MemoryPressureEvictionPluginOptions is the options of MemoryPressureEvictionPlugin
type MemoryPressureEvictionPluginOptions struct {
	EnableNumaLevelDetection             bool
	EnableSystemLevelDetection           bool
	NumaFreeBelowWatermarkTimesThreshold int
	SystemKswapdRateThreshold            int
	SystemKswapdRateExceedTimesThreshold int
	NumaEvictionRankingMetrics           []string
	SystemEvictionRankingMetrics         []string
	GracePeriod                          int64
	EnableRSSOveruseDetection            bool
	RssOveruseRateThreshold              float64
}

// NewMemoryPressureEvictionPluginOptions returns a new MemoryPressureEvictionPluginOptions
func NewMemoryPressureEvictionPluginOptions() *MemoryPressureEvictionPluginOptions {
	return &MemoryPressureEvictionPluginOptions{
		EnableNumaLevelDetection:             eviction.DefaultEnableNumaLevelDetection,
		EnableSystemLevelDetection:           eviction.DefaultEnableSystemLevelDetection,
		NumaFreeBelowWatermarkTimesThreshold: eviction.DefaultNumaFreeBelowWatermarkTimesThreshold,
		SystemKswapdRateThreshold:            eviction.DefaultSystemKswapdRateThreshold,
		SystemKswapdRateExceedTimesThreshold: eviction.DefaultSystemKswapdRateExceedTimesThreshold,
		NumaEvictionRankingMetrics:           eviction.DefaultNumaEvictionRankingMetrics,
		SystemEvictionRankingMetrics:         eviction.DefaultSystemEvictionRankingMetrics,
		GracePeriod:                          eviction.DefaultGracePeriod,
		EnableRSSOveruseDetection:            eviction.DefaultEnableRssOveruseDetection,
		RssOveruseRateThreshold:              eviction.DefaultRssOveruseRateThreshold,
	}
}

// AddFlags parses the flags to MemoryPressureEvictionPluginOptions
func (o *MemoryPressureEvictionPluginOptions) AddFlags(fss *cliflag.NamedFlagSets) {
	fs := fss.FlagSet("eviction-memory-pressure")

	fs.BoolVar(&o.EnableNumaLevelDetection, "eviction-enable-numa-level-detection",
		o.EnableNumaLevelDetection,
		"whether to enable numa-level detection")
	fs.BoolVar(&o.EnableSystemLevelDetection, "eviction-enable-system-level-detection",
		o.EnableSystemLevelDetection,
		"whether to enable system-level detection")
	fs.IntVar(&o.NumaFreeBelowWatermarkTimesThreshold, "eviction-numa-free-below-watermark-times-threshold",
		o.NumaFreeBelowWatermarkTimesThreshold,
		"the threshold for the number of times NUMA's free memory falls below the watermark")
	fs.IntVar(&o.SystemKswapdRateThreshold, "eviction-system-kswapd-rate-threshold", o.SystemKswapdRateThreshold,
		"the threshold for the rate of kswapd reclaiming rate")
	fs.IntVar(&o.SystemKswapdRateExceedTimesThreshold, "eviction-system-kswapd-rate-exceed-times-threshold",
		o.SystemKswapdRateExceedTimesThreshold,
		"the threshold for the number of times the kswapd reclaiming rate exceeds the threshold")
	fs.StringSliceVar(&o.NumaEvictionRankingMetrics, "eviction-numa-ranking-metrics", o.NumaEvictionRankingMetrics,
		"the metrics used to rank pods for eviction at the NUMA level")
	fs.StringSliceVar(&o.SystemEvictionRankingMetrics, "eviction-system-ranking-metrics", o.SystemEvictionRankingMetrics,
		"the metrics used to rank pods for eviction at the system level")
	fs.Int64Var(&o.GracePeriod, "eviction-grace-period", o.GracePeriod,
		"the grace period of memory pressure eviction")
	fs.BoolVar(&o.EnableRSSOveruseDetection, "eviction-enable-rss-overuse-detection", o.EnableRSSOveruseDetection,
		"whether to enable pod-level rss overuse detection")
	fs.Float64Var(&o.RssOveruseRateThreshold, "eviction-rss-overuse-rate-threshold", o.RssOveruseRateThreshold,
		"the threshold for the rate of rss overuse threshold")
}

// ApplyTo applies MemoryPressureEvictionPluginOptions to MemoryPressureEvictionPluginConfiguration
func (o *MemoryPressureEvictionPluginOptions) ApplyTo(c *eviction.MemoryPressureEvictionPluginConfiguration) error {
	c.EnableNumaLevelDetection = o.EnableNumaLevelDetection
	c.EnableSystemLevelDetection = o.EnableSystemLevelDetection
	c.NumaFreeBelowWatermarkTimesThreshold = o.NumaFreeBelowWatermarkTimesThreshold
	c.SystemKswapdRateThreshold = o.SystemKswapdRateThreshold
	c.SystemKswapdRateExceedTimesThreshold = o.SystemKswapdRateExceedTimesThreshold
	c.NumaEvictionRankingMetrics = o.NumaEvictionRankingMetrics
	c.SystemEvictionRankingMetrics = o.SystemEvictionRankingMetrics
	c.GracePeriod = o.GracePeriod
	c.EnableRssOveruseDetection = o.EnableRSSOveruseDetection
	c.RssOveruseRateThreshold = o.RssOveruseRateThreshold

	return nil
}