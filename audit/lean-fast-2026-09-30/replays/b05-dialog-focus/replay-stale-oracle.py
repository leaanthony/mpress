#!/usr/bin/env python3
"""Replay the invalid opener expectation: 12 script failures, 48 native passes."""
import json
import os
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
tools = root / "audit/lean-fast-2026-09-30/tools"
with tempfile.TemporaryDirectory(prefix="mpress-focus-oracle-") as work:
    temp = pathlib.Path(work)
    binary = temp / "mpress"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/mpress"], cwd=root, check=True)
    source = tools / "dialog_focus_test.go"
    text = source.read_text()
    stale = text.replace('\t\tif activation == "script" {\n\t\t\texpected = "window.auditDialogPriorFocus"\n\t\t}\n', '')
    if stale == text:
        raise SystemExit("Expected oracle branch was not found")
    original = temp / "stale_test.go"
    original.write_text(stale)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(original)}}))
    result = subprocess.run([
        "go", "test", "-overlay=" + str(overlay), "-run", "^TestDialogOpenerFocus$", "-count=1", "-v"
    ], cwd=tools, env=dict(os.environ, MPRESS_DIALOG_BIN=str(binary)),
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failures = set(re.findall(r"^    --- FAIL: TestDialogOpenerFocus/(\S+)", result.stdout, re.M))
    expected = {f"{dialog}/script/{close}" for dialog in ["image-1", "image-2", "contribution"]
                for close in ["escape", "button", "backdrop", "script"]}
    passes = re.findall(r"^    --- PASS: TestDialogOpenerFocus/(\S+)", result.stdout, re.M)
    if result.returncode == 0 or "[build failed]" in result.stdout or failures != expected or len(passes) != 48:
        raise SystemExit("Did not reproduce exactly 12 stale script-oracle failures and 48 native controls")
