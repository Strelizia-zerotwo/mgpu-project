package runner

import (
	"strings"

	"github.com/sarchlab/mgpusim/v5/amd/timing/sectortlb"
)

// Sector TLB counters are reported even without -report-all and even when zero. This
// makes missing/inactive GPUs distinguishable from absent instrumentation.
func (r *reporter) reportSectorTLB(c *sectortlb.Comp) {
	prefix := "l3_"
	if strings.Contains(c.Name(), ".L1") {
		prefix = "l1_"
	} else if strings.Contains(c.Name(), ".L2TLB") {
		prefix = "l2_"
	}
	s, spec := c.State.Stats, c.Spec()
	put := func(name string, value float64, unit string) {
		r.dataRecorder.InsertData(tableName, metric{Location: c.Name(), What: name, Value: value, Unit: unit})
	}
	put("hit", float64(s.Hits), "count")
	put("miss", float64(s.Misses), "count")
	put("mshr-hit", float64(s.Coalesced), "count")
	counts := map[string]uint64{
		"requests": s.Requests, "completed": s.Completed, "lower_responses": s.LowerResponses,
		"fills": s.Fills, "sector_evictions": s.Evictions, "invalidated_pages": s.Invalidated,
		"stale_responses": s.StaleResponses, "reset_dropped": s.ResetDropped,
		"outstanding_peak": s.OutstandingPeak, "outstanding_end": uint64(c.State.Outstanding),
		"mshr_peak": s.MSHRPeak, "mshr_end": uint64(len(c.State.Misses)),
		"entries": uint64(spec.Entries), "ways": uint64(spec.Ways), "sets": uint64(spec.Entries / spec.Ways),
		"subentries": uint64(spec.Subentries), "width": uint64(spec.Width), "mshrs": uint64(spec.MSHRs),
		"max_waiters": uint64(spec.MaxWaiters), "max_inflight": uint64(spec.MaxInflight),
	}
	for name, value := range counts {
		put(prefix+name, float64(value), "count")
	}
	put(prefix+"page_size", float64(uint64(1)<<spec.Log2PageSize), "byte")
	put(prefix+"frequency_hz", 1e9, "Hz")
	put(prefix+"lookup_cycles", float64(spec.LookupCycles), "cycles")
	put(prefix+"admission_blocked_cycles", float64(s.AdmissionBlockedCycles), "cycles")
	put(prefix+"lookup_blocked_cycles", float64(s.LookupBlockedCycles), "cycles")
	put(prefix+"response_blocked_cycles", float64(s.ResponseBlockedCycles), "cycles")
	put(prefix+"translation_average", averageSeconds(s.LatencySum, s.Completed), "second")
	put(prefix+"translation_max", secondsOf(s.LatencyMax), "second")
}
