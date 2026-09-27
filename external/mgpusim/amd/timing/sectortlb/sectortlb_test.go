package sectortlb

import (
	"testing"

	"github.com/sarchlab/akita/v5/hooking"
	"github.com/sarchlab/akita/v5/mem/memcontrolprotocol"
	"github.com/sarchlab/akita/v5/mem/vm"
	"github.com/sarchlab/akita/v5/mem/vm/vmprotocol"
	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/akita/v5/modeling"
	"github.com/sarchlab/akita/v5/timing"
)

type noopConnection struct{ hooking.HookableBase }

func (c *noopConnection) Name() string                     { return "test" }
func (c *noopConnection) PlugIn(p messaging.Port)          { p.SetConnection(c) }
func (c *noopConnection) Unplug(_ messaging.Port)          {}
func (c *noopConnection) NotifyAvailable(_ messaging.Port) {}
func (c *noopConnection) NotifySend()                      {}

type fixture struct {
	c         *Comp
	m         *middleware
	now       timing.VTimeInPicoSec
	responses map[uint64]vmprotocol.TranslationRsp
	times     map[uint64]timing.VTimeInPicoSec
}

func makeFixture(config Config) *fixture {
	reg := modeling.NewStandaloneRegistrar(timing.NewSerialEngine())
	c := Build(reg, "GPU[1].L3TLB", MakeSpec(config, "MMU.Top", 12))
	for _, name := range []string{"Top", "Bottom", "Control"} {
		(&noopConnection{}).PlugIn(c.GetPortByName(name))
	}
	return &fixture{c: c, m: c.Middlewares()[0].(*middleware),
		responses: make(map[uint64]vmprotocol.TranslationRsp), times: make(map[uint64]timing.VTimeInPicoSec)}
}

func page(pid vm.PID, vpn uint64) vm.Page {
	// Deliberately non-contiguous physical pages within a virtual sector.
	return vm.Page{PID: pid, VAddr: vpn * 4096, PAddr: (uint64(pid)*1024 + vpn*3) * 4096,
		PageSize: 4096, Valid: true, Unified: true, DeviceID: 1}
}

func (f *fixture) submit(pid vm.PID, addr uint64) uint64 {
	p := f.c.GetPortByName("Top")
	if !p.CanDeliver() {
		panic("test input overflow")
	}
	req := vmprotocol.TranslationReq{MsgMeta: messaging.MsgMeta{ID: timing.GetIDGenerator().Generate(),
		Src: "L2.Bottom", Dst: p.AsRemote(), TrafficClass: "vmprotocol.TranslationReq"}, PID: pid, VAddr: addr, DeviceID: 1}
	p.Deliver(req)
	return req.ID
}

func (f *fixture) step(collect bool) {
	f.m.step(f.now)
	if collect {
		p := f.c.GetPortByName("Top")
		for msg := p.RetrieveOutgoing(); msg != nil; msg = p.RetrieveOutgoing() {
			rsp := msg.(vmprotocol.TranslationRsp)
			if _, exists := f.responses[rsp.RspTo]; exists {
				panic("duplicate response")
			}
			f.responses[rsp.RspTo] = rsp
			f.times[rsp.RspTo] = f.now
		}
	}
	f.now += Cycle
}

func (f *fixture) steps(n int) {
	for range n {
		f.step(true)
	}
}

func (f *fixture) nextMiss(t *testing.T) vmprotocol.TranslationReq {
	t.Helper()
	p := f.c.GetPortByName("Bottom")
	for range 1000 {
		if msg := p.RetrieveOutgoing(); msg != nil {
			return msg.(vmprotocol.TranslationReq)
		}
		f.step(true)
	}
	t.Fatal("missing downstream request")
	return vmprotocol.TranslationReq{}
}

func (f *fixture) reply(req vmprotocol.TranslationReq, data vm.Page) {
	p := f.c.GetPortByName("Bottom")
	if !p.CanDeliver() {
		panic("test lower input overflow")
	}
	p.Deliver(vmprotocol.TranslationRsp{MsgMeta: messaging.MsgMeta{ID: timing.GetIDGenerator().Generate(),
		RspTo: req.ID, Src: "MMU.Top", Dst: p.AsRemote(), TrafficClass: "vmprotocol.TranslationRsp"}, Page: data})
}

