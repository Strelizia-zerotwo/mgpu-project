package sectortlb

import (
	"fmt"

	"github.com/sarchlab/akita/v5/mem/memcontrolprotocol"
	"github.com/sarchlab/akita/v5/mem/vm/vmprotocol"
	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/akita/v5/modeling"
	"github.com/sarchlab/akita/v5/timing"
	"github.com/sarchlab/akita/v5/tracing"
)

// Build creates a cold TLB; each GPU must get its own component.
func Build(reg modeling.Registrar, name string, spec Spec) *Comp {
	spec.Config().Validate()
	if spec.LowerPort == "" || spec.Log2PageSize == 0 || spec.Log2PageSize > 62 {
		panic("sectortlb: missing downstream port or invalid page size")
	}
	c := modeling.NewBuilder[Spec, State, Resources]().WithEngine(reg.GetEngine()).
		WithFreq(timing.GHz).WithSpec(spec).Build(name)
	c.State.Entries = make([]entry, spec.Entries)
	c.DeclarePort("Top", vmprotocol.Responder)
	c.DeclarePort("Bottom", vmprotocol.Requester)
	c.DeclarePort("Control", memcontrolprotocol.Responder)
	c.AddMiddleware(&middleware{comp: c})
	reg.RegisterComponent(c)
	for _, name := range []string{"Top", "Bottom", "Control"} {
		capacity := spec.Width
		if name == "Control" {
			capacity = 1
		}
		p := modeling.MakePortBuilder().WithRegistrar(reg).WithComponent(c).
			WithSpec(modeling.PortSpec{BufSize: capacity}).Build(name)
		c.AssignPort(name, p)
	}
	return c
}

type middleware struct{ comp *Comp }

func (m *middleware) Tick() bool { return m.step(m.comp.CurrentTime()) }

func (m *middleware) step(now timing.VTimeInPicoSec) bool {
	progress := m.control()
	state := &m.comp.State
	if !state.Paused {
		progress = m.receiveLower() || progress
		m.lookupReady(now)
		m.respond(now)
		if !state.Draining {
			m.accept(now)
		}
	}
	progress = m.finishDrain() || progress
	return progress || state.Draining || (!state.Paused && (state.Outstanding > 0 ||
		m.comp.GetPortByName("Top").PeekIncoming() != nil))
}

func (m *middleware) pageAddress(addr uint64) uint64 {
	shift := m.comp.Spec().Log2PageSize
	return (addr >> shift) << shift
}

func peak(value *uint64, candidate int) {
	if uint64(candidate) > *value {
		*value = uint64(candidate)
	}
}

func (m *middleware) accept(now timing.VTimeInPicoSec) {
	port, state, spec := m.comp.GetPortByName("Top"), &m.comp.State, m.comp.Spec()
	for range spec.Width {
		msg := port.PeekIncoming()
		if msg == nil {
			return
		}
		if state.Outstanding >= spec.MaxInflight {
			state.Stats.AdmissionBlockedCycles++
			return
		}
		req, ok := msg.(vmprotocol.TranslationReq)
		if !ok {
			panic(fmt.Sprintf("sectortlb: unexpected request %T", msg))
		}
		port.RetrieveIncoming()
		tracing.TraceReqReceive(m.comp, req)
		state.Lookups = append(state.Lookups, pending{Req: req, Arrived: now,
			Due: now + timing.VTimeInPicoSec(spec.LookupCycles)*Cycle})
		state.Stats.Requests++
		state.Outstanding++
		peak(&state.Stats.OutstandingPeak, state.Outstanding)
	}
}

func (m *middleware) tag(req vmprotocol.TranslationReq, kind string) {
	tracing.AddTaskTag(m.comp, tracing.TaskTag{TaskID: tracing.MsgIDAtReceiver(req, m.comp), What: kind})
}

func (m *middleware) lookupReady(now timing.VTimeInPicoSec) {
	state := &m.comp.State
	for range m.comp.Spec().Width {
		if len(state.Lookups) == 0 || state.Lookups[0].Due > now {
			return
		}
		p := state.Lookups[0]
		if !m.resolve(p) {
			state.Stats.LookupBlockedCycles++
			return
		}
		state.Lookups = state.Lookups[1:]
	}
}

