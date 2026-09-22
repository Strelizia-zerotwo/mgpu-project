package idealmapping

import "github.com/sarchlab/akita/v5/mem/vm"

// NewDemandLocal preinstalls ordinary allocations (code, arguments, explicit
// device buffers) but leaves unified allocations absent in every GPU table.
func NewDemandLocal(log2PageSize uint64, numGPUs int) *Tables {
	t := NewIdealLocal(log2PageSize, numGPUs)
	t.ideal = false
	t.demand = true
	return t
}

// UseFixedPlacement rejects later remaps/frees once translation has begun.
// New allocations remain supported. This guard avoids silently stale TLB entries
// until a future extension implements a complete invalidation protocol.
func (t *Tables) UseFixedPlacement() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fixedPlacement = true
}

func (t *Tables) rejectLiveRemap() {
	if t.fixedPlacement && t.translationsStarted {
		panic("faultvm: live remap/free requires TLB invalidation; this baseline uses fixed placement")
	}
}

// Install installs a resolved page only in this GPU's local table. The returned
// mapping must match the authoritative allocation and be usable now.
func (v *View) Install(page vm.Page) {
	v.owner.mu.Lock()
	defer v.owner.mu.Unlock()
	current, found := v.owner.master.Find(page.PID, page.VAddr)
	if !found || current != page || !page.Valid || page.IsMigrating {
		panic("faultvm: stale, invalid, or unallocated mapping response")
	}
	if _, exists := v.table.Find(page.PID, page.VAddr); exists {
		v.table.Update(page)
	} else {
		v.table.Insert(page)
	}
}
