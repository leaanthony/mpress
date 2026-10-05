#!/usr/bin/env python3
"""Replay invalid identities and ambiguous generated, legacy and mounted sections."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-knowledge-identity-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/knowledge/artifact.go", "internal/knowledge/store.go"]:
        original = temp / name.replace("/", "-")
        original.write_bytes(subprocess.check_output(["git", "show", "c4b8ddf:" + name], cwd=root))
        replacements[str(root / name)] = str(original)
    empty = temp / "future-budget-tests.go"
    empty.write_text("package knowledge\n")
    replacements[str(root / "internal/knowledge/limits_test.go")] = str(empty)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/knowledge",
        "-run", "^(TestKnowledgeRejectsDuplicateArtifactIdentities|TestKnowledgeSnapshotMatchingCurrentVersionHasDistinctResources|TestLoadAllIncludesMountedVersionsWithDistinctResources|TestCompressedKnowledgeRoundTrip|TestKnowledgeLegacyDuplicateChunkIDsRemainReadable|TestGeneratedRepeatedAndUnnamedSectionsHaveDistinctChunkIDs)$",
        "-count=1", "-v"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    subfail = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    topfail = set(re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M))
    expected_sub = {"TestKnowledgeRejectsDuplicateArtifactIdentities/" + kind for kind in ["page", "term"]}
    expected_sub |= {"TestKnowledgeLegacyDuplicateChunkIDsRemainReadable/" + threshold for threshold in ["1", "20971520"]}
    expected_top = {"TestKnowledgeRejectsDuplicateArtifactIdentities", "TestKnowledgeSnapshotMatchingCurrentVersionHasDistinctResources",
                    "TestKnowledgeLegacyDuplicateChunkIDsRemainReadable", "TestGeneratedRepeatedAndUnnamedSectionsHaveDistinctChunkIDs"}
    if result.returncode == 0 or "[build failed]" in result.stdout or subfail != expected_sub or topfail != expected_top or not all(
        "--- PASS: " + name in result.stdout for name in ["TestLoadAllIncludesMountedVersionsWithDistinctResources", "TestCompressedKnowledgeRoundTrip"]):
        raise SystemExit("Did not reproduce the exact identity failures and two positive controls")
