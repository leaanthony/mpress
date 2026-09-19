package dev

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestTranslationWorkflowPreservesMarkdownSources(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed")
	}
	script, err := os.ReadFile("assets/devbar.js")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(script), "const runTranslation = async plan => {")
	if start < 0 {
		t.Fatal("translation workflow not found")
	}
	end := strings.Index(string(script)[start:], "const renderTranslationResult")
	if end < 0 {
		t.Fatal("translation result renderer not found")
	}
	workflow := string(script)[start : start+end]
	harness := `
const assert = require('node:assert/strict');
const state = {suppressReloadUntil: 0};
const calls = [];
const languageLabel = value => value;
const formats = () => ({nativeMPD: false, markdown: 100});
const selectedStage = () => ({});
const modelName = () => 'test';
const renderProgress = () => {};
const updateBuild = () => {};
const loadTranslationData = async () => {};
const renderTranslationResult = () => {};
const setDrawerBody = (...args) => { throw new Error(JSON.stringify(args)); };
const api = async (path, options) => { calls.push(JSON.parse(options.body)); return {state: {}, report: {}}; };
` + workflow + `
(async () => {
  const plan = {language: 'fr', file: 'guide.md', scope: 'stale', report: {pending: 2}};
  await runTranslation(plan);
  assert.deepEqual(calls, [plan, {action: 'audit', language: 'fr', file: 'guide.md', refine: true}]);
  assert.equal(plan.file, 'guide.md');
  assert.notEqual(state.suppressReloadUntil, Number.MAX_SAFE_INTEGER);
})().catch(error => { console.error(error); process.exitCode = 1; });
`
	command := exec.Command(node, "-e", harness)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("translation workflow: %v\n%s", err, output)
	}
}
