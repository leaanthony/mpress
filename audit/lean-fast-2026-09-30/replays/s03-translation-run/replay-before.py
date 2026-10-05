#!/usr/bin/env python3
"""Replay original planning/execution IO; the borrowed-root seam forwards only."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-translation-run-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/dev/server.go", "internal/translate/engine.go", "internal/translate/state.go", "internal/translate/audit.go", "internal/translate/refine.go",
                 "internal/translate/comparison.go", "internal/dev/translation_models.go",
                 "internal/translate/check.go", "internal/translate/conversion.go",
                 "internal/projectconvert/convert.go", "internal/translate/migration.go"]:
        original = temp / name.replace("/", "-")
        data = subprocess.check_output(["git", "show", "227c049:" + name], cwd=root)
        if name.endswith("engine.go"):
            # This private test seam did not exist at the baseline. Forward to
            # the genuine original production implementation, ignoring the root.
            data += b'''\nfunc (e *Engine) RunRoot(ctx context.Context, files *projectfs.FS, options Options) (Report,error) {
                return e.Run(ctx,options)
            }\n'''
        original.write_bytes(data)
        replacements[str(root / name)] = str(original)
    future = temp / "future-input-tests.go"
    future.write_text("package translate\n")
    replacements[str(root / "internal/translate/root_inputs_test.go")] = str(future)
    future_audit = temp / "future-audit-tests.go"
    future_audit.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_audit_confinement_test.go")] = str(future_audit)
    future_comparison = temp / "future-comparison-tests.go"
    future_comparison.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_comparison_confinement_test.go")] = str(future_comparison)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev",
        "-run", "^(TestAuthoringTranslationPlanRejectsExternalFiles|TestTranslationRunRejectsExternalFilesBeforeProvider|TestTranslationRunRejectsParentsRearrangedDuringProvider|TestTranslationRunRetainsInternalAliases)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    boundaries = ["source", "target", "state-file", "state-parent", "state-discovery"]
    expected = {"TestAuthoringTranslationPlanRejectsExternalFiles/" + v for v in boundaries + ["style-guide", "glossary"]}
    expected |= {"TestTranslationRunRejectsExternalFilesBeforeProvider/" + v for v in boundaries}
    expected |= {"TestTranslationRunRejectsParentsRearrangedDuringProvider/" + v for v in ["target", "state-file"]}
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or top != {
            "TestAuthoringTranslationPlanRejectsExternalFiles", "TestTranslationRunRejectsExternalFilesBeforeProvider", "TestTranslationRunRejectsParentsRearrangedDuringProvider"} or \
            "--- PASS: TestTranslationRunRetainsInternalAliases" not in result.stdout:
        raise SystemExit("Did not reproduce all original planning/execution failures and alias controls")
