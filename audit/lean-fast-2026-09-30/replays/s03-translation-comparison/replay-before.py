#!/usr/bin/env python3
"""Replay original comparison/estimate methods through real authenticated HTTP."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-comparison-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/translate/comparison.go", "internal/translate/engine.go", "internal/dev/translation_models.go"]:
        data = subprocess.check_output(["git", "show", "cbb4c5c:" + name], cwd=root)
        if name.endswith("comparison.go"):
            # New methods occur only in the future ownership test, not in the
            # selected old HTTP handlers. Forward to genuine original methods.
            data += b'''
func (e *Engine) Estimate() (ProjectEstimate,error) { return EstimateProject(e.Project,e.Config) }
func (e *Engine) CompareModels(ctx context.Context, language,file string,candidates []ComparisonCandidate) (ModelComparison,error) {
    return CompareProjectModelsForFile(ctx,e.Project,e.Config,language,file,candidates)
}
'''
        original = temp / name.replace("/", "-")
        original.write_bytes(data)
        replacements[str(root / name)] = str(original)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev",
        "-run", "^(TestAuthoringComparisonRejectsExternalFilesBeforeProvider|TestAuthoringEstimateRejectsExternalFilesBeforeTargetsExist|TestAuthoringComparisonRetainsInternalSourceAliasesWithoutWrites)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    cases = {
        "TestAuthoringComparisonRejectsExternalFilesBeforeProvider": ["source", "style-guide", "glossary"],
        "TestAuthoringEstimateRejectsExternalFilesBeforeTargetsExist": ["source", "style-guide"],
    }
    expected = {test + "/" + case for test, entries in cases.items() for case in entries}
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    top = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or top != set(cases) or \
            result.stdout.count("HTTP 200 provider calls=2") != 3 or \
            "--- PASS: TestAuthoringComparisonRetainsInternalSourceAliasesWithoutWrites" not in result.stdout:
        raise SystemExit("Did not reproduce original external reads/provider transmission and safe aliases")
