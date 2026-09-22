// Package idealmapping models instantaneous propagation of page-table mappings.
// It does not model host faults, data migration, or TLB shootdown.
package idealmapping

import (
	"fmt"
	"sync"

	"github.com/sarchlab/akita/v5/mem/vm"
)

// Stats counts MMU-side Find calls, including response-backpressure retries.
// The categories are mutually exclusive; driver lookups are not counted.
type Stats struct {
	Lookups   uint64
	Hits      uint64
	Missing   uint64
	Invalid   uint64
	Migrating uint64
}

// Tables owns the authoritative driver table and optional per-GPU replicas.
// The lock makes a mapping mutation atomic with respect to all local lookups.
// CPU execution of this synchronization consumes no simulated cycles.
type Tables struct {
	mu                  sync.Mutex
	master              vm.PageTable
	views               []*View
	log2PageSize        uint64
	ideal               bool
	demand              bool
	fixedPlacement      bool
	translationsStarted bool
}

// View is the page table used by one MMU. Its counters exclude driver accesses.
type View struct {
	owner *Tables
	table vm.PageTable
	stats Stats
}

// NewShared wraps the original shared table without changing lookup semantics.
func NewShared(log2PageSize uint64) *Tables {
	t := &Tables{master: vm.NewPageTable(log2PageSize), log2PageSize: log2PageSize}
	t.views = []*View{{owner: t, table: t.master}}
	return t
}

// NewIdealLocal creates independent local tables before any memory allocations.
// Every later Insert, Update, and Remove is propagated to all replicas for free.
func NewIdealLocal(log2PageSize uint64, numGPUs int) *Tables {
	if numGPUs < 1 {
		panic("ideal-local page tables require at least one GPU")
	}
	t := &Tables{master: vm.NewPageTable(log2PageSize), log2PageSize: log2PageSize, ideal: true}
	for i := 0; i < numGPUs; i++ {
		t.views = append(t.views, &View{owner: t, table: vm.NewPageTable(log2PageSize)})
	}
	return t
}

// View returns the zero-based MMU view (GPU 1 uses index 0).
func (t *Tables) View(index int) *View { return t.views[index] }

// GetLog2PageSize exposes the page size for Akita MMU validation.
func (t *Tables) GetLog2PageSize() uint64 { return t.log2PageSize }

// Insert publishes a new mapping before the driver returns from allocation.
func (t *Tables) Insert(page vm.Page) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.master.Insert(page)
	if t.ideal || (t.demand && !page.Unified) {
		for _, view := range t.views {
			view.table.Insert(page)
		}
	}
}

// Update publishes the current physical address and all page metadata.
// This synchronizes page tables only; a caller must still handle TLB invalidation.
func (t *Tables) Update(page vm.Page) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rejectLiveRemap()
	t.master.Update(page)
	if t.demand {
		for _, view := range t.views {
			if _, found := view.table.Find(page.PID, page.VAddr); found {
				view.table.Update(page)
			}
		}
	}
	if t.ideal {
		for _, view := range t.views {
			view.table.Update(page)
		}
	}
}

// Remove removes the mapping from the authoritative table and all replicas.
func (t *Tables) Remove(pid vm.PID, addr uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rejectLiveRemap()
	t.master.Remove(pid, addr)
	if t.demand {
		for _, view := range t.views {
			if _, found := view.table.Find(pid, addr); found {
				view.table.Remove(pid, addr)
			}
		}
	}
	if t.ideal {
		for _, view := range t.views {
			view.table.Remove(pid, addr)
		}
	}
}

// Find serves driver queries without contributing to MMU lookup statistics.
func (t *Tables) Find(pid vm.PID, addr uint64) (vm.Page, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.master.Find(pid, addr)
}

// ReverseLookup serves physical-to-virtual queries made by the allocator.
func (t *Tables) ReverseLookup(addr uint64) (vm.Page, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.master.ReverseLookup(addr)
}

// Find uses only this MMU's table; it never falls back to the master table.
// Ideal mode fails explicitly on unmapped, invalid, or migrating pages instead
// of inventing a physical address or bypassing a migration in progress.
func (v *View) Find(pid vm.PID, addr uint64) (vm.Page, bool) {
	v.owner.mu.Lock()
	defer v.owner.mu.Unlock()
	v.owner.translationsStarted = true
	page, found := v.table.Find(pid, addr)
	v.stats.Lookups++
	switch {
	case !found:
		v.stats.Missing++
	case !page.Valid:
		v.stats.Invalid++
	case page.IsMigrating:
		v.stats.Migrating++
	default:
		v.stats.Hits++
	}
	if v.owner.ideal && (!found || !page.Valid || page.IsMigrating) {
		panic(fmt.Sprintf("ideal-local mapping unavailable: pid=%d vaddr=%#x found=%t valid=%t migrating=%t",
			pid, addr, found, page.Valid, page.IsMigrating))
	}
	return page, found
}

// Snapshot returns consistent MMU lookup counters.
func (v *View) Snapshot() Stats {
	v.owner.mu.Lock()
	defer v.owner.mu.Unlock()
	return v.stats
}

// IdealLocal reports whether this view belongs to an ideal local replica.
func (v *View) IdealLocal() bool { return v.owner.ideal }

// GetLog2PageSize exposes this view's page size to Akita's validation.
func (v *View) GetLog2PageSize() uint64 { return v.owner.log2PageSize }

// Insert routes mutations through the owner to keep every replica consistent.
func (v *View) Insert(page vm.Page) { v.owner.Insert(page) }

// Update routes mutations through the owner to keep every replica consistent.
func (v *View) Update(page vm.Page) { v.owner.Update(page) }

// Remove routes mutations through the owner to keep every replica consistent.
func (v *View) Remove(pid vm.PID, addr uint64) { v.owner.Remove(pid, addr) }

// ReverseLookup reads the local replica without affecting lookup counters.
func (v *View) ReverseLookup(addr uint64) (vm.Page, bool) {
	v.owner.mu.Lock()
	defer v.owner.mu.Unlock()
	return v.table.ReverseLookup(addr)
}
