package site

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSVGObjectsUseIntrinsicDimensions(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed")
	}
	start := strings.Index(defaultThemeJS, "  const svgObjectRatios =")
	end := strings.Index(defaultThemeJS, "  const storage =")
	if start < 0 || end <= start {
		t.Fatal("SVG object initialiser missing")
	}
	harness := `
const assert = require('node:assert/strict');
const elements = [];
const document = {querySelectorAll: () => elements};
const getComputedStyle = object => ({aspectRatio: object.style.aspectRatio || 'auto'});
const make = (svg, attributes = {}) => {
  const object = {contentDocument: svg ? {documentElement: svg} : null, style: {},
    hasAttribute: name => name in attributes,
    addEventListener: (name, handler) => object.onLoad = handler};
  elements.push(object);
  return object;
};
const svg = (width, height) => ({localName:'svg',viewBox:{baseVal:{width,height}}});
const tall = make(svg(800,2524));
const wide = make(svg(1600,400));
const delayed = make(null);
const explicit = make(svg(800,2524),{height:'180'});
const styled = make(svg(800,2524)); styled.style.height='200px';
const ratio = make(svg(800,2524)); ratio.style.aspectRatio='1 / 1';
const invalid = make(svg(0,0));
const nonfinite = make(svg(Infinity,500));
const fallback = make({localName:'svg',width:{baseVal:{value:320,unitType:1}},height:{baseVal:{value:640,unitType:1}}});
const fixedSize = make({...svg(200,100),width:{baseVal:{value:100,unitType:1}},height:{baseVal:{value:100,unitType:1}}});
const percentSize = make({...svg(800,2524),width:{baseVal:{value:320,unitType:2}},height:{baseVal:{value:640,unitType:2}}});
const foreign = make(null); Object.defineProperty(foreign,'contentDocument',{get(){throw new Error('cross-origin')}});
` + defaultThemeJS[start:end] + `
assert.equal(tall.style.aspectRatio,'800 / 2524');
assert.equal(tall.style.height,undefined);
assert.equal(wide.style.aspectRatio,'1600 / 400');
assert.equal(fallback.style.aspectRatio,'320 / 640');
assert.equal(fixedSize.style.aspectRatio,'100 / 100');
assert.equal(percentSize.style.aspectRatio,'800 / 2524');
for (const item of [explicit,styled,invalid,nonfinite,foreign]) assert.equal(item.style.aspectRatio,undefined);
assert.equal(styled.style.height,'200px');
assert.equal(ratio.style.aspectRatio,'1 / 1');
delayed.contentDocument={documentElement:svg(400,1200)};
delayed.onLoad();
assert.equal(delayed.style.aspectRatio,'400 / 1200');
tall.onLoad();
assert.equal(tall.style.aspectRatio,'800 / 2524');
tall.contentDocument={documentElement:svg(1200,300)};
tall.onLoad();
assert.equal(tall.style.aspectRatio,'1200 / 300');
tall.style.aspectRatio='2 / 1';
tall.onLoad();
assert.equal(tall.style.aspectRatio,'2 / 1');
`
	if output, err := exec.Command(node, "-e", harness).CombinedOutput(); err != nil {
		t.Fatalf("SVG object sizing: %v\n%s", err, output)
	}
}
