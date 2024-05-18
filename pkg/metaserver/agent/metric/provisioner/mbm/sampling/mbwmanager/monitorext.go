package mbwmanager

// this file is extension of mbw monitor, for the ease of integration
// except this file, all mbw monitor source code files are kept in the original form
// as much as possible to facilitate evolution of its own

// GetPackageSamples return total memory bandwidth of all packages
// it reuses input param as the buffer to avoid excessive object allocations
func (m MBMonitor) GetPackageSamples(samples []float64) []float64 {
	if m.MemoryBandwidth.Cores[0].LRMB_Delta != 0 {
		// ignore the result at the 1st sec
		return nil
	}

	m.MemoryBandwidth.PackageLocker.RLock()
	defer m.MemoryBandwidth.PackageLocker.RUnlock()

	for i, pkg := range m.MemoryBandwidth.Packages {
		samples[i] = float64(pkg.Total)
	}

	return samples
}

// GetNUMASamples return total memory bandwidth of all (likely fake) numa nodes
// it reuses input param as the buffer to avoid excessive object allocations
func (m MBMonitor) GetNUMASamples(samples []float64) []float64 {
	if m.MemoryBandwidth.Cores[0].LRMB_Delta != 0 {
		// ignore the result at the 1st sec
		return nil
	}

	m.MemoryBandwidth.CoreLocker.RLock()
	defer m.MemoryBandwidth.CoreLocker.RUnlock()

	for i, numa := range m.MemoryBandwidth.Numas {
		samples[i] = float64(numa.Total)
	}

	return samples
}
