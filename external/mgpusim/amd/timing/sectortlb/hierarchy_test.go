package sectortlb

import (
	"testing"

	"github.com/sarchlab/akita/v5/mem/vm/vmprotocol"
	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/akita/v5/modeling"
	"github.com/sarchlab/akita/v5/timing"
)

type hierarchyFixture struct {
	levels []*Comp
	now    timing.VTimeInPicoSec
	walks  int
}

func newHierarchyFixture() *hierarchyFixture {
	reg := modeling.NewStandaloneRegistrar(timing.NewSerialEngine())
	l3cfg := DefaultConfig()
	l2cfg := l3cfg
	l2cfg.Entries, l2cfg.LookupCycles = 128, 10
	l1cfg := l3cfg
	l1cfg.Entries, l1cfg.Ways, l1cfg.Subentries, l1cfg.LookupCycles = 16, 16, 1, 1
	l3 := Build(reg, "GPU[1].L3TLB", MakeSpec(l3cfg, "MMU.Top", 12))
	l2 := Build(reg, "GPU[1].L2TLB", MakeSpec(l2cfg, l3.GetPortByName("Top").AsRemote(), 12))
	l1 := Build(reg, "GPU[1].L1VTLB", MakeSpec(l1cfg, l2.GetPortByName("Top").AsRemote(), 12))
	levels := []*Comp{l1, l2, l3}
	for _, c := range levels {
		for _, name := range []string{"Top", "Bottom", "Control"} {
			(&noopConnection{}).PlugIn(c.GetPortByName(name))
		}
	}
	return &hierarchyFixture{levels: levels}
}

func transferHierarchyMessages(from, to messaging.Port) {
	for from.PeekOutgoing() != nil && to.CanDeliver() {
		to.Deliver(from.RetrieveOutgoing())
	}
}

func (f *hierarchyFixture) step() {
	for _, c := range f.levels {
		c.Middlewares()[0].(*middleware).step(f.now)
	}
	for i := 0; i < 2; i++ {
		upper, lower := f.levels[i], f.levels[i+1]
		transferHierarchyMessages(upper.GetPortByName("Bottom"), lower.GetPortByName("Top"))
		transferHierarchyMessages(lower.GetPortByName("Top"), upper.GetPortByName("Bottom"))
	}
	p := f.levels[2].GetPortByName("Bottom")
	for p.PeekOutgoing() != nil && p.CanDeliver() {
		req := p.RetrieveOutgoing().(vmprotocol.TranslationReq)
		f.walks++
		p.Deliver(vmprotocol.TranslationRsp{MsgMeta: messaging.MsgMeta{
			ID: timing.GetIDGenerator().Generate(), Src: "MMU.Top", Dst: p.AsRemote(), RspTo: req.ID,
		}, Page: page(req.PID, req.VAddr/4096)})
	}
	f.now += Cycle
}

func (f *hierarchyFixture) request(t *testing.T, vpn uint64) {
	t.Helper()
	p := f.levels[0].GetPortByName("Top")
	req := vmprotocol.TranslationReq{MsgMeta: messaging.MsgMeta{
		ID: timing.GetIDGenerator().Generate(), Src: "Client", Dst: p.AsRemote(),
	}, PID: 1, VAddr: vpn * 4096, DeviceID: 1}
	p.Deliver(req)
	for range 1000 {
		f.step()
		if msg := p.RetrieveOutgoing(); msg != nil {
			rsp := msg.(vmprotocol.TranslationRsp)
			if rsp.RspTo != req.ID || rsp.Page != page(1, vpn) {
				t.Fatal("wrong or misrouted mapping")
			}
			return
		}
	}
	t.Fatal("translation failed to complete")
}

// Exercise three connected TLBs, including refilling an L2 eviction from L3
// without another page walk. No cache state is preseeded by the test.
func TestThreeLevelEvictionAndResidentHit(t *testing.T) {
	f := newHierarchyFixture()
	l1, l2, l3 := f.levels[0], f.levels[1], f.levels[2]
	// One subpage in each of 129 sectors exceeds the 128-tag L2. Neighboring
	// subpages are untouched: 2048 plain entries would fail this eviction test.
	for sector := uint64(0); sector < 129; sector++ {
		f.request(t, sector*16)
	}
	if f.walks != 129 || l2.State.Stats.Evictions != 1 || l3.State.Stats.Evictions != 0 {
		t.Fatalf("wrong sector occupancy: walks=%d L2=%+v L3=%+v", f.walks, l2.State.Stats, l3.State.Stats)
	}
	f.request(t, 0)
	if f.walks != 129 || l3.State.Stats.Hits != 1 || l2.State.Stats.Misses != 130 {
		t.Fatalf("L2 eviction did not hit L3: walks=%d L2=%+v L3=%+v", f.walks, l2.State.Stats, l3.State.Stats)
	}
	f.request(t, 0)
	if l1.State.Stats.Hits != 1 || l3.State.Stats.Requests != 130 {
		t.Fatal("L1 resident hit unnecessarily reached lower levels")
	}
	f.request(t, 1) // An unfilled neighbor subpage must still walk.
	if f.walks != 130 {
		t.Fatal("neighbor subpage was incorrectly prefetched")
	}
	for _, c := range f.levels {
		s := c.State.Stats
		if s.Requests != s.Completed || s.Requests != s.Hits+s.Misses+s.Coalesced || c.State.Outstanding != 0 {
			t.Fatalf("incomplete or unclassified requests at %s: %+v", c.Name(), s)
		}
	}
	t.Log("129 cold sectors then one L3 resident hit without a page walk; L1 subsequent hit and subpage isolation passed")
}

func TestOneAndTenCycleResidentLookup(t *testing.T) {
	for _, latency := range []int{1, 10} {
		cfg := DefaultConfig()
		cfg.LookupCycles = latency
		if latency == 1 {
			cfg.Entries, cfg.Ways, cfg.Subentries = 16, 16, 1
		}
		f := makeFixture(cfg)
		f.submit(1, 4096)
		req := f.nextMiss(t)
		f.reply(req, page(1, 1))
		f.step(true)
		start := f.now
		id := f.submit(1, 4096)
		f.steps(latency)
		if _, ok := f.responses[id]; ok {
			t.Fatal("resident lookup completed too early")
		}
		f.step(true)
		if f.responses[id].Page != page(1, 1) || f.times[id]-start != timing.VTimeInPicoSec(latency)*Cycle {
			t.Fatalf("incorrect %d-cycle resident lookup", latency)
		}
	}
}
