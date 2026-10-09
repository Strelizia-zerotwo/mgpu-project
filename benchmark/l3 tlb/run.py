#!/usr/bin/env python3
"""Run an MGPUSim L3 experiment and keep its outputs under the project."""

import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import sys
from datetime import datetime


HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
SOURCE = ROOT / "external" / "mgpusim"
RESULTS = ROOT / "results" / "l3 tlb"
ALIASES = {
    "simpleconvolution": "sc", "matrixmultiplication": "mm",
    "matrixtranspose": "mt", "bitonicsort": "bs", "kmeans": "km",
}


def help_text():
    print('Usage: bash "benchmark/l3 tlb/run.sh" APP [simulator options]')
    print("Apps: fir, sc, mm, mt, bs, km (full sample names also work)")
    print('Examples: run.sh fir -length=81920; sc.sh -width=126 -height=126')
    print("Defaults: config.json. Use -key=value or -key value to override them.")
    print("Outputs: results/l3 tlb/APP/TIMESTAMP_INPUT/; APP/latest -> last successful run")
    print("This runner requires timing, UVM, demand-l3, verification and full reporting.")
    print("TLB default: libra-capacity (paper capacities/latencies, AMD sharing); -tlb-profile=legacy restores old L1/L2.")


def value_text(value):
    return str(value).lower() if isinstance(value, bool) else str(value)


def overrides(args, allowed):
    result = {}
    i = 0
    while i < len(args):
        arg = args[i]
        if not arg.startswith("-") or arg == "--":
            raise ValueError(f"Expected an option, got: {arg}")
        key, sep, value = arg.lstrip("-").partition("=")
        if key not in allowed:
            raise ValueError(f"Unsupported runner option: {key}; see config.json")
        if not sep:
            if i + 1 < len(args) and not args[i + 1].startswith("-"):
                i += 1
                value = args[i]
            else:
                value = "true"
        result[key] = value
        i += 1
    return result


def save_json(path, data):
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def capture(args, cwd=ROOT):
    return subprocess.check_output(args, cwd=cwd, text=True, encoding="utf-8", errors="replace")


def run_logged(command, cwd, log, env):
    print("$ " + shlex.join(command), flush=True)
    with log.open("w", encoding="utf-8") as output:
        process = subprocess.Popen(command, cwd=cwd, env=env, stdout=subprocess.PIPE,
                                   stderr=subprocess.STDOUT, text=True,
                                   encoding="utf-8", errors="replace")
        try:
            for line in process.stdout:
                print(line, end="", flush=True)
                output.write(line)
            code = process.wait()
        except BaseException:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            raise
    if code:
        raise RuntimeError(f"Command exited with {code}; see {log}")


def input_label(app, options):
    if app in ("fir", "bs"):
        label = "length-" + options["length"]
    elif app == "sc":
        label = f'{options["width"]}x{options["height"]}_mask-{options["mask-size"]}'
    elif app == "mm":
        label = "x".join(options[k] for k in ("x", "y", "z"))
    elif app == "mt":
        label = "width-" + options["width"]
    else:
        label = f'points-{options["points"]}_features-{options["features"]}'
    return re.sub(r"[^A-Za-z0-9_.-]", "_", label)[:100]


def new_run(parent, label):
    parent.mkdir(parents=True, exist_ok=True)
    stem = datetime.now().astimezone().strftime("%Y-%m-%d_%H-%M-%S") + "_" + label
    for suffix in range(10000):
        path = parent / (stem if suffix == 0 else f"{stem}_{suffix + 1:02d}")
        try:
            path.mkdir()
            return path
        except FileExistsError:
            continue
    raise RuntimeError("Too many runs with the same timestamp")