func (m *middleware) resolve(p pending) bool {
	state, spec := &m.comp.State, m.comp.Spec()
	if page, hit := m.lookup(p.Req.PID, p.Req.VAddr); hit {
		state.Replies = append(state.Replies, reply{Pending: p, Page: page})
		state.Stats.Hits++
		m.tag(p.Req, "hit")
		return true
	}
	addr := m.pageAddress(p.Req.VAddr)
	for i := range state.Misses {
		tx := &state.Misses[i]
		if tx.LowerReq.PID != p.Req.PID || tx.LowerReq.VAddr != addr {
			continue
		}
		if len(tx.Waiters) >= spec.MaxWaiters {
			return false
		}
		tx.Waiters = append(tx.Waiters, p)
		state.Stats.Coalesced++
		m.tag(p.Req, "mshr-hit")
		return true
	}
	port := m.comp.GetPortByName("Bottom")
	if len(state.Misses) >= spec.MSHRs || !port.CanSend() {
		return false
	}
	req := vmprotocol.TranslationReq{MsgMeta: messaging.MsgMeta{
		ID: timing.GetIDGenerator().Generate(), Src: port.AsRemote(), Dst: spec.LowerPort,
		TrafficClass: "vmprotocol.TranslationReq"}, PID: p.Req.PID, VAddr: addr, DeviceID: p.Req.DeviceID}
	port.Send(req)
	state.Misses = append(state.Misses, miss{LowerReq: req, Waiters: []pending{p}})
	state.Stats.Misses++
	peak(&state.Stats.MSHRPeak, len(state.Misses))
	m.tag(p.Req, "miss")
	tracing.TraceReqInitiate(m.comp, req, tracing.MsgIDAtReceiver(p.Req, m.comp))
	return true
}

func (m *middleware) receiveLower() bool {
	port, state := m.comp.GetPortByName("Bottom"), &m.comp.State
	progress := false
	for range m.comp.Spec().Width {
		msg := port.PeekIncoming()
		if msg == nil {
			break
		}
		rsp, ok := msg.(vmprotocol.TranslationRsp)
		if !ok {
			panic(fmt.Sprintf("sectortlb: unexpected response %T", msg))
		}
		index := -1
		for i := range state.Misses {
			if state.Misses[i].LowerReq.ID == rsp.RspTo {
				index = i
				break
			}
		}
		if index >= 0 {
			m.completeMiss(index, rsp)
		} else {
			// Responses from requests discarded by Reset must not refill the cache.
			state.Stats.StaleResponses++
		}
		port.RetrieveIncoming()
		progress = true
	}
	return progress
}

func (m *middleware) completeMiss(index int, rsp vmprotocol.TranslationRsp) {
	state := &m.comp.State
	tx := state.Misses[index]
	page := rsp.Page
	size := uint64(1) << m.comp.Spec().Log2PageSize
	if rsp.Src != m.comp.Spec().LowerPort || page.PID != tx.LowerReq.PID ||
		page.VAddr != tx.LowerReq.VAddr || !page.Valid || page.IsMigrating ||
		page.PageSize != size || page.PAddr%size != 0 {
		panic("sectortlb: downstream returned an invalid or mismatched mapping")
	}
	m.fill(page)
	for _, p := range tx.Waiters {
		state.Replies = append(state.Replies, reply{Pending: p, Page: page})
	}
	tracing.TraceReqFinalize(m.comp, &tx.LowerReq)
	state.Misses = append(state.Misses[:index], state.Misses[index+1:]...)
	state.Stats.LowerResponses++
}

func (m *middleware) respond(now timing.VTimeInPicoSec) {
	port, state := m.comp.GetPortByName("Top"), &m.comp.State
	for range m.comp.Spec().Width {
		if len(state.Replies) == 0 {
			return
		}
		if !port.CanSend() {
			state.Stats.ResponseBlockedCycles++
			return
		}
		r := state.Replies[0]
		rsp := vmprotocol.TranslationRsp{MsgMeta: messaging.MsgMeta{
			ID: timing.GetIDGenerator().Generate(), Src: port.AsRemote(), Dst: r.Pending.Req.Src,
			RspTo: r.Pending.Req.ID, TrafficClass: "vmprotocol.TranslationRsp"}, Page: r.Page}
		port.Send(rsp)
		tracing.TraceReqComplete(m.comp, r.Pending.Req)
		state.Stats.Completed++
		elapsed := now - r.Pending.Arrived
		state.Stats.LatencySum += elapsed
		if elapsed > state.Stats.LatencyMax {
			state.Stats.LatencyMax = elapsed
		}
		state.Outstanding--
		state.Replies = state.Replies[1:]
	}
}
