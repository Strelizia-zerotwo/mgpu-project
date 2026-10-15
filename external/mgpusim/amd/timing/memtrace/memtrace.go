// Package memtrace dumps the memory transactions issued by the compute units,
// together with driver-level events (allocations, kernel launches, memory
// copies), for offline page-sharing analysis.
//
// Tracing is off unless the environment variable MGPU_MEMTRACE names an output
// prefix. Two files are written:
//
//	<prefix>.bin  fixed 32-byte little-endian access records
//	<prefix>.meta one text line per driver event, in simulation-time order
//
// Access record layout:
//
//	off size field
//	0   1    kind: 'R' vector read, 'W' vector write, 'S' scalar read,
//	         'C' bytes whose value changed (emulation only; precedes its 'W')
//	1   1    GPU ID (1-based, as in GPU[n])
//	2   2    CU index within the GPU (best effort, from the component name)
//	4   4    reserved
//	8   8    time in ps
//	16  8    virtual address
//	24  8    reads: byte count; writes: bit i set if byte addr+i is written
//
// Write transactions wider than 64 bytes are split into several records.
package memtrace

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"regexp"
	"strconv"
	"sync"
)

var (
	once    sync.Once
	mu      sync.Mutex
	enabled bool
	binOut  *bufio.Writer
	metaOut *bufio.Writer
	binFile *os.File
	metaF   *os.File
)

func initTracer() {
	prefix := os.Getenv("MGPU_MEMTRACE")
	if prefix == "" {
		return
	}

	var err error
	binFile, err = os.Create(prefix + ".bin")
	if err != nil {
		panic(err)
	}
	metaF, err = os.Create(prefix + ".meta")
	if err != nil {
		panic(err)
	}
	binOut = bufio.NewWriterSize(binFile, 1<<22)
	metaOut = bufio.NewWriterSize(metaF, 1<<16)
	enabled = true
}

// Enabled reports whether tracing is on.
func Enabled() bool {
	once.Do(initTracer)
	return enabled
}

var nameRe = regexp.MustCompile(`GPU\[(\d+)\](?:.*?SA\[(\d+)\])?(?:.*?CU\[(\d+)\])?`)

// Location identifies the GPU and CU that issue requests.
type Location struct {
	GPU uint8
	CU  uint16
}

// ParseLocation extracts the GPU and CU index from a component name such as
// "GPU[2].SA[3].CU[01]". CU indices are flattened as SA*16+CU.
func ParseLocation(name string) Location {
	m := nameRe.FindStringSubmatch(name)
	if m == nil {
		return Location{}
	}
	gpu, _ := strconv.Atoi(m[1])
	sa, _ := strconv.Atoi(m[2])
	cu, _ := strconv.Atoi(m[3])
	return Location{GPU: uint8(gpu), CU: uint16(sa*16 + cu)}
}

func writeRecord(kind byte, loc Location, now, addr, payload uint64) {
	var rec [32]byte
	rec[0] = kind
	rec[1] = loc.GPU
	binary.LittleEndian.PutUint16(rec[2:], loc.CU)
	binary.LittleEndian.PutUint64(rec[8:], now)
	binary.LittleEndian.PutUint64(rec[16:], addr)
	binary.LittleEndian.PutUint64(rec[24:], payload)
	_, _ = binOut.Write(rec[:])
}

// Read records a read transaction.
func Read(kind byte, loc Location, now, addr, size uint64) {
	if !Enabled() {
		return
	}
	mu.Lock()
	writeRecord(kind, loc, now, addr, size)
	mu.Unlock()
}

// Write records a write transaction. A nil dirty mask means all bytes.
func Write(loc Location, now, addr uint64, size int, dirty []bool) {
	if !Enabled() {
		return
	}
	mu.Lock()
	for base := 0; base < size; base += 64 {
		var mask uint64
		for i := 0; i < 64 && base+i < size; i++ {
			if dirty == nil || dirty[base+i] {
				mask |= 1 << uint(i)
			}
		}
		if mask != 0 {
			writeRecord('W', loc, now, addr+uint64(base), mask)
		}
	}
	mu.Unlock()
}

// Event records a driver-level event line: "<time_ps> <fields...>".
func Event(now uint64, format string, args ...interface{}) {
	if !Enabled() {
		return
	}
	mu.Lock()
	fmt.Fprintf(metaOut, "%d ", now)
	fmt.Fprintf(metaOut, format, args...)
	metaOut.WriteByte('\n')
	mu.Unlock()
}

// Close flushes and closes the trace files.
func Close() {
	if !Enabled() {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	flushMergers()
	_ = binOut.Flush()
	_ = metaOut.Flush()
	_ = binFile.Close()
	_ = metaF.Close()
	enabled = false
}

// lineMerger folds the per-lane accesses of the functional emulator into
// 64-byte line transactions, matching the coalesced timing-mode records.
type lineMerger struct {
	valid bool
	kind  byte
	loc   Location
	now   uint64
	line  uint64
	mask  uint64
	cmask uint64
}

var mergers = map[Location]*lineMerger{}

func (m *lineMerger) flush() {
	if !m.valid {
		return
	}
	if m.kind == 'W' {
		writeRecord('C', m.loc, m.now, m.line, m.cmask)
		writeRecord('W', m.loc, m.now, m.line, m.mask)
	} else {
		lo := uint64(bits.TrailingZeros64(m.mask))
		hi := 63 - uint64(bits.LeadingZeros64(m.mask))
		writeRecord(m.kind, m.loc, m.now, m.line+lo, hi-lo+1)
	}
	m.valid = false
}

// EmuAccess records one functional-emulator access of size bytes at addr.
func EmuAccess(kind byte, loc Location, now, addr, size uint64) {
	emuAccess(kind, loc, now, addr, size, nil, nil)
}

// EmuWrite records a functional-emulator write, marking the bytes whose value
// differs from what memory held before.
func EmuWrite(loc Location, now, addr uint64, old, data []byte) {
	emuAccess('W', loc, now, addr, uint64(len(data)), old, data)
}

func emuAccess(kind byte, loc Location, now, addr, size uint64, old, data []byte) {
	if !Enabled() || size == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	m := mergers[loc]
	if m == nil {
		m = &lineMerger{}
		mergers[loc] = m
	}
	for size > 0 {
		line := addr &^ 63
		off := addr - line
		n := 64 - off
		if n > size {
			n = size
		}
		var mask uint64
		if n == 64 {
			mask = ^uint64(0)
		} else {
			mask = ((uint64(1) << n) - 1) << off
		}
		if !m.valid || m.kind != kind || m.now != now || m.line != line {
			m.flush()
			*m = lineMerger{valid: true, kind: kind, loc: loc, now: now, line: line}
		}
		m.mask |= mask
		for i := uint64(0); i < n && data != nil; i++ {
			if old[i] != data[i] {
				m.cmask |= 1 << (off + i)
			}
		}
		if data != nil {
			old, data = old[n:], data[n:]
		}
		addr += n
		size -= n
	}
}

func flushMergers() {
	for _, m := range mergers {
		m.flush()
	}
}
