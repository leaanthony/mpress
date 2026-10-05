#!/usr/bin/env python3
"""Build original production CSS and reproduce eight constrained viewport failures."""
import io
import os
import json
import pathlib
import re
import subprocess
import tarfile
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-a11y-replay-") as work:
    temp = pathlib.Path(work)
    project = temp / "project"
    project.mkdir()
    archive = subprocess.check_output([
        "git", "archive", "902cd7a", "mpress.yaml", "docs", "static",
        "docs-theme.css", "CONTRIBUTING.md"
    ], cwd=root)
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(project, filter="data")
    original = temp / "accessibility.go"
    original.write_bytes(subprocess.check_output([
        "git", "show", "902cd7a:internal/site/accessibility.go"
    ], cwd=root))
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/site/accessibility.go"): str(original)
    }}))
    binary = temp / "mpress"
    subprocess.run(["go", "build", "-overlay=" + str(overlay), "-o", str(binary), "./cmd/mpress"], cwd=root, check=True)
    subprocess.run([str(binary), "build", "--strict"], cwd=project, check=True, stdout=subprocess.PIPE)
    env = dict(os.environ, MPRESS_A11Y_SITE=str(project / "site"))
    env.pop("MPRESS_A11Y_REPORT", None)
    result = subprocess.run([
        "go", "test", "-run", "^TestAccessibilityViewportReachability$", "-count=1", "-v"
    ], cwd=root / "audit/lean-fast-2026-09-30/tools", env=env,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failures = set(re.findall(r"^    --- FAIL: TestAccessibilityViewportReachability/(\S+)", result.stdout, re.M))
    expected = {
        "375x667-text1-light", "667x375-text1-dark", "375x375-text1-light",
        "320x320-text2-dark", "667x375-text2-light", "1024x375-text2-dark",
        "375x667-text2-light", "761x601-text2-light",
    }
    controls_pass = all("--- PASS: TestAccessibilityViewportReachability/" + name in result.stdout
                        for name in ["1280x900-text1-dark", "1280x900-text2-dark"])
    if result.returncode == 0 or "[build failed]" in result.stdout or failures != expected or not controls_pass:
        raise SystemExit("Replay did not reproduce the expected eight behavior failures and two desktop controls")
