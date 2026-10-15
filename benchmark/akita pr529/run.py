#!/usr/bin/env python3
"""Isolated four-GPU validation of PR #529's admission semantics on beta.10."""
import argparse
from datetime import datetime
import fcntl
import json
from pathlib import Path
import re
import sqlite3
import subprocess

ROOT = Path("/home/only/projects/akita-529-validation-20261013")
SOURCE = ROOT / "mgpusim"
BUILDER = SOURCE / "amd/samples/runner/timingconfig/shaderarray/builder.go"
GO = "/usr/local/go/bin/go"
VARIANTS = {"original-2": (2, False), "fixed-2": (2, True), "fixed-1": (1, True)}


def run_command(command, cwd, log):
    print("$", " ".join(map(str, command)), flush=True)
    with log.open("w") as out:
        result = subprocess.run(command, cwd=cwd, stdout=out, stderr=subprocess.STDOUT)
    if result.returncode:
        raise RuntimeError(f"command failed ({result.returncode}); see {log}")


def check_metrics(directory, stages):
    databases = list(directory.glob("*.sqlite3"))
    if len(databases) != 1:
        raise RuntimeError(f"expected one SQLite database, found {databases}")
    result = {}
    with sqlite3.connect(databases[0]) as db:
        rows = db.execute("SELECT Name, Spec FROM component_spec WHERE Name LIKE '%L1VTLB%'").fetchall()
        for gpu in range(1, 5):
            prefix = f"GPU[{gpu}]."
            specs = [json.loads(spec) for name, spec in rows if name.startswith(prefix)]
            if len(specs) != 64 or any(spec["latency"] != stages for spec in specs):
                raise RuntimeError(f"GPU {gpu}: actual vector L1 specifications do not match {stages} stages")
            kernel = db.execute("SELECT Value FROM mgpusim_metrics WHERE Location=? AND What='kernel_time'",
                                (prefix + "CommandProcessor",)).fetchone()
            requests = db.execute("SELECT SUM(Value) FROM mgpusim_metrics WHERE Location LIKE ? "
                                  "AND What IN ('hit','miss','mshr-hit')",
                                  (prefix + "%L1VTLB%",)).fetchone()[0]
            if not kernel or float(kernel[0]) <= 0 or requests is None or float(requests) <= 0:
                raise RuntimeError(f"GPU {gpu} did not report active kernel/vector translation work")
            result[str(gpu)] = {"vector_l1_components": len(specs), "latency_stages": stages,
                                "kernel_seconds": kernel[0], "vector_l1_requests": requests}
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app", choices=["sc", "fir", "all"], default="sc")
    parser.add_argument("--variant", choices=[*VARIANTS, "all"], default="fixed-1")
    parser.add_argument("--width", type=int, default=62)
    parser.add_argument("--height", type=int, default=62)
    parser.add_argument("--mask-size", type=int, default=3)
    parser.add_argument("--length", type=int, default=4096)
    args = parser.parse_args()
    if any(value <= 0 for value in [args.width, args.height, args.mask_size, args.length]):
        parser.error("sizes must be positive")
    variants = list(VARIANTS) if args.variant == "all" else [args.variant]
    apps = ["sc", "fir"] if args.app == "all" else [args.app]
    lock = (ROOT / ".runner.lock").open("w")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError as exc:
        raise RuntimeError("another validation run is editing the isolated fixture; wait for it to finish") from exc
    canonical = BUILDER.read_text()
    pattern = r"b\.buildTLB\(name, 4, 64, 64, 32, [12]\)"
    if len(re.findall(pattern, canonical)) != 1:
        raise RuntimeError("unexpected vector L1 source; refusing to edit")
    stamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S_%f")
    builds = ROOT / "builds" / stamp
    builds.mkdir(parents=True)
    summary = []
    try:
        for variant in variants:
            stages, patched = VARIANTS[variant]
            BUILDER.write_text(re.sub(pattern, f"b.buildTLB(name, 4, 64, 64, 32, {stages})", canonical))
            flags = [] if patched else [f"-modfile={ROOT / 'reference.mod'}"]
            module = subprocess.check_output([GO, "list", *flags, "-m", "-json",
                                               "github.com/sarchlab/akita/v5"], cwd=SOURCE, text=True)
            for app in apps:
                size = f"{args.width}x{args.height}_mask-{args.mask_size}" if app == "sc" else f"length-{args.length}"
                directory = Path("/home/only/projects/mgpu-project/results/akita pr529") / app / f"{stamp}_{size}" / variant
                directory.mkdir(parents=True)
                binary = builds / f"{app}-{variant}"
                package = "simpleconvolution" if app == "sc" else "fir"
                run_command([GO, "build", *flags, "-p", "2", "-o", str(binary),
                             f"./amd/samples/{package}"], SOURCE, directory / "build.log")
                options = [f"-width={args.width}", f"-height={args.height}", f"-mask-size={args.mask_size}"] if app == "sc" else [f"-length={args.length}"]
                command = [str(binary), "-timing", "-verify", "-disable-rtm", "-gpus=1,2,3,4",
                           "-use-unified-memory", "-report-all", *options]
                (directory / "akita-module.json").write_text(module)
                (directory / "command.json").write_text(json.dumps(command, indent=2))
                (directory / "shaderarray-builder.go.txt").write_text(BUILDER.read_text())
                (directory / "backport.patch").write_bytes((ROOT / "backport.patch").read_bytes())
                run_command(command, directory, directory / "run.log")
                if "Passed!" not in (directory / "run.log").read_text():
                    raise RuntimeError(f"result verification did not pass: {directory}")
                metrics = check_metrics(directory, stages)
                record = {"app": app, "variant": variant, "output": str(directory), "gpus": metrics,
                          "scope": "beta.10 TLB admission backport; not a full MGPUSim run against PR #529"}
                (directory / "validation.json").write_text(json.dumps(record, indent=2))
                summary.append(record)
                print(f"PASS {app} {variant}: all four GPUs active, vector L1 depth={stages}; {directory}", flush=True)
    finally:
        BUILDER.write_text(canonical)
    (builds / "summary.json").write_text(json.dumps(summary, indent=2))
    print(f"Summary: {builds / 'summary.json'}", flush=True)


if __name__ == "__main__":
    main()
