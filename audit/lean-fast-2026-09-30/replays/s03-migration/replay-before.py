#!/usr/bin/env python3
"""Require all nine original migration boundary failures and safe controls."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-migration-replay-") as work:
    temporary = pathlib.Path(work)
    original = temporary / "migration.go"
    original.write_bytes(subprocess.check_output(
        ["git", "show", "83faa1d:internal/translate/migration.go"], cwd=root))
    overlay = temporary / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/translate/migration.go"): str(original)}}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/translate",
        "-run", "^TestMigration(RejectsExternalTripletBoundaries|RetainsInternalAliasesAndSeparateSnapshot|BorrowedRootRetainsOwnership)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    expected = {"TestMigrationRejectsExternalTripletBoundaries/" + boundary for boundary in
                ["old-config", "old-source", "old-target", "old-state-file", "old-state-parent",
                 "current-source", "current-target", "current-state-file", "current-state-parent"]}
    failures = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    top = re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M)
    controls = set(re.findall(r"^    --- PASS: TestMigrationRetainsInternalAliasesAndSeparateSnapshot/(\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failures != expected or \
            top != ["TestMigrationRejectsExternalTripletBoundaries"] or len(controls) != 9 or \
            "external migration sentinel changed" not in result.stdout or \
            "--- PASS: TestMigrationBorrowedRootRetainsOwnership" not in result.stdout:
        raise SystemExit("Did not reproduce exact migration failures, external overwrite and safe controls")
