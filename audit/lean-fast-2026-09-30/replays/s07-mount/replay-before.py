#!/usr/bin/env python3
"""Replay mounting/reader defects against 5bdc828 production on native Linux."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-mount-replay-") as work:
    temporary = pathlib.Path(work)
    original = temporary / "version.go"
    original.write_bytes(subprocess.check_output([
        "git", "show", "5bdc828:internal/version/version.go"
    ], cwd=root))
    overlay = temporary / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/version/version.go"): str(original)
    }}))
    pattern = "^Test(MountRejectsExternalDestinations|MountPreservesProjectInputs|MountPreservesInputsBehindNestedLinks|MountRetainsNavigationAssetsAndSafeAliases|VersionReadersWaitForReplacement)$"
    result = subprocess.run([
        "go", "test", "-overlay=" + str(overlay), "./internal/version",
        "-run", pattern, "-count=1"
    ], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failures = re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M)
    if result.returncode == 0 or "[build failed]" in result.stdout or len(failures) != 5:
        raise SystemExit("Replay did not reproduce the expected behavior failures")
