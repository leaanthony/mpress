package site

// accessibilityBootstrapJS restores visual preferences before the stylesheet
// is evaluated. This avoids a flash of the default reading presentation.
const accessibilityBootstrapJS = `(function(){try{var p=JSON.parse(localStorage.getItem('mpress-accessibility')||'{}');var d=document.documentElement;var values={text:p.text||'default',siteWidth:p.siteWidth||'full',spacing:p.spacing?'relaxed':'default',font:p.readable?'readable':'default',contrast:p.contrast?'high':'default',links:p.links?'underlined':'default',focus:p.focus?'true':'false',guide:p.guide?'true':'false',motion:p.motion?'reduced':'default',colour:p.colour||'default',bionic:p.bionic?'true':'false'};Object.keys(values).forEach(function(k){d.setAttribute('data-a11y-'+k.replace(/[A-Z]/g,function(m){return '-'+m.toLowerCase();}),values[k]);});if(Number.isFinite(p.width))d.setAttribute('data-a11y-width',String(p.width));}catch(_){}})();`

const accessibilityCSS = `
/* Optional visitor accessibility and reading tools. */
html[data-a11y-site-width="fixed"] body > header,
html[data-a11y-site-width="fixed"] .docs-page .layout { width: min(100%, 1500px); margin-inline: auto; }
.accessibility-select .utility-menu-trigger {
  width: 38px;
  min-width: 38px;
  justify-content: center;
  padding: 0;
  border-color: transparent;
  border-radius: 6px;
  background: transparent;
  box-shadow: none;
  color: var(--muted);
}
.accessibility-select .utility-menu-trigger:hover,
.accessibility-select .utility-menu-trigger:focus-visible,
.accessibility-select .utility-menu-trigger[aria-expanded="true"] { border-color: transparent; background: var(--surface); color: var(--hover); box-shadow: none; }
.accessibility-select .utility-menu-trigger[data-active="true"] { color: var(--accent); }
.mpress-accessibility-panel[popover] {
  position: fixed;
  inset: auto;
  display: grid;
  width: min(560px, calc(100vw - 1.5rem));
  min-width: min(560px, calc(100vw - 1.5rem));
  max-width: min(560px, calc(100vw - 1.5rem));
  max-height: min(820px, calc(100dvh - 1rem));
  margin: 0;
  padding: 0;
  overflow: hidden;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  border: 1px solid color-mix(in srgb, var(--accent) 18%, var(--border));
  border-radius: 1rem;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-solid) 97%, transparent), color-mix(in srgb, var(--surface-solid) 92%, transparent));
  color: var(--text);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / .11), inset 0 -1px 0 rgb(255 255 255 / .05), 0 18px 50px rgb(0 0 0 / .28);
  -webkit-backdrop-filter: blur(16px) saturate(1.3);
  backdrop-filter: blur(16px) saturate(1.3);
}
.mpress-accessibility-panel::backdrop { background: transparent; }
.mpress-accessibility-header { position: relative; display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; padding: .9rem 1rem .75rem; background: transparent; }
.mpress-accessibility-header h2 { margin: 0; font-size: 18px; line-height: 1.3; }
.mpress-accessibility-header p { margin: .2rem 0 0; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
.mpress-accessibility-close { display: grid; width: 34px; height: 34px; flex: 0 0 auto; padding: 0; place-items: center; border: 0; border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; }
.mpress-accessibility-close:hover { background: var(--panel); color: var(--text); }
.mpress-accessibility-tabs { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); padding: 0 .65rem; border-bottom: 1px solid var(--border); }
.mpress-accessibility-tabs button { min-height: 38px; padding: .45rem .35rem; border: 0; border-bottom: 2px solid transparent; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12.5px; font-weight: 600; }
.mpress-accessibility-tabs button:hover { background: color-mix(in srgb, var(--text) 6%, transparent); color: var(--text); }
.mpress-accessibility-tabs button[aria-selected="true"] { border-bottom-color: var(--accent); background: color-mix(in srgb, var(--accent) 8%, transparent); color: var(--accent); }
.mpress-accessibility-tabs button:focus-visible { outline: none; background: color-mix(in srgb, var(--accent) 10%, transparent); color: var(--accent); box-shadow: inset 0 -2px var(--accent); }
.mpress-accessibility-body { min-height: 400px; overflow-y: auto; overscroll-behavior: contain; }
.mpress-accessibility-section { padding: .9rem 1rem 1rem; }
.mpress-accessibility-section[hidden] { display: none; }
.mpress-accessibility-section h3 { margin: 0 0 .2rem; color: var(--text); font-size: 13px; font-weight: 700; letter-spacing: normal; text-transform: none; }
.mpress-accessibility-section > p { margin: 0 0 .8rem; color: var(--muted); font-size: 12px; line-height: 1.5; }
.mpress-a11y-choice { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .35rem; margin-bottom: .75rem; }
.mpress-a11y-choice button { min-height: 38px; padding: .35rem .45rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px; font-weight: 650; }
.mpress-a11y-choice button:hover { border-color: color-mix(in srgb, var(--text) 30%, var(--border)); color: var(--text); }
.mpress-a11y-choice button[aria-checked="true"] { border-color: color-mix(in srgb, var(--accent) 58%, var(--border)); background: color-mix(in srgb, var(--accent) 12%, var(--surface)); color: var(--text); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 18%, transparent); }
.mpress-a11y-width { margin: 0 0 .75rem; padding: .7rem 0 .8rem; border-top: 1px solid color-mix(in srgb, var(--border) 70%, transparent); border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent); }
.mpress-a11y-width-heading, .mpress-a11y-width-footer { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.mpress-a11y-width-heading label { color: var(--text); font-size: 13.5px; font-weight: 600; }
.mpress-a11y-width-heading output { color: var(--muted); font-size: 12px; font-variant-numeric: tabular-nums; }
.mpress-a11y-width-mode { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .25rem; margin-top: .5rem; padding: .2rem; border: 1px solid var(--border); border-radius: 6px; background: color-mix(in srgb, var(--panel) 65%, transparent); }
.mpress-a11y-width-mode button { min-height: 30px; padding: .25rem .5rem; border: 0; border-radius: 4px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 11.5px; font-weight: 650; }
.mpress-a11y-width-mode button:hover { color: var(--text); }
.mpress-a11y-width-mode button[aria-checked="true"] { background: var(--surface-solid); color: var(--text); box-shadow: 0 1px 3px rgb(0 0 0 / .16); }
.mpress-a11y-width input[type="range"] { width: 100%; height: 24px; margin: .35rem 0 .2rem; accent-color: var(--accent); cursor: pointer; }
.mpress-a11y-width-footer small { color: var(--muted); font-size: 11.5px; line-height: 1.4; }
.mpress-a11y-width-footer button { padding: .2rem .4rem; border: 0; border-radius: 4px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 11.5px; font-weight: 650; }
.mpress-a11y-width-footer button:hover:not(:disabled) { background: var(--panel); color: var(--text); }
.mpress-a11y-width-footer button:disabled { opacity: .45; cursor: default; }
.mpress-a11y-toggle { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .8rem; align-items: center; min-height: 46px; padding: .35rem 0; cursor: pointer; }
.mpress-a11y-toggle + .mpress-a11y-toggle { border-top: 1px solid color-mix(in srgb, var(--border) 70%, transparent); }
.mpress-a11y-toggle strong { display: block; color: var(--text); font-size: 13.5px; font-weight: 600; line-height: 1.3; }
.mpress-a11y-toggle small { display: block; margin-top: .12rem; color: var(--muted); font-size: 11.5px; line-height: 1.4; }
.mpress-a11y-toggle input { position: relative; width: 36px; height: 20px; margin: 0; appearance: none; border: 1px solid var(--border); border-radius: 999px; background: color-mix(in srgb, var(--panel) 82%, transparent); cursor: pointer; box-shadow: inset 0 1px 2px rgb(0 0 0 / .18); transition: background .15s ease, border-color .15s ease, box-shadow .15s ease; }
.mpress-a11y-toggle input::after { content: ""; position: absolute; top: 50%; left: 2px; width: 14px; height: 14px; border-radius: 50%; background: var(--muted); box-shadow: 0 1px 3px rgb(0 0 0 / .28); transform: translateY(-50%); transition: transform .16s cubic-bezier(.2, .8, .2, 1), background .15s ease; }
.mpress-a11y-toggle input:checked { border-color: var(--accent); background: var(--accent); box-shadow: inset 0 1px 1px rgb(255 255 255 / .18), 0 0 0 1px color-mix(in srgb, var(--accent) 14%, transparent); }
.mpress-a11y-toggle input:checked::after { transform: translate(16px, -50%); background: var(--surface-solid); }
.mpress-a11y-toggle input:focus-visible { outline: 2px solid color-mix(in srgb, var(--accent) 58%, transparent); outline-offset: 2px; }
.mpress-a11y-field { display: grid; gap: .35rem; color: var(--text); font-size: 13px; font-weight: 600; }
.mpress-a11y-select { position: relative; }
.mpress-a11y-select-trigger { display: flex; width: 100%; min-height: 42px; align-items: center; justify-content: space-between; gap: .75rem; padding: .5rem .65rem; border: 1px solid var(--border); border-radius: 7px; background: color-mix(in srgb, var(--surface-solid) 66%, transparent); color: var(--text); cursor: pointer; font: inherit; font-weight: 500; text-align: left; box-shadow: inset 0 1px 0 rgb(255 255 255 / .06); -webkit-backdrop-filter: blur(12px); backdrop-filter: blur(12px); }
.mpress-a11y-select-trigger:hover, .mpress-a11y-select-trigger[aria-expanded="true"] { border-color: color-mix(in srgb, var(--accent) 46%, var(--border)); background: color-mix(in srgb, var(--surface-solid) 78%, transparent); }
.mpress-a11y-select-trigger:focus-visible { outline: 2px solid color-mix(in srgb, var(--accent) 58%, transparent); outline-offset: 2px; }
.mpress-a11y-select-trigger .lucide { color: var(--muted); transition: transform .16s ease; }
.mpress-a11y-select-trigger[aria-expanded="true"] .lucide { transform: rotate(180deg); }
.mpress-a11y-options { position: absolute; z-index: 4; top: calc(100% + .35rem); right: 0; left: 0; display: grid; gap: .12rem; padding: .35rem; border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border)); border-radius: 9px; background: linear-gradient(180deg, color-mix(in srgb, var(--surface-solid) 88%, transparent), color-mix(in srgb, var(--surface-solid) 76%, transparent)); box-shadow: inset 0 1px 0 rgb(255 255 255 / .08), 0 14px 34px rgb(0 0 0 / .32); -webkit-backdrop-filter: blur(18px) saturate(1.25); backdrop-filter: blur(18px) saturate(1.25); }
.mpress-a11y-options[hidden] { display: none; }
.mpress-a11y-option { position: relative; min-height: 36px; padding: .45rem 1.7rem .45rem .65rem; border: 0; border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12.5px; font-weight: 500; text-align: left; }
.mpress-a11y-option:hover, .mpress-a11y-option:focus-visible { outline: 0; background: color-mix(in srgb, var(--text) 7%, transparent); color: var(--text); }
.mpress-a11y-option[aria-selected="true"] { background: color-mix(in srgb, var(--accent) 11%, transparent); color: var(--text); }
.mpress-a11y-option[aria-selected="true"]::after { content: ""; position: absolute; top: 50%; right: .72rem; width: .45rem; height: .45rem; transform: translateY(-50%); border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent); }
.mpress-accessibility-footer { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .75rem 1rem .9rem; border-top: 1px solid var(--border); background: color-mix(in srgb, var(--surface-solid) 58%, transparent); }
.mpress-accessibility-footer small { color: var(--muted); font-size: 11px; }
.mpress-accessibility-reset { min-height: 36px; padding: .4rem .7rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--text); cursor: pointer; font-size: 12px; font-weight: 650; }
.mpress-accessibility-reset:hover { border-color: color-mix(in srgb, var(--text) 35%, var(--border)); }
.mpress-accessibility-demo { overflow: hidden; border: 1px solid var(--border); border-radius: 10px; background: color-mix(in srgb, var(--surface-solid) 74%, transparent); box-shadow: 0 18px 45px rgb(0 0 0 / .14); }
.mpress-accessibility-demo > header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; min-height: 64px; padding: .75rem 1rem; border-bottom: 1px solid var(--border); }
.mpress-accessibility-demo > header div { display: grid; gap: .08rem; }
.mpress-accessibility-demo > header span { color: var(--muted); font-size: 11px; font-weight: 750; letter-spacing: .08em; text-transform: uppercase; }
.mpress-accessibility-demo > header strong { color: var(--text); font-size: 14px; }
.mpress-accessibility-demo > header button { min-height: 34px; padding: .35rem .65rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--text); cursor: pointer; font-size: 12px; font-weight: 650; }
.mpress-accessibility-demo > .mpress-accessibility-section { min-height: 276px; }
.mpress-accessibility-demo .mpress-a11y-width-mode { margin-bottom: .65rem; }

html[data-a11y-text="large"] main :is(p, li, td, th, blockquote) { font-size: 18px !important; }
html[data-a11y-text="larger"] main :is(p, li, td, th, blockquote) { font-size: 20px !important; }
html[data-a11y-spacing="relaxed"] main :is(article, .blog-intro, .blog-grid, .landing-main) { letter-spacing: .022em; word-spacing: .075em; }
html[data-a11y-spacing="relaxed"] main :is(p, li, td, th, blockquote) { line-height: 1.95 !important; }
html[data-a11y-font="readable"] main,
html[data-a11y-font="readable"] main :is(h1, h2, h3, h4, h5, h6, p, li, td, th, blockquote) { font-family: "OpenDyslexic", "Atkinson Hyperlegible", Verdana, Tahoma, Arial, sans-serif !important; }
html[data-a11y-links="underlined"] main a:not(.mpress-button):not(.mpress-frontmatter-hero-action) { text-decoration: underline !important; text-decoration-thickness: .12em !important; text-underline-offset: .2em !important; }
html[data-a11y-focus="true"] :is(body > header, .sidebar, .toc, body > footer) { opacity: .34; filter: saturate(.45); transition: opacity .16s ease, filter .16s ease; }
html[data-a11y-focus="true"] :is(body > header, .sidebar, .toc, body > footer):is(:hover, :focus-within) { opacity: 1; filter: none; }
html[data-a11y-focus="true"] .docs-stage > main,
html[data-a11y-focus="true"] .blog-main,
html[data-a11y-focus="true"] .blog-article-main { background: var(--bg); box-shadow: 0 0 0 100vmax color-mix(in srgb, var(--bg) 34%, transparent); clip-path: inset(0 -100vmax); }
html[data-a11y-guide="true"] body::after { content: ""; position: fixed; z-index: 9997; left: 0; right: 0; top: calc(var(--mpress-guide-y, 50vh) - 1.45rem); height: 2.9rem; border-top: 1px solid color-mix(in srgb, var(--accent) 42%, transparent); border-bottom: 1px solid color-mix(in srgb, var(--accent) 42%, transparent); background: color-mix(in srgb, var(--accent) 8%, transparent); pointer-events: none; }
html[data-a11y-motion="reduced"] { scroll-behavior: auto !important; }
html[data-a11y-motion="reduced"] *,
html[data-a11y-motion="reduced"] *::before,
html[data-a11y-motion="reduced"] *::after { scroll-behavior: auto !important; transition-duration: .01ms !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; }
.mpress-bionic-prefix { font-weight: 800; }

html[data-a11y-contrast="high"] { --bg: #fff !important; --surface: #fff !important; --surface-solid: #fff !important; --panel: #eee !important; --text: #000 !important; --muted: #202020 !important; --border: #111 !important; --hover: #000 !important; --wails-accent-link: #000 !important; --wails-accent-readable: #000 !important; --wails-accent-strong: #000 !important; }
html[data-a11y-contrast="high"] :is(.mpress-admonition, .mpress-terminal, .mpress-codeframe, .mpress-card, .mpress-tabs, table) { border-width: 2px !important; }
html[data-a11y-contrast="high"] main a { text-decoration: underline !important; }
html[data-a11y-contrast="high"][data-theme="dark"] { --bg: #000 !important; --surface: #080808 !important; --surface-solid: #080808 !important; --panel: #151515 !important; --text: #fff !important; --muted: #eee !important; --border: #fff !important; --hover: #fff !important; --wails-accent-link: #fff !important; --wails-accent-readable: #fff !important; --wails-accent-strong: #fff !important; }
@media (prefers-color-scheme: dark) {
  html[data-a11y-contrast="high"][data-theme="system"] { --bg: #000 !important; --surface: #080808 !important; --surface-solid: #080808 !important; --panel: #151515 !important; --text: #fff !important; --muted: #eee !important; --border: #fff !important; --hover: #fff !important; --wails-accent-link: #fff !important; --wails-accent-readable: #fff !important; --wails-accent-strong: #fff !important; }
}

html[data-a11y-colour="red-green"] { --accent: #0072b2 !important; --success: #009e73 !important; --warning: #e69f00 !important; --danger: #cc79a7 !important; --wails-accent-link: #0072b2 !important; --wails-accent-readable: #005f95 !important; --wails-accent-strong: #0072b2 !important; }
html[data-a11y-colour="blue-yellow"] { --accent: #d55e00 !important; --success: #008442 !important; --warning: #cc3311 !important; --danger: #a61b61 !important; --wails-accent-link: #d55e00 !important; --wails-accent-readable: #9f4600 !important; --wails-accent-strong: #d55e00 !important; }
html[data-a11y-colour="low"] { --accent: #555b66 !important; --success: #59636e !important; --warning: #6f6252 !important; --danger: #6d5960 !important; --wails-accent-link: #555b66 !important; --wails-accent-readable: #3f444c !important; --wails-accent-strong: #555b66 !important; }
html[data-a11y-colour="low"] main :is(img, video, canvas) { filter: saturate(.25); }
html[data-theme="dark"][data-a11y-colour="red-green"] { --accent: #56b4e9 !important; --wails-accent-link: #56b4e9 !important; --wails-accent-readable: #8fd3f4 !important; --wails-accent-strong: #56b4e9 !important; }
html[data-theme="dark"][data-a11y-colour="blue-yellow"] { --accent: #f08a45 !important; --wails-accent-link: #f08a45 !important; --wails-accent-readable: #ffb17d !important; --wails-accent-strong: #f08a45 !important; }
html[data-theme="dark"][data-a11y-colour="low"] { --accent: #d5d8de !important; --wails-accent-link: #d5d8de !important; --wails-accent-readable: #f1f2f4 !important; --wails-accent-strong: #d5d8de !important; }
@media (prefers-color-scheme: dark) {
  html[data-theme="system"][data-a11y-colour="red-green"] { --accent: #56b4e9 !important; --wails-accent-link: #56b4e9 !important; --wails-accent-readable: #8fd3f4 !important; --wails-accent-strong: #56b4e9 !important; }
  html[data-theme="system"][data-a11y-colour="blue-yellow"] { --accent: #f08a45 !important; --wails-accent-link: #f08a45 !important; --wails-accent-readable: #ffb17d !important; --wails-accent-strong: #f08a45 !important; }
  html[data-theme="system"][data-a11y-colour="low"] { --accent: #d5d8de !important; --wails-accent-link: #d5d8de !important; --wails-accent-readable: #f1f2f4 !important; --wails-accent-strong: #d5d8de !important; }
}

@media (prefers-contrast: more) {
  main a { text-decoration: underline !important; }
  :is(.mpress-admonition, .mpress-terminal, .mpress-codeframe, .mpress-card, .mpress-tabs, table) { border-width: 2px !important; }
}
@media (forced-colors: active) {
  .mpress-accessibility-panel, button, select, input { border: 1px solid ButtonText; }
  .mpress-accessibility-panel { forced-color-adjust: auto; background: Canvas; color: CanvasText; }
  .mpress-token-keyword, .mpress-token-function { font-weight: 800; color: Highlight; }
  .mpress-token-comment { font-style: italic; color: GrayText; }
  .mpress-token-string { text-decoration: underline; color: LinkText; }
}
@media (prefers-reduced-transparency: reduce) {
  .mpress-accessibility-panel[popover] { background: var(--surface-solid); -webkit-backdrop-filter: none; backdrop-filter: none; }
  .mpress-a11y-options { background: var(--surface-solid); -webkit-backdrop-filter: none; backdrop-filter: none; }
}
@media (max-width: 760px) {
  .accessibility-select { display: inline-flex; }
  .accessibility-select .utility-menu-trigger { width: 34px; min-width: 34px; }
  .mpress-accessibility-panel[popover] { width: calc(100vw - 1rem); max-height: calc(100dvh - 1rem); overflow-y: auto; }
}
`

