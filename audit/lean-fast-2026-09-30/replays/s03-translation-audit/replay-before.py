#!/usr/bin/env python3
"""Replay genuine pre-fix audit/refinement IO with forwarding test seams only."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-translation-audit-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/dev/server.go", "internal/translate/engine.go",
                 "internal/translate/audit.go", "internal/translate/refine.go",
                 "internal/translate/comparison.go", "internal/dev/translation_models.go",
                 "internal/translate/check.go", "internal/translate/conversion.go",
                 "internal/projectconvert/convert.go", "internal/translate/migration.go"]:
        original = temp / name.replace("/", "-")
        data = subprocess.check_output(["git", "show", "61d1462:" + name], cwd=root)
        if name.endswith("engine.go"):
            # The baseline has no general borrowed-root setup.
            # The adapter ignores the root and returns the original engine.
            data += b'''
func (e *Engine) BorrowRoot(root *projectfs.FS) (*Engine,error) {
    return e,nil
}
'''
        original.write_bytes(data)
        replacements[str(root / name)] = str(original)
    future = temp / "future-input-tests.go"
    future.write_text("package translate\n")
    replacements[str(root / "internal/translate/root_inputs_test.go")] = str(future)
    future_comparison = temp / "future-comparison-tests.go"
    future_comparison.write_text("package dev\n")
    replacements[str(root / "internal/dev/translation_comparison_confinement_test.go")] = str(future_comparison)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev",
        "-run", "^(TestAuthoringTranslationAuditRejectsExternalFiles|TestTranslationAuditRejectsExternalFilesBeforeReviewer|TestTranslationRefinementRejectsExternalFilesBeforeProvider|TestTranslationRefinementRejectsParentsRearrangedDuringProvider|TestTranslationAuditRefinementRetainInternalAliases)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    cases = {
        "TestAuthoringTranslationAuditRejectsExternalFiles": ["source", "target", "glossary"],
        "TestTranslationAuditRejectsExternalFilesBeforeReviewer": ["source", "target", "style-guide", "glossary"],
        "TestTranslationRefinementRejectsExternalFilesBeforeProvider": ["source", "target", "state-file", "state-parent", "style-guide", "glossary"],
        "TestTranslationRefinementRejectsParentsRearrangedDuringProvider": ["target", "state-file"],
    }
    expected = {test + "/" + case for test, entries in cases.items() for case in entries}
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or top != set(cases) or \
            "--- PASS: TestTranslationAuditRefinementRetainInternalAliases" not in result.stdout or \
            result.stdout.count("sentinel index.json changed") < 2 or "sentinel index.md changed" not in result.stdout:
        raise SystemExit("Did not reproduce exact audit/refinement failures, external overwrites and safe aliases")
