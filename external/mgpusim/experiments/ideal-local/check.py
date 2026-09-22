"""Validate an actual shared/ideal FIR pair; no third-party Python packages."""
import sqlite3
import sys
from pathlib import Path


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def check(directory, mode):
    log = (directory / f"{mode}.log").read_text()
    require("Passed!" in log, f"{mode}: FIR verification did not pass")
    database = directory / f"{mode}.sqlite3"
    with sqlite3.connect(database.resolve().as_uri() + "?mode=ro", uri=True) as conn:
        rows = conn.execute("SELECT Location, What, Value FROM mgpusim_metrics").fetchall()
    metrics = {(location, what): (float(value) if value is not None else float("nan"))
               for location, what, value in rows}
    mmus = sorted(location for location, what in metrics if what == "mmu_ptw_completed")
    expected = ["GPU[1].MMU", "GPU[2].MMU"] if mode == "ideal" else ["MMU"]
    require(mmus == expected, f"{mode}: unexpected MMUs: {mmus}")
    print(f"{mode}: kernel_time={metrics['Driver', 'kernel_time']:.9g} s")
    for mmu in mmus:
        def get(name):
            return metrics[mmu, name]
        walks = get("mmu_ptw_completed")
        lookups = get("mmu_pt_lookup_count")
        hits = get("mmu_pt_hit_count")
        require(walks > 0, f"{mmu}: no completed walks; the experiment was not exercised")
        require(lookups >= walks and lookups == hits, f"{mmu}: unsuccessful page-table lookup")
        require(get("mmu_ptw_configured_cycles") == 100, f"{mmu}: walk latency changed")
        require(get("mmu_frequency_hz") == 1e9, f"{mmu}: unexpected MMU frequency")
        require(get("mmu_ideal_local") == (mode == "ideal"), f"{mmu}: wrong mode")
        for key in ("mmu_pt_missing_count", "mmu_pt_invalid_count", "mmu_pt_migrating_count"):
            require(get(key) == 0, f"{mmu}: nonzero {key}")
        require(get("mmu_translation_average_latency") >= 100e-9, f"{mmu}: walk delay was bypassed")
        print(f"  {mmu}: completed PTW={walks:g}, PT hits/lookups={hits:g}/{lookups:g}, "
              f"missing/invalid/migrating=0/0/0, average translation="
              f"{get('mmu_translation_average_latency') * 1e9:g} ns")
    for gpu in (1, 2):
        misses = sum(value for (loc, what), value in metrics.items()
                     if loc.startswith(f"GPU[{gpu}].") and "TLB" in loc and what == "miss")
        require(misses > 0, f"GPU[{gpu}]: no observed TLB misses")
        print(f"  GPU[{gpu}]: aggregate TLB misses={misses:g}")
    return metrics


if __name__ == "__main__":
    result_dir = Path(sys.argv[1])
    for selected_mode in ("shared", "ideal"):
        check(result_dir, selected_mode)
    print("PASS: local mappings available, TLB misses and PTW latency retained.")
    print("This is not a real host-UVM fault baseline or a measured UVM-fault speedup.")