def main():
    if len(sys.argv) < 2 or any(x in ("-h", "--help", "-help") for x in sys.argv[1:]):
        help_text()
        return 0
    config = json.loads((HERE / "config.json").read_text(encoding="utf-8"))
    app = ALIASES.get(sys.argv[1], sys.argv[1])
    if app not in config["apps"]:
        raise ValueError(f"Unknown app {app}; choose fir, sc, mm, mt, bs or km")
    app_config = config["apps"][app]
    options = {k: value_text(v) for k, v in config["common"].items()}
    options.update({k: value_text(v) for k, v in app_config["options"].items()})
    extra = {"vm-local-walk-cycles", "vm-local-walkers", "vm-transaction-slots",
             "vm-max-waiters", "vm-host-walk-cycles", "vm-host-walkers", "vm-host-queue",
             "vm-link-cycles", "vm-install-cycles", "l3-tlb-width", "l3-tlb-mshrs",
             "l3-tlb-max-waiters", "l3-tlb-max-inflight"}
    options.update(overrides(sys.argv[2:], set(options) | extra))
    for key, expected in {"vm-mode": "demand-l3", "timing": "true",
                          "use-unified-memory": "true", "verify": "true",
                          "report-all": "true", "disable-rtm": "true"}.items():
        if options.get(key) != expected:
            raise ValueError(f"This L3 experiment runner requires -{key}={expected}")
    gpus = [int(x) for x in options["gpus"].split(",")]
    if not gpus or min(gpus) < 1 or len(set(gpus)) != len(gpus):
        raise ValueError("gpus must be unique positive GPU IDs")
    profile = options.get("tlb-profile", "legacy")
    if profile not in ("legacy", "libra-capacity"):
        raise ValueError("tlb-profile must be legacy or libra-capacity")
    gpu_list = ",".join(map(str, gpus))
    options["gpus"] = gpu_list
    go = shutil.which("go")
    if not go and Path("/usr/local/go/bin/go").is_file():
        go = "/usr/local/go/bin/go"
    if not go:
        raise RuntimeError("Go not found. Add /usr/local/go/bin to PATH.")
    report = SOURCE / "experiments/libra-baseline-guide/report_tlb.py"
    checker = SOURCE / "experiments/l3-tlb/check_metrics.py"
    for path in (SOURCE / "go.mod", report, checker, SOURCE / "amd/timing/sectortlb"):
        if not path.exists():
            raise RuntimeError(f"Required source/tool missing: {path}")
    parent = RESULTS / app
    run = new_run(parent, input_label(app, options) + "_" + profile)
    print(f"Result directory: {run}", flush=True)
    binary = run / app_config["sample"]
    command = [str(binary)] + [f"-{key}={value}" for key, value in options.items()]
    command.append(f"-metric-file-name={run / 'metrics'}")
    metadata = {"app": app, "sample": app_config["sample"], "options": options,
                "source": str(SOURCE), "command": command, "cwd": str(run),
                "started": datetime.now().astimezone().isoformat(), "status": "running"}
    save_json(run / "run.json", metadata)
    (run / "status.txt").write_text("RUNNING\n")
    env = os.environ.copy()
    env["PATH"] = str(Path(go).parent) + os.pathsep + env.get("PATH", "")
    try:
        (run / "commit.txt").write_text(capture(["git", "rev-parse", "HEAD"]))
        (run / "git-status.txt").write_text(capture(["git", "status", "--short"]))
        (run / "working-tree.diff").write_text(capture(["git", "diff", "--no-ext-diff"]))
        (run / "staged.diff").write_text(capture(["git", "diff", "--cached", "--no-ext-diff"]))
        (run / "go-version.txt").write_text(capture([go, "version"]))
        shutil.copyfile(HERE / "config.json", run / "config.json")
        (run / "command.sh").write_text("#!/usr/bin/env bash\n# Recorded invocation; use the runner for a NEW run.\n"
                                        + "cd " + shlex.quote(str(run)) + "\n"
                                        + shlex.join(command) + "\n")
        run_logged([go, "build", "-p", "2", "-o", str(binary),
                    "./amd/samples/" + app_config["sample"]], SOURCE, run / "build.log", env)
        with binary.open("rb") as binary_file:
            metadata["binary_sha256"] = hashlib.file_digest(binary_file, "sha256").hexdigest()
        run_logged(command, run, run / "run.log", env)
        for level in ("L1", "L2", "L3"):
            run_logged([sys.executable, str(report), str(run / "metrics.sqlite3"),
                        "--level", level, "--gpus", gpu_list], run,
                       run / f"{level.lower()}-hit-rate.txt", env)
        run_logged([sys.executable, str(checker), str(run / "metrics.sqlite3"),
                    "--gpus", gpu_list], run, run / "check.log", env)
        run_logged([sys.executable, str(HERE / "check_tlb_profile.py"),
                    str(run / "metrics.sqlite3"), "--profile", profile, "--gpus", gpu_list],
                   run, run / "tlb-config-check.txt", env)
        latest = parent / "latest"
        if latest.exists() and not latest.is_symlink():
            raise RuntimeError(f"Refusing to replace non-symlink: {latest}")
        temp_link = parent / f".latest-{os.getpid()}"
        temp_link.symlink_to(run.name, target_is_directory=True)
        temp_link.replace(latest)
        metadata["status"] = "success"
        (run / "status.txt").write_text("SUCCESS\n")
        print(f"Completed. L3 report: {latest / 'l3-hit-rate.txt'}")
    except (Exception, KeyboardInterrupt) as exc:
        metadata["status"] = "interrupted" if isinstance(exc, KeyboardInterrupt) else "failed"
        metadata["error"] = str(exc)
        (run / "status.txt").write_text(metadata["status"].upper() + "\n")
        print(f"Run {metadata['status']}: {exc}\nSaved logs: {run}", file=sys.stderr)
        return 130 if isinstance(exc, KeyboardInterrupt) else 1
    finally:
        metadata["finished"] = datetime.now().astimezone().isoformat()
        save_json(run / "run.json", metadata)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, RuntimeError, OSError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(2)
