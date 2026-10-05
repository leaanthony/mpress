#!/usr/bin/env python3
"""Replay original authoring conversion and state planning through HTTP."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-conversion-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/dev/server.go", "internal/projectconvert/convert.go", "internal/translate/conversion.go"]:
        original = temp / name.replace("/", "-")
        original.write_bytes(subprocess.check_output(["git", "show", "e8eac75:" + name], cwd=root))
        replacements[str(root / name)] = str(original)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev",
        "-run", "^(TestAuthoringConversionRejectsExternalInputs|TestAuthoringConversionPreservesTemporarySentinels|TestAuthoringConversionRetainsInternalAliases)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    cases = {
        "TestAuthoringConversionRejectsExternalInputs": ["source", "state-file", "state-parent"],
        "TestAuthoringConversionPreservesTemporarySentinels": ["owned-file", "external-link"],
    }
    expected = {test + "/" + case for test, entries in cases.items() for case in entries}
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or top != set(cases) or \
            "sentinel index.json changed" not in result.stdout or "sentinel must-survive changed" not in result.stdout or \
            "--- PASS: TestAuthoringConversionRetainsInternalAliases" not in result.stdout:
        raise SystemExit("Did not reproduce exact conversion failures, external overwrites and alias controls")
