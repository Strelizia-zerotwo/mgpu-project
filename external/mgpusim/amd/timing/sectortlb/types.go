// Package sectortlb implements a finite, set-associative sector TLB. Each
// sector tags adjacent virtual pages, but every subentry is filled on demand.
package sectortlb

import (
	"github.com/sarchlab/akita/v5/mem/vm"
	"github.com/sarchlab/akita/v5/mem/vm/vmprotocol"
	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/akita/v5/modeling"
	"github.com/sarchlab/akita/v5/timing"
)

// Cycle is one cycle of the component's fixed 1 GHz clock, in picoseconds.
const Cycle = timing.VTimeInPicoSec(1000)

// Config describes sector geometry and explicitly bounded execution resources.
type Config struct {
	Entries, Ways, Subentries, LookupCycles int
	Width, MSHRs, MaxWaiters, MaxInflight   int
}

// DefaultConfig uses 1024 sector tags, eight ways, 16 pages per sector, and
// 40-cycle lookup. Resource widths/capacities are adjustable model assumptions.
func DefaultConfig() Config {
	return Config{Entries: 1024, Ways: 8, Subentries: 16, LookupCycles: 40,
		Width: 4, MSHRs: 64, MaxWaiters: 64, MaxInflight: 256}
}

// Validate rejects invalid geometry or unbounded/zero resource capacities.
func (c Config) Validate() {
	if c.Entries < 1 || c.Ways < 1 || c.Entries%c.Ways != 0 ||
		c.Subentries < 1 || c.Subentries&(c.Subentries-1) != 0 || c.LookupCycles < 1 ||
		c.Width < 1 || c.MSHRs < 1 || c.MaxWaiters < 1 || c.MaxInflight < 1 {
		panic("sectortlb: invalid geometry, capacity, or lookup cycles")
	}
}

// Spec is flattened for Akita's checkpoint schema.
type Spec struct {
	Entries, Ways, Subentries, LookupCycles int
	Width, MSHRs, MaxWaiters, MaxInflight   int
	Log2PageSize                            uint64
	LowerPort                               messaging.RemotePort
}

// MakeSpec supplies the downstream GMMU port and platform page size.
func MakeSpec(c Config, lower messaging.RemotePort, log2PageSize uint64) Spec {
	return Spec{Entries: c.Entries, Ways: c.Ways, Subentries: c.Subentries,
		LookupCycles: c.LookupCycles, Width: c.Width, MSHRs: c.MSHRs,
		MaxWaiters: c.MaxWaiters, MaxInflight: c.MaxInflight,
		Log2PageSize: log2PageSize, LowerPort: lower}
}

// Config returns the immutable geometry and resource settings.
func (s Spec) Config() Config {
	return Config{Entries: s.Entries, Ways: s.Ways, Subentries: s.Subentries,
		LookupCycles: s.LookupCycles, Width: s.Width, MSHRs: s.MSHRs,
		MaxWaiters: s.MaxWaiters, MaxInflight: s.MaxInflight}
}

// Stats separates resident hits, newly issued misses, and outstanding-miss
// merges. A request is classified once, after its timed lookup can retire.
type Stats struct {
	Requests, Completed, Hits, Misses, Coalesced                                uint64
	LowerResponses, Fills, Evictions, Invalidated, StaleResponses, ResetDropped uint64
	OutstandingPeak, MSHRPeak                                                   uint64
	AdmissionBlockedCycles, LookupBlockedCycles, ResponseBlockedCycles          uint64
	LatencySum, LatencyMax                                                      timing.VTimeInPicoSec
}

type entry struct {
	Valid        bool
	PID          vm.PID
	Sector, Used uint64
	Pages        []vm.Page
}

type pending struct {
	Req          vmprotocol.TranslationReq
	Arrived, Due timing.VTimeInPicoSec
}

type miss struct {
	LowerReq vmprotocol.TranslationReq
	Waiters  []pending
}

type reply struct {
	Pending pending
	Page    vm.Page
}

// State is value-only, serializable state. MaxInflight bounds the sum of the
// lookup queue, MSHR waiters, and pending replies; ports have separate buffers.
type State struct {
	Entries          []entry
	Stamp            uint64
	Lookups          []pending
	Misses           []miss
	Replies          []reply
	Outstanding      int
	Paused, Draining bool
	DrainID          uint64
	DrainSrc         messaging.RemotePort
	Stats            Stats
}

// Resources is empty: the component communicates only through registered ports.
type Resources struct{}

// Comp is a sector TLB with Top, Bottom, and Control ports.
type Comp = modeling.Component[Spec, State, Resources]
