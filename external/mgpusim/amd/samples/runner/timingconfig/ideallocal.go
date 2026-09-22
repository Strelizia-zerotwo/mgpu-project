package timingconfig

import (
	"fmt"

	"github.com/sarchlab/mgpusim/v5/amd/driver"
	"github.com/sarchlab/mgpusim/v5/amd/timing/idealmapping"
)

// WithIdealLocalPageTable gives every GPU an independent MMU and page table.
// Mapping propagation is instantaneous. TLBs and MMU walk latency are unchanged.
func (b Builder) WithIdealLocalPageTable() Builder {
	b.idealLocalPageTable = true
	return b
}

func (b *Builder) buildIdealLocalPlatform() *driver.Driver {
	tables := idealmapping.NewIdealLocal(b.log2PageSize, b.numGPUs)
	gpuDriver := b.buildGPUDriver(tables)
	conn := b.createConnection(gpuDriver, nil)

	for index := 1; index <= b.numGPUs; index++ {
		mmuComp := b.buildMMU(fmt.Sprintf("GPU[%d].MMU", index), tables.View(index-1))
		conn.PlugIn(mmuComp.GetPortByName("Top"))
		gpuBuilder := b.createGPUBuilder(mmuComp, gpuDriver)
		b.createGPU(index, gpuBuilder, gpuDriver, conn)
	}
	return gpuDriver
}
