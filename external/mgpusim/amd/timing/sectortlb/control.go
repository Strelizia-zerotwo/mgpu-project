package sectortlb

import (
	"github.com/sarchlab/akita/v5/mem/memcontrolprotocol"
	"github.com/sarchlab/akita/v5/messaging"
	"github.com/sarchlab/akita/v5/timing"
	"github.com/sarchlab/akita/v5/tracing"
)

func (m *middleware) control() bool {
	port, state := m.comp.GetPortByName("Control"), &m.comp.State
	if state.Draining {
		return false
	}
	msg := port.PeekIncoming()
	if msg == nil || !port.CanSend() {
		return false
	}
	req, ok := msg.(memcontrolprotocol.Req)
	if !ok {
		panic("sectortlb: unexpected control message")
	}
	success, reason := true, ""
	switch req.Command {
	case memcontrolprotocol.CmdPause:
		state.Paused = true
	case memcontrolprotocol.CmdEnable:
		state.Paused = false
	case memcontrolprotocol.CmdDrain:
		state.Paused = false
		state.Draining = true
		state.DrainID, state.DrainSrc = req.ID, req.Src
	case memcontrolprotocol.CmdInvalidate:
		if !state.Paused || state.Outstanding != 0 {
			success, reason = false, memcontrolprotocol.ErrMustBePausedOrDrained
		} else {
			m.invalidate(req.PID, req.Addresses)
		}
	case memcontrolprotocol.CmdReset:
		m.reset()
	default:
		success, reason = false, memcontrolprotocol.ErrUnsupported
	}
	port.RetrieveIncoming()
	tracing.ForgetMsgIDAtReceiver(req.ID, m.comp)
	if !state.Draining {
		m.controlReply(req.Command, req.Src, req.ID, success, reason)
	}
	return true
}

func (m *middleware) finishDrain() bool {
	state := &m.comp.State
	if !state.Draining || state.Outstanding != 0 || !m.comp.GetPortByName("Control").CanSend() {
		return false
	}
	m.controlReply(memcontrolprotocol.CmdDrain, state.DrainSrc, state.DrainID, true, "")
	state.Draining = false
	state.Paused = true
	state.DrainID = 0
	state.DrainSrc = ""
	return true
}

func (m *middleware) controlReply(cmd memcontrolprotocol.Command, dst messaging.RemotePort,
	id uint64, success bool, reason string) {
	port := m.comp.GetPortByName("Control")
	port.Send(memcontrolprotocol.Rsp{MsgMeta: messaging.MsgMeta{
		ID: timing.GetIDGenerator().Generate(), Src: port.AsRemote(), Dst: dst, RspTo: id,
		TrafficClass: "memcontrolprotocol.Rsp"}, Command: cmd, Success: success, Error: reason})
}

func (m *middleware) reset() {
	state := &m.comp.State
	for _, p := range state.Lookups {
		tracing.EndReqInOnReset(m.comp, p.Req.ID)
	}
	for _, tx := range state.Misses {
		for _, p := range tx.Waiters {
			tracing.EndReqInOnReset(m.comp, p.Req.ID)
		}
		tracing.EndTaskOnReset(m.comp, tx.LowerReq.ID)
	}
	for _, r := range state.Replies {
		tracing.EndReqInOnReset(m.comp, r.Pending.Req.ID)
	}
	stats := state.Stats
	stats.ResetDropped += uint64(state.Outstanding)
	*state = State{Entries: make([]entry, m.comp.Spec().Entries), Stats: stats}
	for _, name := range []string{"Top", "Bottom"} {
		port := m.comp.GetPortByName(name)
		for msg := port.RetrieveIncoming(); msg != nil; msg = port.RetrieveIncoming() {
			tracing.ForgetMsgIDAtReceiver(msg.Meta().ID, m.comp)
		}
	}
}
