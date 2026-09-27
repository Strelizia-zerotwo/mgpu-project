"""Check completion/conservation of a demand-l3 run; optionally compare old modes."""

import argparse
import math
from pathlib import Path
import sqlite3


def read(path):
    path = Path(path)
    if not path.is_file():
        raise ValueError(f"Missing database: {path}")
    with sqlite3.connect(path.resolve().as_uri() + "?mode=ro", uri=True) as conn:
        rows = conn.execute("SELECT Location, What, Value, Unit FROM mgpusim_metrics").fetchall()
    result = {}
    for loc, what, value, unit in rows:
        key = (loc, what, unit)
        if key in result:
            raise ValueError(f"Duplicate metric: {key}")
        result[key] = value
    return result


def require(condition, message):
    if not condition:
        raise ValueError(message)


def check(path, gpus):
    data = read(path)
    for gpu in gpus:
        loc = f"GPU[{gpu}].L3TLB"

        def count(name):
            value = data[(loc, name, "count")]
            require(value is not None and math.isfinite(value) and value >= 0
                    and float(value).is_integer(), f"Invalid count: {loc}/{name}")
            return int(value)

        requests = count("l3_requests")
        require(requests > 0, f"No L3 traffic for GPU {gpu}")
        require(requests == count("hit") + count("miss") + count("mshr-hit"),
                f"Unclassified requests on GPU {gpu}")
        require(requests == count("l3_completed"), f"Incomplete requests on GPU {gpu}")
        require(count("miss") == count("l3_lower_responses") == count("l3_fills")
                == data[(f"GPU[{gpu}].MMU", "vm_requests", "count")],
                f"L3 to GMMU conservation failed for GPU {gpu}")
        for name in ["l3_outstanding_end", "l3_mshr_end", "l3_reset_dropped", "l3_stale_responses"]:
            require(count(name) == 0, f"Unexpected {loc}/{name}")
        require(count("l3_outstanding_peak") <= count("l3_max_inflight")
                and count("l3_mshr_peak") <= count("l3_mshrs"),
                f"Capacity exceeded for GPU {gpu}")
        print(f"GPU {gpu}: {requests} requests, {count('hit')} hits, {count('miss')} misses, "
              f"{count('mshr-hit')} merges; completion and capacity checks passed")


def compare(before, after):
    left, right = read(before), read(after)
    require(left.keys() == right.keys(), "Old-mode metric keys changed")
    for key, value in left.items():
        other = right[key]
        if value is None or other is None:
            require(value is other, f"NULL changed: {key}")
        elif key[2] == "count":
            require(value == other, f"Counter changed: {key}: {value} != {other}")
        else:
            require(math.isclose(value, other, rel_tol=1e-10,
                                 abs_tol=1e-18 if key[2] == "second" else 1e-10),
                    f"Metric changed: {key}: {value} != {other}")
    print(f"Old-mode comparison passed: {len(left)} metrics: {before} vs {after}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("database", nargs="?")
    parser.add_argument("--gpus", default="1,2,3,4")
    parser.add_argument("--compare", nargs=2, metavar=("BEFORE", "AFTER"))
    args = parser.parse_args()
    if args.compare:
        compare(*args.compare)
    elif args.database:
        gpus = [int(value) for value in args.gpus.split(",")]
        require(len(gpus) == len(set(gpus)) and all(gpu > 0 for gpu in gpus), "Invalid GPU IDs")
        check(args.database, gpus)
    else:
        parser.error("Supply a database or --compare BEFORE AFTER")


if __name__ == "__main__":
    main()
