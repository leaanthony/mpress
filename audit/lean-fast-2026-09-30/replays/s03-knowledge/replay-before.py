#!/usr/bin/env python3
"""Replay seven original authenticated knowledge reads and eleven loader failures."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-knowledge-replay-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/knowledge/artifact.go", "internal/knowledge/storage.go",
                 "internal/knowledge/storage_test.go", "internal/dev/server.go"]:
        original = temp / name.replace("/", "-")
        original.write_bytes(subprocess.check_output(["git", "show", "be3685d:" + name], cwd=root))
        replacements[str(root / name)] = str(original)
    empty = temp / "future-budget-tests.go"
    empty.write_text("package knowledge\n")
    replacements[str(root / "internal/knowledge/limits_test.go")] = str(empty)
    replacements[str(root / "internal/knowledge/limits.go")] = str(empty)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    pattern = "^(TestAuthoringKnowledgeRejectsExternalBundles|TestKnowledgeRejectsNonportableArtifactNames|TestKnowledgeRejectsArtifactsOutsideBundleWithinSite|TestKnowledgeRetainsInternalArtifactAliases|TestKnowledgeMountedMissingArtifactsRemainFatal)$"
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/dev", "./internal/knowledge",
                             "-run", pattern, "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    expected = {"TestAuthoringKnowledgeRejectsExternalBundles/" + name for name in
                ["output", "bundle", "manifest", "artifacts", "traversal", "versions", "version-bundle"]}
    expected |= {"TestKnowledgeRejectsNonportableArtifactNames/" + name for name in
                 ["traversal", "noncanonical", "absolute", "volume", "backslash", "repeated-slash",
                  "empty", "device", "trailing-dot", "trailing-space"]}
    top_failed = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    controls = ["TestKnowledgeRetainsInternalArtifactAliases", "TestKnowledgeMountedMissingArtifactsRemainFatal"]
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or top_failed != {
            "TestAuthoringKnowledgeRejectsExternalBundles", "TestKnowledgeRejectsNonportableArtifactNames",
            "TestKnowledgeRejectsArtifactsOutsideBundleWithinSite"} or not all("--- PASS: " + name in result.stdout for name in controls):
        raise SystemExit("Did not reproduce the exact original knowledge failures and positive controls")