const accessibilityJS = `
(() => {
  const doc = document.documentElement;
  const storage = {
    get(key) { try { return localStorage.getItem(key); } catch (_) { return null; } },
    set(key, value) { try { localStorage.setItem(key, value); } catch (_) {} }
  };
  const accessibilityButton = document.querySelector('#accessibility');
  const accessibilityPanel = document.querySelector('#mpress-accessibility-panel');
  if (accessibilityButton && accessibilityPanel) {
	const accessibilityLaunchers = [...document.querySelectorAll('[data-open-accessibility]')];
	accessibilityLaunchers.forEach(launcher => launcher.addEventListener('click', () => {
	  if (!accessibilityPanel.matches(':popover-open')) accessibilityButton.click();
	}));
    const accessibilityShortcut = document.body.dataset.shortcutAccessibility || 'Mod+A';
    const keyboard = window.mpressKeyboard;
    if (keyboard && accessibilityShortcut !== 'None') {
      accessibilityButton.setAttribute('aria-keyshortcuts', keyboard.aria(accessibilityShortcut));
      accessibilityButton.title = 'Accessibility settings (' + keyboard.label(accessibilityShortcut) + ')';
    }
    const defaultAccessibility = {text: 'default', siteWidth: 'full', width: null, widthUnit: 'percent', spacing: false, readable: false, contrast: false, links: false, focus: false, guide: false, motion: false, colour: 'default', bionic: false};
    const loadAccessibility = () => {
      try { return Object.assign({}, defaultAccessibility, JSON.parse(storage.get('mpress-accessibility') || '{}')); }
      catch (_) { return Object.assign({}, defaultAccessibility); }
    };
    let accessibility = loadAccessibility();
    const readingRoots = () => [...document.querySelectorAll('.docs-stage article, .blog-article, .blog-main, .landing-main')];
    const excludedBionicParent = element => element.closest('pre, code, kbd, button, [role="button"], a.mpress-button, input, textarea, select, script, style, svg, strong, b, [aria-hidden="true"], .mpress-search-overlay, .mpress-accessibility-panel');
    const removeBionic = () => {
      document.querySelectorAll('strong[data-mpress-bionic]').forEach(prefix => prefix.replaceWith(document.createTextNode(prefix.textContent || '')));
      readingRoots().forEach(root => root.normalize());
    };
    const addBionic = () => {
      removeBionic();
      readingRoots().forEach(root => {
        const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
        const nodes = [];
        while (walker.nextNode()) {
          const node = walker.currentNode;
          if (node.textContent.trim() && node.parentElement && !excludedBionicParent(node.parentElement)) nodes.push(node);
        }
        nodes.forEach(node => {
          const text = node.textContent;
          const pattern = /\p{L}[\p{L}\p{M}'’\-]{1,}/gu;
          let match;
          let offset = 0;
          let changed = false;
          const fragment = document.createDocumentFragment();
          while ((match = pattern.exec(text))) {
            changed = true;
            fragment.append(document.createTextNode(text.slice(offset, match.index)));
            const characters = [...match[0]];
            const middle = Math.ceil(characters.length / 2);
            const prefix = document.createElement('strong');
            prefix.className = 'mpress-bionic-prefix';
            prefix.dataset.mpressBionic = '';
            prefix.textContent = characters.slice(0, middle).join('');
            fragment.append(prefix, document.createTextNode(characters.slice(middle).join('')));
            offset = match.index + match[0].length;
          }
          if (!changed) return;
          fragment.append(document.createTextNode(text.slice(offset)));
          node.replaceWith(fragment);
        });
      });
    };
    const setData = (name, value) => doc.setAttribute('data-a11y-' + name, value);
    const widthControl = accessibilityPanel.querySelector('[data-a11y-width]');
    const widthOutput = accessibilityPanel.querySelector('[data-a11y-width-output]');
    const widthReset = accessibilityPanel.querySelector('[data-a11y-width-reset]');
    const widthModes = [...accessibilityPanel.querySelectorAll('[data-a11y-width-mode]')];
    const normaliseWidthUnit = value => value === 'fixed' ? 'fixed' : 'percent';
    const normaliseWidth = (value, unit = accessibility.widthUnit) => {
      if (value === null || value === undefined || value === '' || typeof value === 'boolean') return null;
      const width = Number(value);
      if (!Number.isFinite(width)) return null;
      return normaliseWidthUnit(unit) === 'fixed' ? Math.min(2400, Math.max(480, Math.round(width))) : Math.min(100, Math.max(40, Math.round(width)));
    };
    const availableContentWidth = () => {
      const layout = document.querySelector('.docs-page .layout');
      const sidebar = layout?.querySelector(':scope > .sidebar');
      const toc = layout?.querySelector('.docs-stage > .toc');
      if (!layout) return 0;
      const sidebarWidth = sidebar && getComputedStyle(sidebar).display !== 'none' ? sidebar.getBoundingClientRect().width : 0;
      const tocWidth = toc && getComputedStyle(toc).display !== 'none' ? toc.getBoundingClientRect().width : 0;
      const gap = parseFloat(getComputedStyle(doc).getPropertyValue('--layout-content-toc-gap')) || 0;
      return Math.max(0, layout.getBoundingClientRect().width - sidebarWidth - tocWidth - gap);
    };
    const applyContentWidth = () => {
      const unit = normaliseWidthUnit(accessibility.widthUnit);
      const width = normaliseWidth(accessibility.width, unit);
      if (width === null || window.matchMedia('(max-width: 1180px)').matches) {
        doc.style.removeProperty('--layout-content-width');
        return;
      }
      const available = availableContentWidth();
      if (available > 0) doc.style.setProperty('--layout-content-width', Math.min(available, unit === 'fixed' ? width : Math.max(480, available * width / 100)) + 'px');
    };
    const activeAccessibilityCount = () => Object.entries(accessibility).filter(([key, value]) => key === 'widthUnit' ? false : key === 'siteWidth' ? value === 'fixed' : key === 'text' || key === 'colour' ? value !== 'default' : key === 'width' ? normaliseWidth(value) !== null : Boolean(value)).length;
    const applyAccessibility = (save = true) => {
      setData('text', ['default', 'large', 'larger'].includes(accessibility.text) ? accessibility.text : 'default');
      setData('site-width', accessibility.siteWidth === 'fixed' ? 'fixed' : 'full');
      setData('spacing', accessibility.spacing ? 'relaxed' : 'default');
      setData('font', accessibility.readable ? 'readable' : 'default');
      setData('contrast', accessibility.contrast ? 'high' : 'default');
      setData('links', accessibility.links ? 'underlined' : 'default');
      setData('focus', accessibility.focus ? 'true' : 'false');
      setData('guide', accessibility.guide ? 'true' : 'false');
      setData('motion', accessibility.motion ? 'reduced' : 'default');
      setData('colour', ['default', 'red-green', 'blue-yellow', 'low'].includes(accessibility.colour) ? accessibility.colour : 'default');
      setData('bionic', accessibility.bionic ? 'true' : 'false');
      accessibility.widthUnit = normaliseWidthUnit(accessibility.widthUnit);
      const width = normaliseWidth(accessibility.width);
      if (width === null) doc.removeAttribute('data-a11y-width'); else doc.setAttribute('data-a11y-width', String(width));
      applyContentWidth();
      if (accessibility.bionic) addBionic(); else removeBionic();
      const active = activeAccessibilityCount();
      accessibilityButton.dataset.active = String(active > 0);
      accessibilityButton.setAttribute('aria-label', active ? 'Accessibility settings, ' + active + ' active' : 'Accessibility settings');
      if (save) storage.set('mpress-accessibility', JSON.stringify(accessibility));
    };
    const accessibilitySelects = [...accessibilityPanel.querySelectorAll('[data-a11y-select]')];
    const setAccessibilitySelectOpen = (control, open, focusSelected = false) => {
      const trigger = control.querySelector('.mpress-a11y-select-trigger');
      const menu = control.querySelector('.mpress-a11y-options');
      if (!trigger || !menu) return;
      trigger.setAttribute('aria-expanded', String(open));
      menu.hidden = !open;
      if (open && focusSelected) {
        const selected = control.querySelector('.mpress-a11y-option[aria-selected="true"]') || control.querySelector('.mpress-a11y-option');
        selected?.focus();
      }
    };
    const syncAccessibilitySelect = control => {
      const value = accessibility[control.dataset.a11ySelect] || 'default';
      const valueLabel = control.querySelector('[data-a11y-select-value]');
      const options = [...control.querySelectorAll('.mpress-a11y-option')];
      let selectedLabel = '';
      options.forEach(option => {
        const selected = option.dataset.value === value;
        option.setAttribute('aria-selected', String(selected));
        option.tabIndex = selected ? 0 : -1;
        if (selected) selectedLabel = option.textContent.trim();
      });
      if (valueLabel && selectedLabel) valueLabel.textContent = selectedLabel;
    };
    const syncAccessibilityControls = () => {
      document.querySelectorAll('[data-a11y-toggle]').forEach(control => control.checked = Boolean(accessibility[control.dataset.a11yToggle]));
      document.querySelectorAll('[data-a11y-choice]').forEach(control => {
        const checked = accessibility[control.dataset.a11yChoice] === control.dataset.value;
        control.setAttribute('aria-checked', String(checked));
        control.tabIndex = checked ? 0 : -1;
      });
      accessibilitySelects.forEach(syncAccessibilitySelect);
      const unit = normaliseWidthUnit(accessibility.widthUnit);
      const width = normaliseWidth(accessibility.width, unit);
      const available = Math.max(480, Math.floor(availableContentWidth()));
      if (widthControl) {
        widthControl.min = unit === 'fixed' ? '480' : '40';
        widthControl.max = unit === 'fixed' ? String(Math.max(480, Math.min(2400, available))) : '100';
        widthControl.step = unit === 'fixed' ? '10' : '1';
        widthControl.value = String(width === null ? (unit === 'fixed' ? Math.min(720, available) : 70) : Math.min(width, Number(widthControl.max)));
      }
      if (widthOutput) widthOutput.textContent = width === null ? 'Default' : width + (unit === 'fixed' ? 'px' : '%');
      if (widthReset) widthReset.disabled = width === null;
      widthModes.forEach(mode => {
        const checked = mode.dataset.a11yWidthMode === unit;
        mode.setAttribute('aria-checked', String(checked));
        mode.tabIndex = checked ? 0 : -1;
      });
    };
    const accessibilityTabs = [...accessibilityPanel.querySelectorAll('[data-a11y-tab]')];
    const activateAccessibilityTab = (tab, moveFocus = true) => {
      if (!tab) return;
      accessibilityTabs.forEach(candidate => {
        const selected = candidate === tab;
        candidate.setAttribute('aria-selected', String(selected));
        candidate.tabIndex = selected ? 0 : -1;
        const panel = accessibilityPanel.querySelector('[data-a11y-tabpanel="' + candidate.dataset.a11yTab + '"]');
        if (panel) panel.hidden = !selected;
      });
      if (moveFocus) tab.focus();
    };
    accessibilityTabs.forEach((tab, index) => {
      tab.addEventListener('click', () => activateAccessibilityTab(tab, true));
      tab.addEventListener('keydown', event => {
        let next = index;
        if (event.key === 'ArrowRight') next = (index + 1) % accessibilityTabs.length;
        else if (event.key === 'ArrowLeft') next = (index - 1 + accessibilityTabs.length) % accessibilityTabs.length;
        else if (event.key === 'Home') next = 0;
        else if (event.key === 'End') next = accessibilityTabs.length - 1;
        else return;
        event.preventDefault();
        activateAccessibilityTab(accessibilityTabs[next], true);
      });
    });
    document.querySelectorAll('[data-a11y-toggle]').forEach(control => control.addEventListener('change', () => {
      accessibility[control.dataset.a11yToggle] = control.checked;
      applyAccessibility();
      syncAccessibilityControls();
    }));
    document.querySelectorAll('[data-a11y-choice]').forEach(control => {
      control.addEventListener('click', () => {
        accessibility[control.dataset.a11yChoice] = control.dataset.value;
        applyAccessibility();
        syncAccessibilityControls();
      });
      control.addEventListener('keydown', event => {
        const choices = [...control.parentElement.querySelectorAll('[data-a11y-choice]')];
        const current = choices.indexOf(control);
        let next = current;
        if (event.key === 'ArrowRight' || event.key === 'ArrowDown') next = (current + 1) % choices.length;
        else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') next = (current - 1 + choices.length) % choices.length;
        else if (event.key === 'Home') next = 0;
        else if (event.key === 'End') next = choices.length - 1;
        else return;
        event.preventDefault();
        choices[next].click();
        choices[next].focus();
      });
    });
    widthControl?.addEventListener('input', () => {
      accessibility.width = normaliseWidth(widthControl.value);
      applyAccessibility();
      syncAccessibilityControls();
    });
    widthReset?.addEventListener('click', () => {
      accessibility.width = null;
      accessibility.widthUnit = 'percent';
      applyAccessibility();
      syncAccessibilityControls();
      widthControl?.focus();
    });
    widthModes.forEach((mode, index) => {
      mode.addEventListener('click', () => {
        const nextUnit = normaliseWidthUnit(mode.dataset.a11yWidthMode);
        if (nextUnit === normaliseWidthUnit(accessibility.widthUnit)) return;
        const currentWidth = document.querySelector('.docs-stage > main')?.getBoundingClientRect().width || 720;
        const available = Math.max(1, availableContentWidth());
        accessibility.widthUnit = nextUnit;
        accessibility.width = nextUnit === 'fixed' ? Math.round(currentWidth) : Math.round(currentWidth / available * 100);
        applyAccessibility();
        syncAccessibilityControls();
        mode.focus();
      });
      mode.addEventListener('keydown', event => {
        let next = index;
        if (event.key === 'ArrowRight' || event.key === 'ArrowDown') next = (index + 1) % widthModes.length;
        else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') next = (index - 1 + widthModes.length) % widthModes.length;
        else return;
        event.preventDefault();
        widthModes[next].click();
      });
    });
    accessibilitySelects.forEach(control => {
      const trigger = control.querySelector('.mpress-a11y-select-trigger');
      const options = [...control.querySelectorAll('.mpress-a11y-option')];
      trigger?.addEventListener('click', () => {
        const open = trigger.getAttribute('aria-expanded') !== 'true';
        accessibilitySelects.forEach(candidate => setAccessibilitySelectOpen(candidate, candidate === control && open));
      });
      trigger?.addEventListener('keydown', event => {
        if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return;
        event.preventDefault();
        setAccessibilitySelectOpen(control, true, true);
      });
      options.forEach((option, index) => {
        option.addEventListener('click', () => {
          accessibility[control.dataset.a11ySelect] = option.dataset.value || 'default';
          applyAccessibility();
          syncAccessibilityControls();
          setAccessibilitySelectOpen(control, false);
          trigger?.focus();
        });
        option.addEventListener('keydown', event => {
          let next = index;
          if (event.key === 'ArrowDown') next = (index + 1) % options.length;
          else if (event.key === 'ArrowUp') next = (index - 1 + options.length) % options.length;
          else if (event.key === 'Home') next = 0;
          else if (event.key === 'End') next = options.length - 1;
          else if (event.key === 'Escape') {
            event.preventDefault();
            setAccessibilitySelectOpen(control, false);
            trigger?.focus();
            return;
          } else return;
          event.preventDefault();
          options[next].focus();
        });
      });
    });
    accessibilityPanel.addEventListener('click', event => {
      accessibilitySelects.forEach(control => {
        if (!control.contains(event.target)) setAccessibilitySelectOpen(control, false);
      });
    });
    accessibilityPanel.querySelector('[data-a11y-reset]')?.addEventListener('click', () => {
      accessibility = Object.assign({}, defaultAccessibility);
      applyAccessibility();
      syncAccessibilityControls();
    });
    document.querySelectorAll('[data-a11y-demo]').forEach(demo => {
      const tabs = [...demo.querySelectorAll('[data-a11y-demo-tab]')];
      tabs.forEach(tab => tab.addEventListener('click', () => {
        tabs.forEach(candidate => {
          const selected = candidate === tab;
          candidate.setAttribute('aria-selected', String(selected));
          const panel = demo.querySelector('[data-a11y-demo-panel="' + candidate.dataset.a11yDemoTab + '"]');
          if (panel) panel.hidden = !selected;
        });
      }));
    });
    accessibilityPanel.querySelector('[data-a11y-close]')?.addEventListener('click', () => {
      accessibilityPanel.hidePopover();
      accessibilityButton.focus();
    });
    document.addEventListener('keydown', event => {
      if (!keyboard || keyboard.editableTarget(event.target) || !keyboard.matches(event, accessibilityShortcut)) return;
      event.preventDefault();
      if (accessibilityPanel.matches(':popover-open')) {
        accessibilityPanel.hidePopover();
        accessibilityButton.focus();
      } else accessibilityButton.click();
    });
    accessibilityPanel.addEventListener('toggle', event => {
      const open = event.newState ? event.newState === 'open' : accessibilityPanel.matches(':popover-open');
      accessibilityButton.setAttribute('aria-expanded', String(open));
      if (open) {
        syncAccessibilityControls();
        const selectedTab = accessibilityTabs.find(tab => tab.getAttribute('aria-selected') === 'true') || accessibilityTabs[0];
        selectedTab?.focus();
      } else accessibilitySelects.forEach(control => setAccessibilitySelectOpen(control, false));
    });
    document.addEventListener('pointermove', event => {
      if (accessibility.guide) doc.style.setProperty('--mpress-guide-y', event.clientY + 'px');
    }, {passive: true});
    window.addEventListener('storage', event => {
      if (event.key !== 'mpress-accessibility') return;
      accessibility = loadAccessibility();
      applyAccessibility(false);
      syncAccessibilityControls();
    });
    window.addEventListener('resize', () => {
      if (normaliseWidth(accessibility.width) !== null) {
        applyContentWidth();
        syncAccessibilityControls();
      }
    }, {passive: true});
    applyAccessibility(false);
    syncAccessibilityControls();
  }
})();
`