func TestColdMissAndFortyCycleResidentHit(t *testing.T) {
	f := makeFixture(DefaultConfig())
	first := f.submit(1, 4096+128)
	req := f.nextMiss(t)
	if f.now != 41*Cycle || req.VAddr != 4096 {
		t.Fatalf("wrong lookup latency/alignment: now=%d req=%+v", f.now, req)
	}
	data := page(1, 1)
	f.reply(req, data)
	f.step(true)
	if f.responses[first].Page != data {
		t.Fatal("lost first mapping")
	}
	start := f.now
	id := f.submit(1, 4096)
	f.steps(40)
	if _, ok := f.responses[id]; ok {
		t.Fatal("hit returned before lookup latency")
	}
	f.step(true)
	if f.responses[id].Page != data || f.times[id]-start != 40*Cycle {
		t.Fatal("warm hit timing or mapping incorrect")
	}
	s := f.c.State.Stats
	if s.Hits != 1 || s.Misses != 1 || s.Completed != 2 || f.c.GetPortByName("Bottom").PeekOutgoing() != nil {
		t.Fatalf("hit unnecessarily walked GMMU: %+v", s)
	}
}

func TestSubentryFillPIDIsolationAndNoNeighborPrefetch(t *testing.T) {
	f := makeFixture(DefaultConfig())
	for _, data := range []vm.Page{page(1, 1), page(1, 2), page(2, 1)} {
		id := f.submit(data.PID, data.VAddr)
		req := f.nextMiss(t)
		f.reply(req, data)
		f.step(true)
		if f.responses[id].Page != data {
			t.Fatal("wrong physical mapping")
		}
	}
	for _, data := range []vm.Page{page(1, 1), page(1, 2), page(2, 1)} {
		f.submit(data.PID, data.VAddr)
	}
	f.steps(41)
	if s := f.c.State.Stats; s.Misses != 3 || s.Hits != 3 || s.Evictions != 0 {
		t.Fatalf("sector/PID alias: %+v", s)
	}
}

func TestSectorLRUAndSetIsolation(t *testing.T) {
	config := DefaultConfig()
	config.Entries = 4
	config.Ways = 2
	f := makeFixture(config)
	f.m.fill(page(1, 0))  // sector0, set0
	f.m.fill(page(1, 32)) // sector2, set0
	f.m.fill(page(1, 16)) // sector1, set1
	if _, ok := f.m.lookup(1, 0); !ok {
		t.Fatal("missing seed mapping")
	}
	f.m.fill(page(1, 64)) // sector4 evicts sector2
	for _, vpn := range []uint64{0, 16, 64} {
		if _, ok := f.m.lookup(1, vpn*4096); !ok {
			t.Fatalf("incorrectly evicted vpn %d", vpn)
		}
	}
	if _, ok := f.m.lookup(1, 32*4096); ok {
		t.Fatal("LRU victim remained")
	}
	f.m.fill(page(1, 65)) // same sector, independent subentry
	if _, ok := f.m.lookup(1, 64*4096); !ok || f.c.State.Stats.Evictions != 1 {
		t.Fatal("subentry fill evicted its own sector")
	}
}

func TestExactPageCoalescingWithDistinctSubentryMisses(t *testing.T) {
	f := makeFixture(DefaultConfig())
	ids := []uint64{f.submit(1, 4096), f.submit(1, 4097), f.submit(1, 8192), f.submit(2, 4096)}
	a, b, c := f.nextMiss(t), f.nextMiss(t), f.nextMiss(t)
	if s := f.c.State.Stats; s.Misses != 3 || s.Coalesced != 1 {
		t.Fatalf("incorrect merges: %+v", s)
	}
	// Out-of-order replies must match request ID, not queue position.
	for _, req := range []vmprotocol.TranslationReq{c, a, b} {
		f.reply(req, page(req.PID, req.VAddr/4096))
	}
	f.step(true)
	for i, id := range ids {
		pid, vpn := vm.PID(1), uint64(1)
		if i == 2 {
			vpn = 2
		}
		if i == 3 {
			pid = 2
		}
		if f.responses[id].Page != page(pid, vpn) {
			t.Fatalf("lost/misrouted id %d", id)
		}
	}
}

