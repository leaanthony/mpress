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
  const plan = {goal: 'page', language: 'fr', file: 'guide.md', scope: 'stale', force: false, workers: 4, report: {pending: 2}};
  await runTranslation(plan);
  assert.deepEqual(calls, [
    {language: 'fr', file: 'guide.md', scope: 'stale', force: false, workers: 4},
    {action: 'audit', language: 'fr', file: 'guide.md', refine: true},
  ]);
  assert.equal(plan.file, 'guide.md');
  assert.notEqual(state.suppressReloadUntil, Number.MAX_SAFE_INTEGER);
})().catch(error => { console.error(error); process.exitCode = 1; });
`
	command := exec.Command(node, "-e", harness)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("translation workflow: %v\n%s", err, output)
	}
}

func TestPageTranslationOfferKeepsPageAndLanguage(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed")
	}
	script, err := os.ReadFile("assets/devbar.js")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(script), "  const translationPageStatus =")
	end := strings.Index(string(script), "  const showTranslations =")
	if start < 0 || end <= start {
		t.Fatal("page translation workflow not found")
	}
	harness := `
const assert = require('node:assert/strict');
const state = {project: {features: {translations: true}}};
const escapeHTML = value => String(value);
let onUpdate, selected, request;
const button = {dataset: {pageTranslate: 'de'}, addEventListener: (_, callback) => onUpdate = callback};
const container = {isConnected: true, innerHTML: '', querySelectorAll: () => [button]};
const q = () => container;
const showTranslations = context => { selected = context; };
const api = async (path, options) => {
  assert.equal(options, undefined, 'offering updates must never send a translation request');
  request = path;
  return {languageLabels: {fr: 'Français', de: 'Deutsch'}, report: {files: [
    {targetLanguage: 'fr', needsTranslation: 0, states: {untracked: 3}},
    {targetLanguage: 'de', needsTranslation: 1, states: {stale: 1, reviewed: 2}},
  ]}};
};
` + string(script)[start:end] + `
(async () => {
  await showPageTranslationStatus('docs/a page.md');
  assert.equal(request, 'translations?file=docs%2Fa%20page.md');
  assert.match(container.innerHTML, /Update this page in other languages/);
  assert.match(container.innerHTML, /Français/);
  assert.match(container.innerHTML, /Existing translation preserved/);
  assert.match(container.innerHTML, /1 passage to translate or update/);
  assert.doesNotMatch(container.innerHTML, /data-page-translate="fr"/);
  assert.match(container.innerHTML, /data-page-translate="de"/);
  onUpdate();
  assert.deepEqual(selected, {file: 'docs/a page.md', language: 'de'});
  assert.match(translationPageStatus({states: {conflict: 1}}), /human-edited passage to reconcile/);
  assert.match(translationPageStatus({states: {'migration-required': 1}}), /migration/);
  assert.equal(translationPageStatus({states: {reviewed: 3}}), 'Up to date with the source');
  container.isConnected = false;
  container.innerHTML = 'next screen';
  await showPageTranslationStatus('index.md');
  assert.doesNotMatch(container.innerHTML, /data-page-translate/);
})().catch(error => { console.error(error); process.exitCode = 1; });
`
	if output, err := exec.Command(node, "-e", harness).CombinedOutput(); err != nil {
		t.Fatalf("page translation offer: %v\n%s", err, output)
	}
}
