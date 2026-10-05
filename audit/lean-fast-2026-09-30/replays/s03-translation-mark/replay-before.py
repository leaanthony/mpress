#!/usr/bin/env python3
"""Replay the original authenticated translation review outside the project."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-translation-review-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/dev/server.go", "internal/content/content.go",
                 "internal/translate/engine.go", "internal/translate/state.go",
                 "internal/translate/audit.go", "internal/translate/refine.go",
                 "internal/translate/comparison.go", "internal/dev/translation_models.go",
                 "internal/translate/check.go", "internal/translate/conversion.go",
                 "internal/projectconvert/convert.go", "internal/translate/migration.go"]:
        original = temp / name.replace("/", "-")
        original.write_bytes(subprocess.check_output(["git", "show", "22dddd1:" + name], cwd=root))
        replacements[str(root / name)] = str(original)
    # Future worker-root tests reference RunRoot, absent at this review baseline.
    future = temp / "future-plan-tests.go"
    future.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_plan_confinement_test.go")] = str(future)
    future_input = temp / "future-input-tests.go"
    future_input.write_text("package translate\n")
    replacements[str(root / "internal/translate/root_inputs_test.go")] = str(future_input)
    future_audit = temp / "future-audit-tests.go"
    future_audit.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_audit_confinement_test.go")] = str(future_audit)
    future_comparison = temp / "future-comparison-tests.go"
    future_comparison.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_comparison_confinement_test.go")] = str(future_comparison)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev",
        "-run", "^TestAuthoringTranslationReview", "-count=1", "-v"], cwd=root,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failures = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    expected = {"TestAuthoringTranslationReviewRejectsExternalFiles/" + v
                for v in ["source", "target", "state-file", "state-parent", "state-discovery"]}
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failures != expected or top != {
            "TestAuthoringTranslationReviewRejectsExternalFiles"} or not all(
            "--- PASS: " + v in result.stdout for v in ["TestAuthoringTranslationReviewRetainsInternalStateAliases",
            "TestAuthoringTranslationReviewRetainsInternalFileAliases"]) or "sentinel index.json changed" not in result.stdout:
        raise SystemExit("Did not reproduce all five original boundaries, external write and safe alias controls")
