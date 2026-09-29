#!/usr/bin/env python3
"""Run SC against the separately pinned, unmodified upstream MGPUSim."""
from datetime import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shlex
import shutil
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
spec = importlib.util.spec_from_file_location("l3_run_helpers", ROOT / "benchmark/l3 tlb/run.py")
helpers = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helpers)


def main():
    args = sys.argv[1:]
    fixed = "--fixed" in args
    args = [x for x in args if x != "--fixed"]
    if any(x in ("-h", "--help", "-help") for x in args):
        print('Usage: bash "benchmark/original/sc.sh" [-width=62 -height=62 -gpus=1,2,3,4]')
        print("Default: pinned official source, without vm-mode or L3 extensions.")
        print("--fixed: current modified checkout with vm-mode=shared; results/original/sc-fixed/.")
        return 0
    source = ROOT / ("external/mgpusim" if fixed else "external/mgpusim-original")
    provenance = json.loads((HERE / "source.json").read_text())
    options = {"timing": "true", "gpu": "r9nano", "arch": "gcn3", "gpus": "1,2,3,4",
               "use-unified-memory": "true", "verify": "true", "report-all": "true",
               "disable-rtm": "true", "width": "62", "height": "62", "mask-size": "3"}
    options.update(helpers.overrides(args, set(options)))
    if fixed:
        options["vm-mode"] = "shared"
    go = shutil.which("go") or "/usr/local/go/bin/go"
    env = os.environ.copy()
    env["PATH"] = str(Path(go).parent) + os.pathsep + env.get("PATH", "")
    run = helpers.new_run(ROOT / "results/original" / ("sc-fixed" if fixed else "sc"), helpers.input_label("sc", options))
    binary = run / "simpleconvolution"
    command = [str(binary)] + [f"-{k}={v}" for k, v in options.items()]
    command.append(f"-metric-file-name={run / 'metrics'}")
    metadata = {"source": str(source), "upstream_reference": provenance, "options": options,
                "variant": "modified-shared" if fixed else "unmodified-upstream",
                "command": command, "started": datetime.now().astimezone().isoformat(),
                "status": "running", "cwd": str(run)}
    helpers.save_json(run / "run.json", metadata)
    (run / "status.txt").write_text("RUNNING\n")
    print("Result directory:", run, flush=True)
    try:
        (run / "command.sh").write_text("#!/usr/bin/env bash\n# Historical command; use sc.sh for a new run.\ncd "
                                        + shlex.quote(str(run)) + "\n" + shlex.join(command) + "\n")
        (run / "go-version.txt").write_text(helpers.capture([go, "version"]))
        helpers.run_logged([go, "build", "-p", "2", "-o", str(binary),
                            "./amd/samples/simpleconvolution"], source, run / "build.log", env)
        with binary.open("rb") as binary_file:
            metadata["binary_sha256"] = hashlib.file_digest(binary_file, "sha256").hexdigest()
        if fixed:
            for name, git_args in {
                "commit.txt": ["rev-parse", "HEAD"],
                "git-status.txt": ["status", "--short"],
                "working-tree.diff": ["diff", "--no-ext-diff"],
                "staged.diff": ["diff", "--cached", "--no-ext-diff"],
            }.items():
                (run / name).write_text(helpers.capture(["git"] + git_args))
        helpers.run_logged(command, run, run / "run.log", env)
        if options["timing"] == "true" and options["report-all"] == "true":
            report = ROOT / "external/mgpusim/experiments/libra-baseline-guide/report_tlb.py"
            helpers.run_logged([sys.executable, str(report), str(run / "metrics.sqlite3"),
                                "--level", "L2", "--gpus", options["gpus"]], run,
                               run / "l2-hit-rate.txt", env)
        metadata["status"] = "success"
        (run / "status.txt").write_text("SUCCESS\n")
        latest = run.parent / "latest"
        if latest.exists() and not latest.is_symlink():
            raise RuntimeError(f"Refusing to replace {latest}")
        temp = run.parent / f".latest-{os.getpid()}"
        temp.symlink_to(run.name, target_is_directory=True)
        temp.replace(latest)
    except (Exception, KeyboardInterrupt) as exc:
        metadata["status"] = "interrupted" if isinstance(exc, KeyboardInterrupt) else "failed"
        metadata["error"] = str(exc)
        (run / "status.txt").write_text(metadata["status"].upper() + "\n")
        print(f"Run {metadata['status']}: {exc}\nSaved logs: {run}", file=sys.stderr)
        return 1
    finally:
        metadata["finished"] = datetime.now().astimezone().isoformat()
        helpers.save_json(run / "run.json", metadata)
    return 0


if __name__ == "__main__":
    sys.exit(main())
