#!/usr/bin/env python3
"""Compare ordinary build and conversion bytes on identical input paths."""
import argparse
import hashlib
import json
import pathlib
import shutil
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("before")
parser.add_argument("after")
parser.add_argument("evidence")
parser.add_argument("self_corpus")
parser.add_argument("wails_corpus")
parser.add_argument("--baseline", default="e8eac75 production")
args = parser.parse_args()
evidence = pathlib.Path(args.evidence)
evidence.mkdir(parents=True, exist_ok=True)


def hashes(directory):
    return {str(path.relative_to(directory)): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in sorted(directory.rglob("*")) if path.is_file()}


def compare(label, before, after):
    changed = sorted(name for name in before.keys() | after.keys()
                     if before.get(name) != after.get(name))
    result = {"baseline": args.baseline, "files": len(after), "changed": changed,
              "sameSourceAndPath": True, "allBytesIdentical": not changed}
    (evidence / (label + "-equivalence.json")).write_text(json.dumps(result, indent=2) + "\n")
    if changed:
        raise SystemExit(label + " changed: " + repr(changed))


for label, directory, extra in [("self", args.self_corpus, []),
                                ("wails", args.wails_corpus, ["--no-purge-css"])]:
    outputs = []
    project = pathlib.Path(directory)
    for phase, binary in [("before", args.before), ("after", args.after)]:
        with (evidence / (label + "-" + phase + "-build.json")).open("w") as out, \
                (evidence / (label + "-" + phase + "-build.stderr")).open("w") as err:
            subprocess.run([binary, "build", "--strict", "--json"] + extra,
                           cwd=project, stdout=out, stderr=err, check=True)
        outputs.append(hashes(project / "site"))
    compare(label, *outputs)

# Reset outside the converter and use the same owned path for both binaries.
# Existing MPD documents remain unchanged; Markdown sources are converted.
selected = evidence / "conversion-corpus"
outputs = []
statuses = []
for phase, binary in [("before", args.before), ("after", args.after)]:
    if selected.exists():
        shutil.rmtree(selected)
    shutil.copytree(args.self_corpus, selected, ignore=shutil.ignore_patterns("site", ".mpress"))
    original = hashes(selected)
    with (evidence / ("conversion-" + phase + ".log")).open("w") as out:
        result = subprocess.run([binary, "convert", "--replace"], cwd=selected,
                                stdout=out, stderr=subprocess.STDOUT)
    outputs.append(hashes(selected))
    statuses.append(result.returncode)
    if result.returncode != 0 and outputs[-1] != original:
        raise SystemExit(phase + " rejected conversion modified inputs")
compare("conversion", *outputs)
before_message = (evidence / "conversion-before.log").read_text()
after_message = (evidence / "conversion-after.log").read_text()
if statuses[0] != statuses[1] or before_message != after_message:
    raise SystemExit("Conversion behavior or diagnostic changed")
(evidence / "conversion-result.json").write_text(json.dumps({
    "exitCodes": statuses, "sameDiagnostic": True,
    "rejectedConversionPreservesInputs": statuses[0] != 0,
    "successfulFullCorpusConversion": statuses[0] == 0,
}, indent=2) + "\n")
