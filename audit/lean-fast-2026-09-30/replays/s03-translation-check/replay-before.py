#!/usr/bin/env python3
"""Replay original coverage and hash-exception readers, with no adapters."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-check-") as work:
    temp = pathlib.Path(work)
    original = temp / "check.go"
    original.write_bytes(subprocess.check_output(["git", "show", "850a2d3:internal/translate/check.go"], cwd=root))
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "internal/translate/check.go"): str(original)}}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/translate",
        "-run", "^(TestCheckExceptionsRejectExternalFiles|TestCheckCoverageRejectsExternalTarget|TestCheckRetainsInternalAliases)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    sub = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or top != {
            "TestCheckExceptionsRejectExternalFiles", "TestCheckCoverageRejectsExternalTarget"} or sub != {
            "TestCheckExceptionsRejectExternalFiles/source", "TestCheckExceptionsRejectExternalFiles/target"} or \
            "--- PASS: TestCheckRetainsInternalAliases" not in result.stdout:
        raise SystemExit("Did not reproduce exact original coverage/exception failures and safe aliases")
