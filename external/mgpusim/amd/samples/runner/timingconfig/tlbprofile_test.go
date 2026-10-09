package timingconfig

import (
	"testing"

	"github.com/sarchlab/akita/v5/mem/vm/tlb"
	"github.com/sarchlab/akita/v5/simulation"
	"github.com/sarchlab/mgpusim/v5/amd/samples/runner/timingconfig/tlbprofile"
	"github.com/sarchlab/mgpusim/v5/amd/timing/faultvm"
	"github.com/sarchlab/mgpusim/v5/amd/timing/sectortlb"
)

func TestPaperTLBCapacityWiring(t *testing.T) {
	s := simulation.MakeBuilder().WithoutMonitoring().WithOutputFileName(t.TempDir() + "/sim").Build()
	defer s.Terminate()
	MakeBuilder().WithSimulation(s).WithTLBProfile(tlbprofile.PaperCapacity).
		WithFaultVM("demand-l3", faultvm.DefaultConfig()).Build()
	expected := map[string][4]int{
		"GPU[1].SA[0].L1VTLB[0]": {16, 16, 1, 1},
		"GPU[1].SA[0].L1STLB":    {16, 16, 1, 1},
		"GPU[1].SA[0].L1ITLB":    {16, 16, 1, 1},
		"GPU[1].L2TLB":           {128, 8, 16, 10},
		"GPU[1].L3TLB":           {1024, 8, 16, 40},
	}
	for name, want := range expected {
		c, ok := s.GetComponentByName(name).(*sectortlb.Comp)
		if !ok {
			t.Fatalf("%s is not a finite sector TLB", name)
		}
		spec := c.Spec()
		if got := [4]int{spec.Entries, spec.Ways, spec.Subentries, spec.LookupCycles}; got != want {
			t.Fatalf("%s geometry: got %v want %v", name, got, want)
		}
	}
	l1 := s.GetComponentByName("GPU[1].SA[0].L1VTLB[0]").(*sectortlb.Comp)
	l2 := s.GetComponentByName("GPU[1].L2TLB").(*sectortlb.Comp)
	if l1.Spec().LowerPort != l2.GetPortByName("Top").AsRemote() || l2.Spec().LowerPort != "GPU[1].L3TLB.Top" {
		t.Fatal("TLB hierarchy bypassed a configured level")
	}
}

func TestLegacyTLBCapacityUnchanged(t *testing.T) {
	s := simulation.MakeBuilder().WithoutMonitoring().WithOutputFileName(t.TempDir() + "/sim").Build()
	defer s.Terminate()
	MakeBuilder().WithSimulation(s).Build()
	l2, ok := s.GetComponentByName("GPU[1].L2TLB").(*tlb.Comp)
	if !ok || l2.Spec().NumSets*l2.Spec().NumWays != 1048576 || l2.Spec().Latency != 4 {
		t.Fatal("legacy L2 geometry or implementation changed")
	}
	l1, ok := s.GetComponentByName("GPU[1].SA[0].L1VTLB[0]").(*tlb.Comp)
	if !ok || l1.Spec().NumSets*l1.Spec().NumWays != 256 || l1.Spec().Latency != 2 {
		t.Fatal("legacy vector L1 geometry or implementation changed")
	}
}
