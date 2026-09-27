"""Read per-GPU TLB outcomes without confusing absent counters with zero misses."""
import argparse
import math
from pathlib import Path
import re
import sqlite3
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("database", type=Path)
    parser.add_argument("--level", choices=("L1", "L2", "L3"), default="L3")
    parser.add_argument("--gpus", default="1,2,3,4", help="Expected GPU IDs, e.g. 1,2,3,4")
    args = parser.parse_args()
    expected = [int(value) for value in args.gpus.split(",")]
    if not expected or len(set(expected)) != len(expected) or any(gpu < 1 for gpu in expected):
        parser.error("--gpus must contain unique positive GPU IDs")
    if not args.database.is_file():
        parser.error(f"Database does not exist: {args.database}")
    pattern = re.compile(r"\." + args.level + r"[VIS]?TLB(?:[.\[]|$)")
    with sqlite3.connect(args.database.resolve().as_uri() + "?mode=ro", uri=True) as conn:
        rows = conn.execute("SELECT Location, What, Value, Unit FROM mgpusim_metrics WHERE What IN ('hit','miss','mshr-hit')").fetchall()
    components = {}
    for location, what, value, unit in rows:
        gpu_match = re.match(r"^GPU\[(\d+)\]\.", location)
        if not gpu_match or not pattern.search(location):
            continue
        gpu = int(gpu_match.group(1))
        if gpu not in expected:
            continue
        if value is None or not math.isfinite(value) or value < 0 or not float(value).is_integer() or unit != "count":
            raise ValueError(f"Invalid count: {location}/{what}={value}, unit={unit}")
        values = components.setdefault((gpu, location), {})
        if what in values:
            raise ValueError(f"Duplicate metric: {location}/{what}")
        values[what] = int(value)
    missing = [gpu for gpu in expected if not any(key[0] == gpu for key in components)]
    if missing:
        raise ValueError(f"Missing {args.level} TLB counters for GPU IDs {missing}. The component may be absent, untraced, or unused; a hit rate cannot be inferred.")
    for key, values in components.items():
        if values.keys() != {"hit", "miss", "mshr-hit"}:
            raise ValueError(f"Incomplete counters for {key}: {sorted(values)}")
    print("GPU\tLevel\tComponents\tHit\tMiss\tMSHR-hit\tTotal\tResident-hit-rate")
    for gpu in expected:
        selected = [values for (gid, _), values in components.items() if gid == gpu]
        if args.level == "L3" and len(selected) != 1:
            raise ValueError(f"GPU {gpu}: expected one shared L3 TLB, found {len(selected)}")
        hit, miss, coalesced = (sum(values[key] for values in selected) for key in ("hit", "miss", "mshr-hit"))
        total = hit + miss + coalesced
        rate = f"{100*hit/total:.6f}%" if total else "N/A"
        print(f"{gpu}\t{args.level}\t{len(selected)}\t{hit}\t{miss}\t{coalesced}\t{total}\t{rate}")
    print("Rate = hit / (hit + miss + mshr-hit); MSHR merging is not a resident mapping hit.")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, sqlite3.Error) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
