#!/usr/bin/env python3
"""Replay negative confinement checks against a06977e production paths."""
import json
import pathlib
import re
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[4]
baseline = "a06977e"
with tempfile.TemporaryDirectory(prefix="mpress-s03-replay-") as work:
    temporary = pathlib.Path(work)
    def original(names):
        replacement = {}
        for name in names:
            target = temporary / name.replace("/", "-")
            data = subprocess.check_output(["git", "show", baseline + ":" + name], cwd=root)
            if name == "internal/config/output.go":
                # Later version guards require this helper at compile time. The
                # original output/preview methods under test remain untouched.
                helper = re.search(r"func \(c Config\) protectedInputs\(.*?\n\}",
                                   (root / name).read_text(), re.S)
                if helper is None:
                    raise SystemExit("Missing current version-guard compilation helper")
                data += ("\n" + helper.group(0) + "\n").encode()
            target.write_bytes(data)
            replacement[str(root / name)] = str(target)
        return replacement

    cases = [
        ("direct", original(["internal/dev/server.go", "internal/dev/mcp.go", "internal/dev/translation_models.go", "internal/config/output.go"]), "./internal/dev", "^Test(AuthoringRESTRejectsExternalSymlinkReadsAndWrites|AuthoringMCPRejectsExternalSymlinkPaths|AuthoringUploadRejectsExternalStaticAndContentParents|AuthoringBackupsRejectExternalParentWithoutChangingSource|NewServerPreservesUnsafePreviewPools|AuthoringConfigurationAndTreeDoNotFollowExternalLinks)$"),
        ("delegated", original(["internal/site/build.go", "internal/site/parse.go"]), "./internal/dev", "^TestAuthoring(RebuildDoesNotWriteCacheThroughExternalParent|PreviewRejectsOtherExternalInputs)$"),
        ("output", original(["internal/config/output.go"]), "./internal/config", "TestRemoveOutputProtectsConfiguredDataAndAncestors/(navigation|localized|translation_glossary|translation_style_guide|contributor_guide)"),
    ]
    # The old Server owns no root handle/Close method. Remove new handle cleanup
    # and the one new cleanup-method test from the baseline's compilation unit.
    current = (root / "internal/dev/confinement_test.go").read_text()
    current = re.sub(r"^\s*t.Cleanup\(func\(\) \{ _ = server.Close\(\) \}\)\n", "\n", current, flags=re.M)
    current = re.sub(r"\nfunc TestAuthoringPreviewCleanupRechecksRedirectedPool\(.*?(?=\nfunc )", "", current, flags=re.S)
    adapted = temporary / "direct-confinement_test.go"
    adapted.write_text(current)
    cases[0][1][str(root / "internal/dev/confinement_test.go")] = str(adapted)
    # The old Server owns no root handle. Later fixture files reference Close,
    # so provide a no-op lifecycle shim in an excluded future test compilation
    # unit. It does not participate in any original endpoint operation.
    lifecycle = temporary / "future-lifecycle-shim_test.go"
    lifecycle.write_text("package dev\nfunc (s *Server) Close() error { return nil }\n")
    cases[0][1][str(root / "internal/dev/translation_confinement_test.go")] = str(lifecycle)
    future_plan = temporary / "future-plan-tests.go"
    future_plan.write_text("package dev\n")
    cases[0][1][str(root / "internal/dev/translation_plan_confinement_test.go")] = str(future_plan)
    cases[0][1][str(root / "internal/dev/translation_audit_confinement_test.go")] = str(future_plan)
    cases[0][1][str(root / "internal/dev/translation_comparison_confinement_test.go")] = str(future_plan)
    cases[0][1][str(root / "internal/dev/conversion_confinement_test.go")] = str(future_plan)
    for label, replacement, package, pattern in cases:
        overlay = temporary / (label + ".json")
        overlay.write_text(json.dumps({"Replace": replacement}))
        print("Baseline replay:", label, flush=True)
        result = subprocess.run(["go", "test", "-overlay=" + str(overlay), package, "-run", pattern, "-count=1"], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        print(result.stdout, end="", flush=True)
        failures = re.findall(r"^--- FAIL: (\S+)", result.stdout, re.M)
        expected = {
            "direct": {"TestAuthoringRESTRejectsExternalSymlinkReadsAndWrites",
                       "TestAuthoringMCPRejectsExternalSymlinkPaths",
                       "TestAuthoringUploadRejectsExternalStaticAndContentParents",
                       "TestAuthoringBackupsRejectExternalParentWithoutChangingSource",
                       "TestNewServerPreservesUnsafePreviewPools",
                       "TestAuthoringConfigurationAndTreeDoNotFollowExternalLinks"},
            "delegated": {"TestAuthoringRebuildDoesNotWriteCacheThroughExternalParent",
                          "TestAuthoringPreviewRejectsOtherExternalInputs"},
            "output": {"TestRemoveOutputProtectsConfiguredDataAndAncestors"},
        }[label]
        if set(failures) != expected or len(failures) != len(expected) or "[build failed]" in result.stdout:
            raise SystemExit("Baseline replay did not reproduce the exact entry points: " + label)
        if label == "output":
            subfailures = set(re.findall(r"^    --- FAIL: (\S+)", result.stdout, re.M))
            expected_sub = {"TestRemoveOutputProtectsConfiguredDataAndAncestors/" + name for name in
                            ["navigation_outside_content", "localized_navigation_outside_content",
                             "translation_glossary", "translation_style_guide", "contributor_guide"]}
            if subfailures != expected_sub:
                raise SystemExit("Baseline replay did not reproduce all five protected-input failures")
        if result.returncode == 0:
            raise SystemExit("Expected failing-before regressions did not fail: " + label)