func TestFiniteResourcesBackpressureAndConservation(t *testing.T) {
	config := DefaultConfig()
	config.Width = 2
	config.MSHRs = 1
	config.MaxWaiters = 1
	config.MaxInflight = 2
	f := makeFixture(config)
	f.submit(1, 0)
	f.submit(1, 4096)
	f.step(false)
	f.submit(1, 8192)
	f.submit(1, 12288)
	first := f.nextMiss(t)
	f.steps(3)
	if s := f.c.State.Stats; s.OutstandingPeak != 2 || s.MSHRPeak != 1 ||
		s.AdmissionBlockedCycles == 0 || s.LookupBlockedCycles == 0 {
		t.Fatalf("capacities did not backpressure: %+v", s)
	}
	f.reply(first, page(first.PID, first.VAddr/4096))
	f.step(true)
	for range 3 {
		req := f.nextMiss(t)
		f.reply(req, page(req.PID, req.VAddr/4096))
		f.step(true)
	}
	s := f.c.State.Stats
	if len(f.responses) != 4 || s.Requests != s.Completed || f.c.State.Outstanding != 0 || len(f.c.State.Misses) != 0 {
		t.Fatalf("lost request or resource leak: %+v", s)
	}
}

func TestWaiterLimitAndBlockedUpstream(t *testing.T) {
	config := DefaultConfig()
	config.Width = 2
	config.MaxWaiters = 2
	f := makeFixture(config)
	f.submit(1, 4096)
	f.submit(1, 4097)
	f.step(false)
	f.submit(1, 4098)
	f.step(false)
	req := f.nextMiss(t)
	f.steps(2)
	if s := f.c.State.Stats; s.Misses != 1 || s.Coalesced != 1 || s.LookupBlockedCycles == 0 {
		t.Fatalf("waiter cap ignored: %+v", s)
	}
	f.reply(req, page(1, 1))
	f.step(false)
	f.step(false)
	if s := f.c.State.Stats; s.ResponseBlockedCycles == 0 || s.Hits != 1 || s.Completed != 2 {
		t.Fatalf("no response backpressure: %+v", s)
	}
	f.step(true)
	f.step(true)
	if len(f.responses) != 3 || f.c.State.Outstanding != 0 {
		t.Fatal("response lost after backpressure cleared")
	}
}

func (f *fixture) command(cmd memcontrolprotocol.Command, pid vm.PID, addrs []uint64) uint64 {
	p := f.c.GetPortByName("Control")
	req := memcontrolprotocol.Req{MsgMeta: messaging.MsgMeta{ID: timing.GetIDGenerator().Generate(),
		Src: "CP.Control", Dst: p.AsRemote(), TrafficClass: "memcontrolprotocol.Req"},
		Command: cmd, PID: pid, Addresses: addrs}
	p.Deliver(req)
	return req.ID
}

func (f *fixture) ack(t *testing.T, success bool) {
	t.Helper()
	msg := f.c.GetPortByName("Control").RetrieveOutgoing()
	if msg == nil {
		t.Fatal("missing control ack")
	}
	if rsp := msg.(memcontrolprotocol.Rsp); rsp.Success != success {
		t.Fatalf("unexpected ack: %+v", rsp)
	}
}

