package sectortlb

import "github.com/sarchlab/akita/v5/mem/vm"

func (m *middleware) indices(addr uint64) (sector uint64, base, sub int) {
	spec := m.comp.Spec()
	vpn := addr >> spec.Log2PageSize
	sector = vpn / uint64(spec.Subentries)
	return sector, int(sector%uint64(spec.Entries/spec.Ways)) * spec.Ways,
		int(vpn % uint64(spec.Subentries))
}

func (m *middleware) touch(e *entry) {
	m.comp.State.Stamp++
	e.Used = m.comp.State.Stamp
}

func (m *middleware) lookup(pid vm.PID, addr uint64) (vm.Page, bool) {
	sector, base, sub := m.indices(addr)
	for i := base; i < base+m.comp.Spec().Ways; i++ {
		e := &m.comp.State.Entries[i]
		if e.Valid && e.PID == pid && e.Sector == sector && e.Pages[sub].Valid {
			m.touch(e)
			return e.Pages[sub], true
		}
	}
	return vm.Page{}, false
}

func (m *middleware) fill(page vm.Page) {
	sector, base, sub := m.indices(page.VAddr)
	state := &m.comp.State
	victim, existing := -1, -1
	for i := base; i < base+m.comp.Spec().Ways; i++ {
		e := &state.Entries[i]
		if e.Valid && e.PID == page.PID && e.Sector == sector {
			existing = i
			break
		}
		if victim == -1 || !e.Valid || state.Entries[victim].Valid && e.Used < state.Entries[victim].Used {
			victim = i
		}
	}
	if existing >= 0 {
		victim = existing
	}
	e := &state.Entries[victim]
	if existing < 0 {
		if e.Valid {
			state.Stats.Evictions++
		}
		*e = entry{Valid: true, PID: page.PID, Sector: sector, Pages: make([]vm.Page, m.comp.Spec().Subentries)}
	}
	// Never infer neighboring physical mappings or prefetch other subentries.
	e.Pages[sub] = page
	m.touch(e)
	state.Stats.Fills++
}

func (m *middleware) invalidate(pid vm.PID, addresses []uint64) {
	matched := make(map[uint64]bool, len(addresses))
	for _, addr := range addresses {
		matched[m.pageAddress(addr)] = true
	}
	for i := range m.comp.State.Entries {
		e := &m.comp.State.Entries[i]
		if !e.Valid || pid != 0 && e.PID != pid {
			continue
		}
		anyValid := false
		for j := range e.Pages {
			page := &e.Pages[j]
			if page.Valid && (len(addresses) == 0 || matched[page.VAddr]) {
				page.Valid = false
				m.comp.State.Stats.Invalidated++
			}
			anyValid = anyValid || page.Valid
		}
		e.Valid = anyValid
	}
}
