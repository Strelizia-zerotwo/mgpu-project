"""Check saved TLB geometry, wiring and traffic for the capacity preset."""
import argparse
import json
from pathlib import Path
import re
import sqlite3


def require(ok, message):
    if not ok:
        raise ValueError(message)


def check(database, profile, gpus):
    require(database.is_file(), f"Missing database: {database}")
    with sqlite3.connect(database.resolve().as_uri() + "?mode=ro", uri=True) as conn:
        specs = {name: json.loads(spec) for name, spec in conn.execute("SELECT Name, Spec FROM component_spec")}
        rows = conn.execute("SELECT Location, What, Value FROM mgpusim_metrics").fetchall()
    metrics = {}
    for loc, what, value in rows:
        require((loc, what) not in metrics, f"Duplicate metric {loc}/{what}")
        metrics[loc, what] = value
    if profile == "legacy":
        for gpu in gpus:
            spec = specs[f"GPU[{gpu}].L2TLB"]
            require("num_sets" in spec, "legacy profile unexpectedly built a sector L2")
            print(f"GPU {gpu}: legacy L2 {spec['num_sets'] * spec['num_ways']} page entries, {spec['latency']} cycles")
        return
    print("Profile: libra-capacity. Capacities/timing per instance; AMD sharing, not NVIDIA TPC/GPC.")
    for gpu in gpus:
        prefix = f"GPU[{gpu}]."
        l1s = [name for name in specs if name.startswith(prefix) and re.search(r"\.L1[VIS]?TLB(?:\[\d+\])?$", name)]
        require(bool(l1s), f"Missing L1 TLBs for GPU {gpu}")
        levels = [(name, (16, 16, 1, 1)) for name in l1s]
        levels.append((prefix + "L2TLB", (128, 8, 16, 10)))
        # L3 flags remain adjustable; read the effective values, do not assume
        # overrides still match the paper defaults.
        l3name = prefix + "L3TLB"
        require(l3name in specs, f"Missing {l3name}")
        for name, expected in levels:
            spec = specs[name]
            actual = tuple(spec[k] for k in ("Entries", "Ways", "Subentries", "LookupCycles"))
            require(actual == expected, f"Wrong {name}: {actual}, expected {expected}")
            require(spec["Log2PageSize"] == 12, f"Unexpected page size in {name}")
            lower = prefix + ("L2TLB.Top" if name in l1s else "L3TLB.Top")
            require(spec["LowerPort"] == lower, f"Wrong next level for {name}")
        l2name = prefix + "L2TLB"
        require(sum(metrics[name, "miss"] for name in l1s) == metrics[l2name, "l2_requests"],
                f"L1 to L2 traffic conservation failed on GPU {gpu}")
        require(metrics[l2name, "miss"] == metrics[l3name, "l3_requests"],
                f"L2 to L3 traffic conservation failed on GPU {gpu}")
        for name in l1s + [l2name]:
            level = "l1_" if name in l1s else "l2_"
            requests = metrics[name, level + "requests"]
            require(requests == metrics[name, level + "completed"], f"Incomplete {name}")
            require(requests == sum(metrics[name, k] for k in ("hit", "miss", "mshr-hit")), f"Unclassified {name}")
            require(metrics[name, "miss"] == metrics[name, level + "fills"] == metrics[name, level + "lower_responses"],
                    f"Missing refill at {name}")
            for field in ("outstanding_end", "mshr_end", "reset_dropped", "stale_responses"):
                require(metrics[name, level + field] == 0, f"Unexpected {name}/{field}")
            require(metrics[name, level + "outstanding_peak"] <= metrics[name, level + "max_inflight"], f"Capacity: {name}")
            require(metrics[name, level + "mshr_peak"] <= metrics[name, level + "mshrs"], f"MSHR capacity: {name}")
        l3 = specs[l3name]
        print(f"GPU {gpu}: {len(l1s)} L1s: 16 entries/16 ways/1 cycle; "
              f"one L2: 128 tags x 16 subentries/8 ways/10 cycles; "
              f"one L3: {l3['Entries']} tags x {l3['Subentries']} subentries/{l3['Ways']} ways/{l3['LookupCycles']} cycles")
        print(f"GPU {gpu}: saved geometry, wiring, completion and inter-level traffic checks passed")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("database", type=Path)
    parser.add_argument("--profile", choices=("legacy", "libra-capacity"), required=True)
    parser.add_argument("--gpus", default="1,2,3,4")
    args = parser.parse_args()
    gpus = [int(x) for x in args.gpus.split(",")]
    require(gpus and len(gpus) == len(set(gpus)) and min(gpus) > 0, "Invalid GPU IDs")
    check(args.database, args.profile, gpus)


if __name__ == "__main__":
    main()
