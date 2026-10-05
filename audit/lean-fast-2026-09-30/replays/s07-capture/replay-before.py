#!/usr/bin/env python3
"""Replay snapshot capture defects against d0edb5f production, rejecting compile failures."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-capture-replay-") as work:
    temporary = pathlib.Path(work)
    original = temporary / "version.go"
    original.write_bytes(subprocess.check_output([
        "git", "show", "d0edb5f:internal/version/version.go"
    ], cwd=root))
    overlay = temporary / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/version/version.go"): str(original)
    }}))
    cases = [
        ("./internal/version", "^TestCapture(PreservesTmpNamedSnapshot|RejectsExternalPaths|IndexesNestedManifestNamedAsset)$", 3),
        ("./internal/dev", "^TestAuthoringMCPCapturePreservesExternalStore$", 1),
    ]
    for package, pattern, expected in cases:
        result = subprocess.run([
            "go", "test", "-overlay=" + str(overlay), package,
            "-run", pattern, "-count=1"
        ], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        print(result.stdout, end="", flush=True)
        failures = re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M)
        if result.returncode == 0 or "[build failed]" in result.stdout or len(failures) != expected:
            raise SystemExit("Replay did not reproduce the expected behavior failures")
