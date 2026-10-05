#!/usr/bin/env python3
"""Replay uncapped readers with small budgets, retaining corruption controls.

Adapters only make the new private test seams callable on the original source:
they forward to the old, unbounded production loaders/readers and ignore limits.
No allocator exhaustion, giant payload or compilation failure is a reproduction.
"""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-knowledge-budget-") as work:
    temp = pathlib.Path(work)
    replacements = {}
    for name in ["internal/knowledge/artifact.go", "internal/knowledge/storage.go"]:
        original = temp / name.replace("/", "-")
        original.write_bytes(subprocess.check_output(["git", "show", "f396e7e:" + name], cwd=root))
        replacements[str(root / name)] = str(original)
    adapter = temp / "original-reader-adapters.go"
    adapter.write_text((root / "internal/knowledge/limits.go").read_text() + '''
var ErrResourceLimit = fmt.Errorf("knowledge resource limit exceeded")
func readArtifactLimit(files *projectfs.FS, name string, ignored int64) ([]byte,error) {
    return readArtifact(files,name)
}
func readSizedArtifact(reader io.Reader, size, maximum int64) ([]byte,error) {
 return io.ReadAll(reader)
}
func loadSiteBudget(site *projectfs.FS, ignored *loadBudget) (*Store,error) {
    return loadSite(site)
}
func loadAllBudget(site *projectfs.FS, ignored *loadBudget) (*Store,error) {
    return LoadAllRoot(site)
}
''')
    replacements[str(root / "internal/knowledge/limits.go")] = str(adapter)
    overlay = temp / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    names = ["TestKnowledgeArtifactByteLimits", "TestKnowledgeGzipIntegrityAndExpansion",
             "TestKnowledgeBundleBudgets", "TestKnowledgeMountedBudgets", "TestKnowledgeUnsupportedSchemas",
             "TestKnowledgeArtifactReplacement", "TestCompressedKnowledgeRoundTrip", "TestKnowledgePlainReadSizeChanges"]
    result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./internal/knowledge",
        "-run", "^(" + "|".join(names) + ")$", "-count=1", "-v"], cwd=root,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end="", flush=True)
    failed = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
    expected = {"TestKnowledgeArtifactByteLimits/" + v for v in ["false", "true"]}
    expected |= {"TestKnowledgeGzipIntegrityAndExpansion/" + v for v in ["expansion", "combined-members"]}
    expected |= {"TestKnowledgeBundleBudgets/" + v for v in ["manifest", "artifact", "bundle", "aggregate"]}
    expected |= {"TestKnowledgeMountedBudgets/" + v for v in ["aggregate", "versions", "entries"]}
    expected |= {"TestKnowledgeUnsupportedSchemas/index", "TestKnowledgePlainReadSizeChanges/grown-outside-budget"}
    controls = ["TestKnowledgeGzipIntegrityAndExpansion/truncated-header",
                "TestKnowledgeGzipIntegrityAndExpansion/truncated-trailer",
                "TestKnowledgeGzipIntegrityAndExpansion/bad-checksum",
                "TestKnowledgeBundleBudgets/exact", "TestKnowledgeMountedBudgets/exact",
                "TestKnowledgeUnsupportedSchemas/manifest", "TestKnowledgeArtifactReplacement",
                "TestCompressedKnowledgeRoundTrip", "TestKnowledgePlainReadSizeChanges/unchanged",
                "TestKnowledgePlainReadSizeChanges/shrunk", "TestKnowledgePlainReadSizeChanges/grown-within-budget"]
    if result.returncode == 0 or "[build failed]" in result.stdout or failed != expected or not all(
            "--- PASS: " + name in result.stdout for name in controls):
        raise SystemExit("Did not reproduce the exact uncapped/schema failures and positive controls")
