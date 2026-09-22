package timingconfig

import (
	"fmt"

	"github.com/sarchlab/mgpusim/v5/amd/driver"
	"github.com/sarchlab/mgpusim/v5/amd/timing/faultvm"
	"github.com/sarchlab/mgpusim/v5/amd/timing/idealmapping"
)

// WithFaultVM selects matched demand/ideal local translation hardware.
// The legacy shared MMU and -ideal-local-page-table paths remain independent.
func (b Builder) WithFaultVM(mode string, config faultvm.Config) Builder {
	if mode != "demand" && mode != "ideal" {
		panic("faultvm: expected demand or ideal mode")
	}
	config.Validate()
	b.faultMode = mode
	b.faultConfig = config
	return b
}

func (b *Builder) buildFaultPlatform() *driver.Driver {
	tables := idealmapping.NewDemandLocal(b.log2PageSize, b.numGPUs)
	if b.faultMode == "ideal" {
		tables = idealmapping.NewIdealLocal(b.log2PageSize, b.numGPUs)
	}
	tables.UseFixedPlacement()
	gpuDriver := b.buildGPUDriver(tables)
	conn := b.createConnection(gpuDriver, nil)
	host := faultvm.BuildHost(b.simulation, "HostMMU", b.faultConfig, tables)
	conn.PlugIn(host.GetPortByName("Top"))
	for index := 1; index <= b.numGPUs; index++ {
		mmuComp := faultvm.BuildGPU(b.simulation, fmt.Sprintf("GPU[%d].MMU", index),
			faultvm.MakeGPUSpec(b.faultConfig, host.GetPortByName("Top").AsRemote(), b.log2PageSize), tables.View(index-1))
		conn.PlugIn(mmuComp.GetPortByName("Top"))
		conn.PlugIn(mmuComp.GetPortByName("Host"))
		b.createGPU(index, b.createGPUBuilder(mmuComp, gpuDriver), gpuDriver, conn)
	}
	return gpuDriver
}
