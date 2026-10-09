// Package tlbprofile defines translation-cache capacity presets independently
// of the VM fault policy. PaperCapacity retains the AMD sharing topology.
package tlbprofile

import "github.com/sarchlab/mgpusim/v5/amd/timing/sectortlb"

const (
	// Legacy preserves the original L1/L2 geometry and timing.
	Legacy = "legacy"
	// PaperCapacity matches LIBRA table capacities, associativity, and lookup
	// cycles per TLB instance, without claiming NVIDIA TPC/GPC organization.
	PaperCapacity = "libra-capacity"
)

// Validate rejects unknown profiles. Empty is the internal legacy default.
func Validate(profile string) {
	if profile != "" && profile != Legacy && profile != PaperCapacity {
		panic("unknown TLB profile; expected legacy or libra-capacity")
	}
}

// L1 uses one fully associative 16-page set and a one-cycle lookup.
// Port width and MSHR count retain the corresponding legacy L1 path values;
// these execution-resource choices are not specified by the paper table.
func L1(width, mshrs int) sectortlb.Config {
	return sectortlb.Config{Entries: 16, Ways: 16, Subentries: 1, LookupCycles: 1,
		Width: width, MSHRs: mshrs, MaxWaiters: 64, MaxInflight: 256}
}

// L2 uses 128 sector tags, eight ways, 16 independently valid subentries,
// and a ten-cycle lookup. Execution resources retain the legacy L2 limits.
func L2() sectortlb.Config {
	return sectortlb.Config{Entries: 128, Ways: 8, Subentries: 16, LookupCycles: 10,
		Width: 1024, MSHRs: 64, MaxWaiters: 64, MaxInflight: 4096}
}
