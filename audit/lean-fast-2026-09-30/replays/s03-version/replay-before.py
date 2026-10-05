#!/usr/bin/env python3
"""Replay version boundary regressions against original 72aea1a production."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
with tempfile.TemporaryDirectory(prefix="mpress-version-replay-") as work:
    temporary = pathlib.Path(work)
    original = temporary / "version.go"
    original.write_bytes(subprocess.check_output([
        "git", "show", "72aea1a:internal/version/version.go"
    ], cwd=root))
    overlay = temporary / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {
        str(root / "internal/version/version.go"): str(original)
    }}))
    cases = [
        ("package", "./internal/version", "^TestVersion(RemoveRejectsExternalArtifactParent|RemovePreservesProjectInputs|ListAndVerifyRejectExternalLinks|VerifyRejectsTraversalLabel|VerifyConfinesManifestToSnapshot|VerifyRejectsUnsupportedSchema|OperationsSharePortableLabels|VerifyRejectsNoncanonicalManifestPaths|RemoveRejectsSnapshotAliasToSource)$", 9),
        ("HTTP/MCP", "./internal/dev", "^TestAuthoringVersionsRejectExternalStore$", 1),
    ]
    for name, package, pattern, expected in cases:
        result = subprocess.run([
            "go", "test", "-overlay=" + str(overlay), package,
            "-run", pattern, "-count=1"
        ], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        print(name + " original-production replay:", flush=True)
        print(result.stdout, end="", flush=True)
        failures = re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M)
        if result.returncode == 0 or "[build failed]" in result.stdout or len(failures) != expected:
            raise SystemExit("Replay did not reproduce the expected behavior failures")
