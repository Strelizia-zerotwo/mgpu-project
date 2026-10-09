package runner

import (
	"flag"

	"github.com/sarchlab/mgpusim/v5/amd/samples/runner/timingconfig/tlbprofile"
)

var tlbProfileFlag = flag.String("tlb-profile", tlbprofile.Legacy,
	"TLB geometry: legacy, or libra-capacity (paper capacities/latencies, AMD sharing; timing r9nano only).")

func (r *Runner) validateTLBProfile() {
	tlbprofile.Validate(*tlbProfileFlag)
	if *tlbProfileFlag == tlbprofile.PaperCapacity && (!r.Timing || r.GPUType != "r9nano") {
		panic("-tlb-profile=libra-capacity requires -timing -gpu=r9nano")
	}
}