func TestPauseDrainEnableAndFilteredInvalidation(t *testing.T) {
	f := makeFixture(DefaultConfig())
	f.m.fill(page(1, 1))
	f.m.fill(page(1, 2))
	f.m.fill(page(2, 1))
	f.command(memcontrolprotocol.CmdInvalidate, 0, nil)
	f.step(true)
	f.ack(t, false)
	f.command(memcontrolprotocol.CmdPause, 0, nil)
	f.step(true)
	f.ack(t, true)
	f.command(memcontrolprotocol.CmdInvalidate, 1, []uint64{4097})
	f.step(true)
	f.ack(t, true)
	if _, hit := f.m.lookup(1, 4096); hit {
		t.Fatal("target subentry not invalidated")
	}
	for _, p := range []vm.Page{page(1, 2), page(2, 1)} {
		if _, hit := f.m.lookup(p.PID, p.VAddr); !hit {
			t.Fatal("invalidation affected unrelated PID/subentry")
		}
	}
	f.submit(1, 8192)
	f.steps(45)
	if f.c.State.Stats.Requests != 0 {
		t.Fatal("paused TLB admitted a request")
	}
	f.command(memcontrolprotocol.CmdEnable, 0, nil)
	f.step(true)
	f.ack(t, true)
	f.submit(1, 12288) // remains at ingress during Drain
	f.command(memcontrolprotocol.CmdDrain, 0, nil)
	f.step(true)
	if f.c.GetPortByName("Control").PeekOutgoing() != nil {
		t.Fatal("drain acknowledged pending work")
	}
	f.steps(40)
	f.ack(t, true)
	if !f.c.State.Paused || f.c.State.Stats.Completed != 1 || f.c.GetPortByName("Top").PeekIncoming() == nil {
		t.Fatal("drain lost input or failed to pause")
	}
	f.command(memcontrolprotocol.CmdInvalidate, 0, nil)
	f.step(true)
	f.ack(t, true)
	if _, hit := f.m.lookup(2, 4096); hit {
		t.Fatal("full invalidation left mapping")
	}
}

func TestResetDiscardsInflightAndRejectsStaleRefill(t *testing.T) {
	f := makeFixture(DefaultConfig())
	old := f.submit(1, 4096)
	req := f.nextMiss(t)
	f.command(memcontrolprotocol.CmdPause, 0, nil)
	f.step(true)
	f.ack(t, true)
	f.command(memcontrolprotocol.CmdInvalidate, 1, nil)
	f.step(true)
	f.ack(t, false)
	f.command(memcontrolprotocol.CmdReset, 0, nil)
	f.step(true)
	f.ack(t, true)
	current := f.submit(1, 4096)
	newReq := f.nextMiss(t)
	f.reply(req, page(1, 1))
	f.step(true)
	if _, hit := f.m.lookup(1, 4096); hit || len(f.responses) != 0 {
		t.Fatal("stale reply refilled/completed new miss")
	}
	f.reply(newReq, page(1, 1))
	f.step(true)
	if _, ok := f.responses[old]; ok || f.responses[current].Page != page(1, 1) {
		t.Fatal("reset confused request IDs")
	}
	s := f.c.State.Stats
	if s.StaleResponses != 1 || s.ResetDropped != 1 || s.Requests != s.Completed+s.ResetDropped {
		t.Fatalf("bad reset accounting: %+v", s)
	}
}

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected explicit rejection")
		}
	}()
	fn()
}

func TestInvalidConfigurationsAndWrongMappingsRejected(t *testing.T) {
	for _, change := range []func(*Config){
		func(c *Config) { c.Entries = 3 }, func(c *Config) { c.Ways = 0 }, func(c *Config) { c.Subentries = 3 },
		func(c *Config) { c.LookupCycles = 0 }, func(c *Config) { c.MSHRs = 0 }, func(c *Config) { c.MaxWaiters = 0 },
		func(c *Config) { c.MaxInflight = 0 }, func(c *Config) { c.Width = 0 },
	} {
		config := DefaultConfig()
		change(&config)
		expectPanic(t, func() { makeFixture(config) })
	}
	for _, change := range []func(*vm.Page){
		func(p *vm.Page) { p.PID = 2 }, func(p *vm.Page) { p.VAddr = 8192 }, func(p *vm.Page) { p.Valid = false },
		func(p *vm.Page) { p.IsMigrating = true }, func(p *vm.Page) { p.PageSize = 8192 }, func(p *vm.Page) { p.PAddr++ },
	} {
		f := makeFixture(DefaultConfig())
		f.submit(1, 4096)
		req := f.nextMiss(t)
		data := page(1, 1)
		change(&data)
		f.reply(req, data)
		expectPanic(t, func() { f.step(true) })
	}
}
