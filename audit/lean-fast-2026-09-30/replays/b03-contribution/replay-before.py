#!/usr/bin/env python3
"""Build original production CSS and reproduce six contribution dialog failures."""
import io
import os
import json
import pathlib
import re
import subprocess
import tarfile
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-contribution-replay-") as work:
    temp = pathlib.Path(work)
    project = temp / "project"
    project.mkdir()
    archive = subprocess.check_output([
        "git", "archive", "7d8ce2f", "mpress.yaml", "docs", "static",
        "docs-theme.css", "CONTRIBUTING.md"
    ], cwd=root)
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(project, filter="data")
    original = temp / "theme_default.go"
    original.write_bytes(subprocess.check_output([
        "git", "show", "7d8ce2f:internal/site/theme_default.go"
    ], cwd=root))
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/site/theme_default.go"): str(original)
    }}))
    binary = temp / "mpress"
    subprocess.run(["go", "build", "-overlay=" + str(overlay), "-o", str(binary), "./cmd/mpress"], cwd=root, check=True)
    subprocess.run([str(binary), "build", "--strict"], cwd=project, check=True, stdout=subprocess.PIPE)
    env = dict(os.environ, MPRESS_CONTRIBUTE_SITE=str(project / "site"))
    env.pop("MPRESS_CONTRIBUTE_REPORT", None)
    result = subprocess.run([
        "go", "test", "-run", "^TestContributionDialogPhoneReachability$", "-count=1", "-v"
    ], cwd=root / "audit/lean-fast-2026-09-30/tools", env=env,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failures = set(re.findall(r"^    --- FAIL: TestContributionDialogPhoneReachability/(\S+)", result.stdout, re.M))
    expected = {"320x568", "375x667", "375x375", "667x375", "320x320", "1024x375"}
    if result.returncode == 0 or "[build failed]" in result.stdout or failures != expected:
        raise SystemExit("Replay did not reproduce the expected six contribution behavior failures")
