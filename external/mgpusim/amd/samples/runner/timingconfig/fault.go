package timingconfig

import (
	"fmt"

	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/mgpusim/v5/amd/driver"
	"github.com/sarchlab/mgpusim/v5/amd/timing/faultvm"
	"github.com/sarchlab/mgpusim/v5/amd/timing/idealmapping"
	"github.com/sarchlab/mgpusim/v5/amd/timing/sectortlb"
)

// WithFaultVM selects demand/ideal hardware, with an optional demand-mode L3.
// The legacy shared MMU and -ideal-local-page-table paths remain independent.
func (b Builder) WithFaultVM(mode string, config faultvm.Config) Builder {
	if mode != "demand" && mode != "ideal" && mode != "demand-l3" {
		panic("faultvm: expected demand, ideal, or demand-l3 mode")
	}
	config.Validate()
	b.faultMode = mode
	b.faultConfig = config
	if b.l3Config == (sectortlb.Config{}) {
		b.l3Config = sectortlb.DefaultConfig()
	}
	return b
}

// WithL3TLB configures the per-GPU sector TLB in demand-l3 mode.
func (b Builder) WithL3TLB(config sectortlb.Config) Builder {
	if b.faultMode != "demand-l3" {
		panic("L3 TLB configuration requires demand-l3 mode")
	}
	config.Validate()
	b.l3Config = config
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
		var provider messaging.Component = mmuComp
		if b.faultMode == "demand-l3" {
			l3 := sectortlb.Build(b.simulation, fmt.Sprintf("GPU[%d].L3TLB", index),
				sectortlb.MakeSpec(b.l3Config, mmuComp.GetPortByName("Top").AsRemote(), b.log2PageSize))
			conn.PlugIn(l3.GetPortByName("Top"))
			conn.PlugIn(l3.GetPortByName("Bottom"))
			provider = l3
		}
		b.createGPU(index, b.createGPUBuilder(provider, gpuDriver), gpuDriver, conn)
	}
	return gpuDriver
}
