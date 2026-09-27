package runner

import (
	"flag"
	"strings"

	"github.com/sarchlab/mgpusim/v5/amd/timing/sectortlb"
)

var l3Entries = flag.Int("l3-tlb-entries", 1024, "L3 sector entries (tags); requires -vm-mode=demand-l3.")
var l3Ways = flag.Int("l3-tlb-ways", 8, "L3 set associativity (sector LRU).")
var l3Subentries = flag.Int("l3-tlb-subentries", 16, "Independently valid adjacent virtual pages per L3 sector.")
var l3Cycles = flag.Int("l3-tlb-latency", 40, "L3 lookup latency in cycles at 1 GHz.")
var l3Width = flag.Int("l3-tlb-width", 4,
	"Maximum L3 admissions, lookups, fills, and responses per cycle; model assumption.")
var l3MSHRs = flag.Int("l3-tlb-mshrs", 64, "L3 outstanding page-miss slots; model assumption.")
var l3Waiters = flag.Int("l3-tlb-max-waiters", 64,
	"L3 requesters per outstanding page, including the first; model assumption.")
var l3Inflight = flag.Int("l3-tlb-max-inflight", 256, "L3 total admitted requests not yet responded; model assumption.")

func l3ConfigFromFlags() sectortlb.Config {
	return sectortlb.Config{Entries: *l3Entries, Ways: *l3Ways, Subentries: *l3Subentries,
		LookupCycles: *l3Cycles, Width: *l3Width, MSHRs: *l3MSHRs, MaxWaiters: *l3Waiters, MaxInflight: *l3Inflight}
}

func validateL3Flags() {
	if *vmModeFlag == "demand-l3" {
		l3ConfigFromFlags().Validate()
		return
	}
	flag.Visit(func(f *flag.Flag) {
		if strings.HasPrefix(f.Name, "l3-tlb-") {
			panic("L3 TLB parameters require -vm-mode=demand-l3")
		}
	})
}
