package site

const defaultThemeCSS = `
:root {
  --accent: #5375f6;
  --accent-text: color-mix(in srgb, var(--accent) 45%, #000);
  --accent-fill: color-mix(in srgb, var(--accent) 45%, #000);
  --accent-fill-text: #fff;
  --hover-light-seed: #7593ff;
  --hover-dark-seed: #95aaff;
  --hover-light: color-mix(in srgb, var(--hover-light-seed) 45%, #000);
  --hover-dark: color-mix(in srgb, var(--hover-dark-seed) 30%, #fff);
  --hover: var(--hover-light);
  --success: #147a70;
  --warning: #936b2d;
  --danger: #b94c59;
  --accent-2: var(--success);
  --accent-soft: color-mix(in srgb, var(--accent) 9%, transparent);
  --bg: #f7f8fa;
  --surface: #fff;
  --surface-solid: #fff;
  --panel: #eef1f5;
  --text: #171a21;
  --muted: #626a78;
  --border: #dfe3e8;
  --shadow: 0 12px 30px rgba(17, 24, 39, .08);
  --code: #f4f6fa;
  --code-text: #252d3a;
  --code-comment: #627082;
  --code-string: #8f5414;
  --code-number: #a54855;
  --code-keyword: #7043ae;
  --code-function: #245ea8;
  --code-type: #087d6c;
  --code-copy: #5f6877;
  --code-copy-border: #cbd2dc;
  --code-copy-hover: #171a21;
  --code-added-bg: rgba(20, 122, 112, .1);
  --code-added-text: #087366;
  --code-removed-bg: rgba(185, 76, 89, .1);
  --code-removed-text: #a43d4a;
  --qr-bg: #fff;
  --qr-foreground: #171a21;
  --terminal-bg: #fbfcfe;
  --terminal-title-bg: #eef1f5;
  --terminal-text: #283242;
  --terminal-border: #d8dde5;
  --terminal-prompt: #087d6c;
  --terminal-command: #18202c;
  --terminal-comment: #667085;
  --terminal-output: #566174;
  --terminal-copy: #5f6877;
  --terminal-copy-border: #cbd2dc;
  --terminal-copy-hover: #171a21;
  --terminal-shadow: 0 4px 16px rgba(17, 24, 39, .08);
  --component-radius: 8px;
  --component-space: 1.5rem;
  --blog-image-default-background: #fff;
  --search-hover: color-mix(in srgb, var(--text) 7%, var(--surface-solid));
  --search-selected: color-mix(in srgb, var(--text) 10%, var(--surface-solid));
  --search-highlight: color-mix(in srgb, var(--text) 17%, var(--surface-solid));
  --search-outline: color-mix(in srgb, var(--text) 22%, var(--border));
  --layout-sidebar-width: 18.75rem;
  --layout-content-width: 50rem;
  --layout-wide-content-width: 64rem;
  --layout-toc-width: 16rem;
  --layout-content-toc-gap: 2.5rem;
  color-scheme: light;
}

html[data-theme="dark"] {
  --accent-text: color-mix(in srgb, var(--accent) 30%, #fff);
  --hover: var(--hover-dark);
  --bg: #0d1119;
  --surface: #131925;
  --surface-solid: #131925;
  --panel: #1a2230;
  --text: #f0f2f5;
  --muted: #9da6b5;
  --border: #273143;
  --shadow: 0 24px 70px rgba(0, 0, 0, .32);
  --code: #060a12;
  --code-text: #eef4ff;
  --code-comment: #adb7b7;
  --code-string: #ecc48d;
  --code-number: #f78c6c;
  --code-keyword: #c792ea;
  --code-function: #82aaff;
  --code-type: #7fdbca;
  --code-copy: #aeb8c9;
  --code-copy-border: #3a4558;
  --code-copy-hover: #fff;
  --code-added-bg: rgba(27, 143, 101, .13);
  --code-added-text: #91dec4;
  --code-removed-bg: rgba(214, 70, 86, .12);
  --code-removed-text: #f3a6af;
  --qr-bg: #060a12;
  --qr-foreground: #f0f2f5;
  --terminal-bg: #1f2026;
  --terminal-title-bg: #2a2c33;
  --terminal-text: #f4f7fb;
  --terminal-border: rgba(255, 255, 255, .12);
  --terminal-prompt: #62d2bf;
  --terminal-command: #f4f7fb;
  --terminal-comment: #7f8da3;
  --terminal-output: #9daabd;
  --terminal-copy: #aeb8c9;
  --terminal-copy-border: #3a4558;
  --terminal-copy-hover: #fff;
  --terminal-shadow: 0 4px 16px rgba(0, 0, 0, .2);
  --success: #52a698;
  --warning: #b48b54;
  --danger: #c76a74;
  --blog-image-default-background: #000;
  color-scheme: dark;
}

@media (prefers-color-scheme: dark) {
  html[data-theme="system"] {
    --accent-text: color-mix(in srgb, var(--accent) 30%, #fff);
    --hover: var(--hover-dark);
    --bg: #0d1119;
    --surface: #131925;
    --surface-solid: #131925;
    --panel: #1a2230;
    --text: #f0f2f5;
    --muted: #9da6b5;
    --border: #273143;
    --shadow: 0 24px 70px rgba(0, 0, 0, .32);
    --code: #060a12;
    --code-text: #eef4ff;
    --code-comment: #adb7b7;
    --code-string: #ecc48d;
    --code-number: #f78c6c;
    --code-keyword: #c792ea;
    --code-function: #82aaff;
    --code-type: #7fdbca;
    --code-copy: #aeb8c9;
    --code-copy-border: #3a4558;
    --code-copy-hover: #fff;
    --code-added-bg: rgba(27, 143, 101, .13);
    --code-added-text: #91dec4;
    --code-removed-bg: rgba(214, 70, 86, .12);
    --code-removed-text: #f3a6af;
    --qr-bg: #060a12;
    --qr-foreground: #f0f2f5;
    --terminal-bg: #1f2026;
    --terminal-title-bg: #2a2c33;
    --terminal-text: #f4f7fb;
    --terminal-border: rgba(255, 255, 255, .12);
    --terminal-prompt: #62d2bf;
    --terminal-command: #f4f7fb;
    --terminal-comment: #7f8da3;
    --terminal-output: #9daabd;
    --terminal-copy: #aeb8c9;
    --terminal-copy-border: #3a4558;
    --terminal-copy-hover: #fff;
    --terminal-shadow: 0 4px 16px rgba(0, 0, 0, .2);
    --success: #52a698;
    --warning: #b48b54;
    --danger: #c76a74;
    --blog-image-default-background: #000;
    color-scheme: dark;
  }
}

* { box-sizing: border-box; }
html { max-width: 100%; overflow-x: clip; scroll-padding-top: 88px; }
::selection { background: color-mix(in srgb, var(--accent) 30%, transparent); color: var(--text); }
body {
  max-width: 100%;
  min-height: 100vh;
  margin: 0;
  overflow-x: clip;
  background: var(--bg);
  color: var(--text);
  font: 400 16px/1.75 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  -webkit-font-smoothing: antialiased;
}
html.js .mpress-audience, html.js .mpress-conditional { display: none; }
html.js .mpress-audience.mpress-visible, html.js .mpress-conditional.mpress-visible { display: block; }
a { color: var(--accent-text); text-decoration-thickness: .08em; text-underline-offset: .18em; }
a:hover { text-decoration: underline; }
button, input, select { font: inherit; }
.lucide { display: block; flex: 0 0 auto; }
.mpress-accessibility-icon { display: block; flex: 0 0 auto; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
:focus-visible { outline: 3px solid color-mix(in srgb, var(--accent) 52%, transparent); outline-offset: 3px; }
.skip { position: fixed; left: -9999px; }
.skip:focus { left: 1rem; top: 1rem; z-index: 20; padding: .65rem 1rem; border-radius: 10px; background: var(--surface-solid); box-shadow: var(--shadow); }

body > header {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: .9rem;
  height: 58px;
  padding: 0 1.25rem;
  border-bottom: 1px solid var(--border);
  background: var(--bg);
}
.brand { display: inline-flex; flex: 0 0 auto; align-items: center; gap: .65rem; color: var(--text); font-size: 24px; font-weight: 600; letter-spacing: 0; line-height: 1.75; text-decoration: none; }
.brand:hover { text-decoration: none; }
.brand-mark {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 5px;
  background: var(--accent);
  color: #fff;
  font-size: 1rem;
  font-weight: 900;
}
.brand img { display: block; max-width: var(--mpress-logo-width, 190px); max-height: 42px; }
.brand-logo { width: var(--mpress-logo-width, 160px); object-fit: contain; object-position: left center; }
.theme-logo { display: inline-flex; flex: 0 0 var(--mpress-logo-width, 160px); width: var(--mpress-logo-width, 160px); height: 30px; align-items: center; }
.theme-logo img { width: 100%; height: 100%; object-fit: contain; object-position: left center; }
.theme-logo-dark { display: none !important; }
html[data-theme="dark"] .theme-logo-light { display: none !important; }
html[data-theme="dark"] .theme-logo-dark { display: block !important; }
@media (prefers-color-scheme: dark) {
  html[data-theme="system"] .theme-logo-light { display: none !important; }
  html[data-theme="system"] .theme-logo-dark { display: block !important; }
}
.primary-links { display: flex; flex: 0 0 auto; align-items: center; gap: .2rem; }
.primary-links a { padding: .38rem .55rem; border-radius: 5px; color: var(--muted); font-size: 16px; font-weight: 400; text-decoration: none; }
.primary-links a:hover { background: var(--surface); color: var(--hover); text-decoration: none; }
.header-spacer { margin-left: auto; }
.primary-links a.header-link-button { display: inline-flex; height: 28px; align-items: center; padding: 0 .8rem; border: 1px solid transparent; border-radius: 5px; font-size: 16px; font-weight: 400; line-height: 1; }
.primary-links a.header-link-button-primary { background: var(--accent); color: #fff; }
.primary-links a.header-link-button-primary:hover { background: color-mix(in srgb, var(--accent) 84%, #000); color: #fff; }
.primary-links a.header-link-button-secondary { border-color: var(--border); background: var(--surface); color: var(--text); }
.primary-links a.header-link-button-secondary:hover { border-color: color-mix(in srgb, var(--border) 55%, var(--text)); background: var(--panel); color: var(--text); }
.primary-links a.header-link-button-outline { min-width: 4rem; justify-content: center; border-color: color-mix(in srgb, var(--text) 16%, transparent); background: color-mix(in srgb, var(--text) 7%, transparent); color: color-mix(in srgb, var(--text) 88%, var(--muted)); box-shadow: inset 0 1px 0 color-mix(in srgb, var(--text) 8%, transparent); }
.primary-links a.header-link-button-outline:hover { border-color: color-mix(in srgb, var(--text) 28%, transparent); background: color-mix(in srgb, var(--text) 12%, transparent); color: var(--hover); box-shadow: inset 0 1px 0 color-mix(in srgb, var(--text) 12%, transparent), 0 2px 8px color-mix(in srgb, #000 16%, transparent); }
.primary-links a.header-link-button-custom { border-color: color-mix(in srgb, var(--header-button-color) 68%, var(--border)); background: transparent; color: var(--header-button-color); }
.primary-links a.header-link-button-custom:hover { border-color: var(--header-button-color); background: color-mix(in srgb, var(--header-button-color) 13%, transparent); color: var(--header-button-color); }
.search { position: relative; flex: 0 1 22rem; width: min(22rem, 28vw); margin-left: auto; }
.search > span { position: absolute; left: -9999px; }
.search button {
  width: 100%;
  height: 40px;
  padding: .5rem 4.4rem .5rem 2.35rem;
  border: 1px solid var(--border);
  border-radius: 7px;
  outline: none;
  background: var(--surface);
  color: var(--text);
  cursor: pointer;
  font-size: 14px;
  text-align: left;
  transition: border-color .15s, box-shadow .15s;
}
.search > .lucide-search { position: absolute; z-index: 1; left: .8rem; top: 50%; color: var(--muted); transform: translateY(-50%); pointer-events: none; }
.search-shortcut { position: absolute; z-index: 1; top: 50%; right: .5rem; display: inline-flex; height: 24px; align-items: center; gap: .3rem; padding: 0 .42rem; transform: translateY(-50%); border: 1px solid var(--border); border-radius: 4px; background: color-mix(in srgb, var(--surface-solid) 72%, transparent); color: var(--muted); font: 500 12px/1 ui-sans-serif, system-ui, sans-serif; pointer-events: none; }
.search-shortcut kbd { font: inherit; }
.search button:focus, .search button[aria-expanded="true"] { border-color: var(--search-outline); box-shadow: 0 0 0 3px color-mix(in srgb, var(--text) 7%, transparent); }
.mpress-search-overlay { position: fixed; z-index: 1000; inset: 0; display: none; align-items: flex-start; justify-content: center; padding: min(9vh, 5.5rem) 1rem 1rem; background: rgb(5 8 13 / .56); backdrop-filter: blur(7px); overscroll-behavior: contain; }
.mpress-search-overlay.open { display: flex; }
body.mpress-search-open { overflow: hidden; }
.mpress-search-dialog { display: grid; grid-template: auto auto minmax(0, 1fr) / minmax(0, 1fr); width: min(92vw, 620px); max-height: min(760px, 78vh); overflow: hidden; border: 1px solid color-mix(in srgb, var(--border) 82%, var(--text) 18%); border-radius: 12px; background: var(--surface-solid); color: var(--text); box-shadow: 0 32px 80px rgb(0 0 0 / .38), 0 1px 0 rgb(255 255 255 / .06) inset; }
.mpress-search-dialog.has-results { width: min(92vw, 900px); grid-template-columns: minmax(320px, .92fr) minmax(320px, 1.08fr); }
.mpress-search-query { grid-column: 1 / -1; display: flex; min-height: 62px; align-items: center; gap: .75rem; padding: .7rem 1rem; border-bottom: 1px solid var(--border); }
.mpress-search-query > .lucide { flex: 0 0 auto; color: var(--muted); }
.mpress-search-query input { min-width: 0; flex: 1; padding: 0; border: 0; outline: 0; background: transparent; color: var(--text); font: 500 17px/1.4 ui-sans-serif, system-ui, sans-serif; }
.mpress-search-query input::placeholder { color: var(--muted); font-weight: 400; }
.mpress-search-close { display: inline-grid; min-width: 30px; height: 25px; padding: 0 .42rem; place-items: center; border: 1px solid var(--border); border-radius: 4px; background: var(--surface); color: var(--muted); cursor: pointer; font: 650 11px/1 ui-sans-serif, system-ui, sans-serif; }
.mpress-search-close:hover { border-color: var(--search-outline); background: var(--search-hover); color: var(--text); }
.mpress-search-meta { grid-column: 1 / -1; display: flex; min-height: 38px; align-items: center; justify-content: space-between; gap: 1rem; padding: .45rem 1rem; border-bottom: 1px solid var(--border); color: var(--muted); font-size: 12px; }
.mpress-search-hint { white-space: nowrap; }
.mpress-search-hint kbd { padding: 0; border: 0; background: transparent; color: inherit; font: inherit; }
.mpress-search-list { min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: .45rem; }
.mpress-search-item { display: block; width: 100%; padding: .68rem .75rem; border: 0; border-radius: 7px; background: transparent; color: var(--text); text-align: left; text-decoration: none; cursor: pointer; }
.mpress-search-item:hover { background: var(--search-hover); color: var(--text); text-decoration: none; }
.mpress-search-item.active { background: var(--search-selected); color: var(--text); text-decoration: none; box-shadow: 0 0 0 1px var(--search-outline) inset; }
.mpress-search-item-title { display: flex; align-items: baseline; gap: .45rem; color: inherit; font-size: 14px; font-weight: 680; line-height: 1.35; }
.mpress-search-item-path { overflow: hidden; color: var(--muted); font-size: 11px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.mpress-search-item-snippet { display: -webkit-box; margin-top: .18rem; overflow: hidden; color: var(--muted); font-size: 12.5px; line-height: 1.45; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.mpress-search-item mark, .mpress-search-preview mark { padding: .02em .08em; border-radius: 2px; background: var(--search-highlight); color: var(--text); box-shadow: 0 0 0 1px color-mix(in srgb, var(--text) 8%, transparent) inset; }
.mpress-search-preview { display: none; min-height: 0; overflow-y: auto; border-left: 1px solid var(--border); padding: 1rem 1.1rem 1.2rem; color: var(--muted); }
.mpress-search-dialog.has-results .mpress-search-preview { display: block; }
.mpress-search-preview h2 { margin: 0 0 .5rem; color: var(--text); font-size: 18px; line-height: 1.35; }
.mpress-search-preview > p { margin: 0 0 .8rem; font-size: 13px; line-height: 1.55; }
.mpress-search-preview-headings { display: grid; gap: .16rem; margin: .8rem 0 1rem; padding: 0; list-style: none; }
.mpress-search-preview-headings a { display: block; padding: .25rem .35rem; border-radius: 4px; color: var(--muted); font-size: 12px; text-decoration: none; }
.mpress-search-preview-headings a:hover { background: var(--search-hover); color: var(--text); }
.mpress-search-preview-headings .level-3 { padding-left: .9rem; }
.mpress-search-preview-body { display: -webkit-box; overflow: hidden; font-size: 12.5px; line-height: 1.6; -webkit-box-orient: vertical; -webkit-line-clamp: 12; }
.mpress-search-empty { padding: 2.8rem 1rem; color: var(--muted); text-align: center; }
.mpress-search-empty strong { display: block; margin-bottom: .35rem; color: var(--text); font-size: 14px; }
.mpress-search-recent-heading { display: flex; align-items: center; justify-content: space-between; padding: .35rem .75rem .45rem; color: var(--muted); font-size: 11px; font-weight: 700; letter-spacing: .06em; text-transform: uppercase; }
.mpress-search-clear { padding: .2rem .35rem; border: 0; border-radius: 3px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; letter-spacing: 0; text-transform: none; }
.mpress-search-clear:hover { background: var(--search-hover); color: var(--text); }
.mpress-search-recent .mpress-search-item { display: flex; align-items: center; gap: .55rem; color: var(--muted); font-size: 13px; }
.mpress-search-recent .mpress-search-item > .lucide { width: 14px; height: 14px; opacity: .7; }
.header-links { display: flex; flex: 0 0 auto; align-items: center; gap: 0; }
.header-links > a { color: var(--muted); font-weight: 400; }
.header-utility-cluster { position: relative; display: inline-flex; align-items: center; gap: 0; margin-left: .65rem; padding-left: .65rem; }
.header-utility-cluster::before { content: ""; position: absolute; left: 0; top: 50%; width: 1px; height: 32px; transform: translateY(-50%); background: var(--border); }
.header-utility-cluster > .header-group { margin-left: 0; padding-left: 0; }
.header-utility-cluster > .header-group::before { display: none; }
.header-group { position: relative; display: inline-flex; align-items: center; min-height: 34px; }
.header-group + .header-group { margin-left: .65rem; padding-left: .65rem; }
.header-group + .header-group::before { content: ""; position: absolute; left: 0; top: 50%; width: 1px; height: 32px; transform: translateY(-50%); background: var(--border); }
.social-links { gap: .1rem; }
.social-link { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 6px; color: var(--muted); }
.social-link:hover { background: var(--surface); color: var(--text); text-decoration: none; }
.header-links select, .header-links button, #menu {
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  cursor: pointer;
}
.utility-select { gap: .25rem; color: var(--muted); }
.header-links select { max-width: 128px; height: 34px; padding: .3rem 1.5rem .3rem .2rem; color: var(--muted); font-size: 14px; font-weight: 400; }
.header-links button { color: var(--muted); }
.header-links button:hover { background: var(--surface); color: var(--text); }
.utility-menu { position: relative; }
.header-links .utility-menu-trigger {
  display: inline-flex;
  width: auto;
  min-width: 34px;
  height: 34px;
  align-items: center;
  gap: .38rem;
  padding: .34rem .58rem;
  border: 1px solid color-mix(in srgb, var(--border) 88%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--surface-solid) 46%, transparent);
  color: var(--muted);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
  box-shadow: inset 0 1px 0 color-mix(in srgb, white 10%, transparent), 0 1px 2px rgb(0 0 0 / .12);
  transition: background .15s ease, border-color .15s ease, color .15s ease, box-shadow .15s ease;
}
.header-links .utility-menu-trigger:hover,
.header-links .utility-menu-trigger:focus-visible,
.header-links .utility-menu-trigger[aria-expanded="true"] {
  border-color: color-mix(in srgb, var(--accent) 42%, var(--border));
  background: color-mix(in srgb, var(--surface-solid) 72%, transparent);
  color: var(--text);
  box-shadow: inset 0 1px 0 color-mix(in srgb, white 13%, transparent), 0 0 0 3px color-mix(in srgb, var(--accent) 10%, transparent);
}
.utility-menu-trigger > .lucide-chevron-down { opacity: .7; transition: transform .16s ease; }
.utility-menu-trigger[aria-expanded="true"] > .lucide-chevron-down { transform: rotate(180deg); }
.header-links .language-select .utility-menu-trigger {
  width: 42px;
  min-width: 42px;
  justify-content: center;
  gap: .08rem;
  padding: 0;
  border-color: transparent;
  border-radius: 6px;
  background: transparent;
  box-shadow: none;
  color: var(--muted);
}
.header-links .language-select .utility-menu-trigger:hover,
.header-links .language-select .utility-menu-trigger:focus-visible,
.header-links .language-select .utility-menu-trigger[aria-expanded="true"] {
  border-color: transparent;
  background: var(--surface);
  color: var(--hover);
  box-shadow: none;
}
.language-select .lucide-languages { flex: 0 0 auto; }
.language-select .lucide-chevron-down { width: 11px; height: 11px; }
.header-links .header-utility-cluster > :is(.accessibility-select, .language-select) .utility-menu-trigger { width: 34px; min-width: 34px; height: 34px; padding: 0; justify-content: center; color: var(--muted); }
.header-links .header-utility-cluster > .version-select .utility-menu-trigger { color: var(--muted); }
.utility-menu-panel[popover] {
  position: fixed;
  inset: auto;
  min-width: 13.5rem;
  max-width: min(19rem, calc(100vw - 1.5rem));
  max-height: min(26rem, calc(100dvh - 5rem));
  margin: 0;
  padding: .45rem;
  overflow-y: auto;
  list-style: none;
  border: 1px solid color-mix(in srgb, var(--accent) 18%, var(--border));
  border-radius: 1rem;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-solid) 88%, transparent), color-mix(in srgb, var(--surface-solid) 74%, transparent));
  color: var(--text);
  opacity: 0;
  transform: translateY(-5px) scale(.985);
  transform-origin: top right;
  -webkit-backdrop-filter: blur(16px) saturate(1.3);
  backdrop-filter: blur(16px) saturate(1.3);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / .11), inset 0 -1px 0 rgb(255 255 255 / .05), 0 18px 50px rgb(0 0 0 / .28);
  transition: opacity .15s ease, transform .15s ease, display .15s allow-discrete, overlay .15s allow-discrete;
}
.utility-menu-panel[popover]:popover-open { opacity: 1; transform: translateY(0) scale(1); }
@starting-style { .utility-menu-panel[popover]:popover-open { opacity: 0; transform: translateY(-5px) scale(.985); } }
.utility-menu-panel li { margin: 0; }
.utility-menu-panel a {
  position: relative;
  display: flex;
  min-height: 2.2rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: .52rem .75rem;
  border-radius: .7rem;
  color: var(--text);
  font-size: 14px;
  font-weight: 500;
  line-height: 1.2;
  text-decoration: none;
  transition: background .14s ease, color .14s ease;
}
.utility-menu-panel a:hover,
.utility-menu-panel a:focus-visible { background: color-mix(in srgb, var(--text) 7%, transparent); color: var(--text); outline: none; }
.utility-menu-panel a[aria-current="true"] {
  padding-right: 2rem;
  background: color-mix(in srgb, var(--accent) 11%, transparent);
  color: var(--accent);
  font-weight: 650;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 22%, transparent);
}
.utility-menu-panel a[aria-current="true"]::after {
  content: "";
  position: absolute;
  top: 50%;
  right: .78rem;
  width: .45rem;
  height: .45rem;
  transform: translateY(-50%);
  border-radius: 999px;
  background: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 17%, transparent);
}
.utility-menu-panel a small { color: var(--muted); font-size: 11px; font-weight: 500; white-space: nowrap; }
.utility-version-menu a { align-items: flex-start; flex-direction: column; gap: .12rem; }
.utility-version-menu a[aria-current="true"]::after { top: 50%; }
.utility-version-label { line-height: 1.2; }
.theme-toggle { display: grid; width: 34px; min-width: 34px; height: 34px; min-height: 34px; place-items: center; padding: 0; border: 0; border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; }
.header-group.theme-toggle { width: 34px; min-width: 34px; }
.theme-toggle:hover { background: var(--surface); color: var(--hover); }
.theme-toggle:focus { outline: none; }
.theme-toggle:focus-visible { background: var(--surface); color: var(--hover); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--muted) 42%, transparent); }
.theme-toggle .lucide { display: none; grid-area: 1 / 1; }
.theme-toggle[data-theme-mode="system"] .lucide-monitor,
.theme-toggle[data-theme-mode="dark"] .lucide-moon,
.theme-toggle[data-theme-mode="light"] .lucide-sun { display: inline-flex; }
.social-link:hover { color: var(--hover); }
#menu { display: none; flex: 0 0 44px; width: 44px; min-width: 44px; height: 44px; padding: 11px; }
#menu .lucide { margin: auto; }

.layout { display: grid; grid-template-columns: var(--layout-sidebar-width) minmax(0, 1fr); width: 100%; margin: 0; padding: 0; }
.mpress-not-found { display: grid; min-height: calc(100dvh - 58px); padding: clamp(3rem, 10vw, 8rem) 1.25rem; place-items: center; text-align: center; }
.mpress-not-found > div { width: min(100%, 34rem); }
.mpress-not-found-code { display: block; margin-bottom: .7rem; color: var(--accent); font-size: .78rem; font-weight: 800; letter-spacing: .14em; }
.mpress-not-found h1 { margin: 0; font-size: clamp(2.4rem, 8vw, 4.5rem); font-weight: 650; line-height: 1.05; letter-spacing: -.045em; text-wrap: balance; }
.mpress-not-found p { max-width: 34rem; margin: 1.1rem auto 1.8rem; color: var(--muted); }
.mpress-not-found .mpress-button { display: inline-flex; }
.docs-stage { --layout-stage-fill: max(var(--layout-content-toc-gap), calc((100% - var(--layout-content-width) - var(--layout-toc-width)) / 2)); container-type: inline-size; display: grid; grid-template-columns: var(--layout-stage-fill) minmax(0, var(--layout-content-width)) var(--layout-stage-fill) minmax(0, var(--layout-toc-width)); width: 100%; min-width: 0; }
.docs-page-wide .docs-stage { --layout-stage-fill: max(var(--layout-content-toc-gap), calc((100% - var(--layout-wide-content-width) - var(--layout-toc-width)) / 2)); grid-template-columns: var(--layout-stage-fill) minmax(0, var(--layout-wide-content-width)) var(--layout-stage-fill) minmax(0, var(--layout-toc-width)); }
.docs-stage > main { grid-column: 2; }
.docs-stage > .toc { grid-column: 4; }
.sidebar, .toc { position: sticky; top: 58px; align-self: start; height: calc(100vh - 58px - var(--mpress-devbar-height, 0px)); overflow: auto; overscroll-behavior: contain; scrollbar-gutter: stable; scrollbar-width: thin; user-select: none; }
.sidebar { padding: 2rem 1rem 3rem; border-right: 1px solid var(--border); }
.sidebar a, .sidebar summary, .sidebar .nav-label { display: block; margin: 1px 0; padding: .38rem .55rem; border-radius: 3px; color: var(--muted); cursor: pointer; font-size: 14px; font-weight: 400; line-height: 1.4; text-decoration: none; }
.sidebar a:hover { background: var(--accent-soft); color: var(--text); }
.sidebar a.active { color: var(--accent); font-weight: 600; }
.sidebar nav > a, .sidebar nav > .nav-label { color: var(--text); font-size: 16px; font-weight: 600; }
.sidebar details > div { margin-left: 0; padding-left: .65rem; }
.sidebar summary { display: flex; align-items: center; justify-content: space-between; list-style: none; color: var(--text); font-size: 16px; font-weight: 600; letter-spacing: normal; text-transform: none; }
.sidebar summary::-webkit-details-marker { display: none; }
.sidebar summary .lucide { color: currentColor; transition: transform .15s ease; }
.sidebar details[open] > summary .lucide { transform: rotate(90deg); }
body.mpress-workspace-document .layout { grid-template-columns: minmax(14.5rem, 16rem) minmax(0, 1fr); }
body.mpress-workspace-document .mpress-workspace-sidebar { padding: .9rem .7rem calc(4.5rem + var(--mpress-devbar-height, 0px)); background: color-mix(in srgb, var(--surface) 82%, var(--bg)); }
body.mpress-workspace-document .mpress-workspace-sidebar nav { display: grid; gap: .38rem; }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-back { display: inline-flex; min-height: 2.55rem; align-items: center; gap: .55rem; margin: 0 0 .45rem; padding: .42rem .55rem; color: var(--text); font-size: 13px; font-weight: 650; }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-back:hover { background: transparent; color: var(--accent); }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-back .lucide { width: 15px; height: 15px; }
body.mpress-workspace-document .mpress-workspace-group { display: grid; gap: .15rem; padding: .18rem 0; }
body.mpress-workspace-document .mpress-workspace-group-toggle { appearance: none; display: flex; width: 100%; min-height: 2rem; align-items: center; justify-content: space-between; gap: .5rem; padding: .25rem .55rem; border: 0; border-radius: 4px; background: transparent; box-shadow: none; color: var(--muted); cursor: pointer; font: 750 10px/1.3 ui-sans-serif, system-ui, sans-serif; letter-spacing: .08em; text-align: left; text-transform: uppercase; }
body.mpress-workspace-document .mpress-workspace-group-toggle:hover { color: var(--text); }
body.mpress-workspace-document .mpress-workspace-group-toggle:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
body.mpress-workspace-document .mpress-workspace-group-toggle .lucide { width: 14px; height: 14px; color: var(--muted); transition: transform .16s ease; }
body.mpress-workspace-document .mpress-workspace-group-toggle[aria-expanded="false"] .lucide { transform: rotate(-90deg); }
body.mpress-workspace-document .mpress-workspace-group-links { display: grid; gap: 2px; padding-left: .18rem; }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link:not(.mpress-workspace-back) { display: flex; min-height: 2.15rem; align-items: center; gap: .6rem; margin: 0; padding: .36rem .55rem; border-radius: 5px; color: var(--muted); font-size: 13px; font-weight: 500; line-height: 1.35; }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link:not(.mpress-workspace-back) .lucide { width: 15px; height: 15px; color: color-mix(in srgb, var(--muted) 84%, var(--text)); }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link:not(.mpress-workspace-back):hover { background: var(--accent-soft); color: var(--text); }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link.active { background: var(--accent); color: #fff; font-weight: 650; }
body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link.active .lucide { color: currentColor; }
.toc { display: flex; min-width: 0; flex-direction: column; gap: .15rem; padding: 2.4rem 1rem 2rem 1.2rem; overflow-x: hidden; border-left: 1px solid var(--border); }
.toc strong { margin-bottom: .55rem; font-size: 18px; font-weight: 600; letter-spacing: normal; line-height: 1.2; }
.toc-mobile-icon { display: none; }
.toc a { max-width: calc(100% + 1.23rem); margin-left: -1.23rem; padding: .3rem 0 .3rem 1.2rem; overflow-wrap: anywhere; border-left: 2px solid transparent; color: var(--muted); font-size: 13px; font-weight: 400; line-height: 1.25; text-decoration: none; transition: border-color .15s, color .15s, background-color .15s; }
.toc a.toc-level-3 { padding-left: 2.2rem; }
.toc a:hover { color: var(--text); }
.toc a.active { border-left-color: var(--accent); background: linear-gradient(90deg, var(--accent-soft), transparent 78%); color: var(--accent); font-weight: 400; }

.docs-stage > main { width: 100%; min-width: 0; padding: 2.4rem 0 6rem; }
article { font-size: 15px; font-weight: 400; line-height: 1.7; }
.docs-page-header { position: relative; margin-bottom: 2rem; padding-bottom: 1.8rem; }
.docs-page-header::after { position: absolute; right: calc(max(var(--layout-content-toc-gap), calc((100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)) * -1); bottom: 0; left: calc(max(var(--layout-content-toc-gap), calc((100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)) * -1); height: 1px; background: var(--border); content: ""; }
.docs-page-wide .docs-page-header::after { right: calc(max(var(--layout-content-toc-gap), calc((100cqw - var(--layout-wide-content-width) - var(--layout-toc-width)) / 2)) * -1); left: calc(max(var(--layout-content-toc-gap), calc((100cqw - var(--layout-wide-content-width) - var(--layout-toc-width)) / 2)) * -1); }
article h1 { max-width: 28ch; margin: 0 0 .7rem; font-size: 38px; font-weight: 600; line-height: 1.18; letter-spacing: normal; text-wrap: balance; }
article h1 + p { max-width: 42rem; margin-top: 0; margin-bottom: 0; color: var(--muted); font-size: 15px; line-height: 1.7; }
article > .page-lead { max-width: 46rem; color: var(--muted); font-size: 16px; line-height: 1.65; }
article h2 { margin: 3rem 0 .85rem; scroll-margin-top: 88px; font-size: 26px; font-weight: 600; line-height: 1.22; letter-spacing: normal; }
article h3 { margin: 2.25rem 0 .7rem; scroll-margin-top: 88px; font-size: 21px; font-weight: 600; line-height: 1.25; letter-spacing: normal; }
article h4 { margin: 2rem 0 .6rem; scroll-margin-top: 88px; font-size: 18px; font-weight: 600; line-height: 1.3; letter-spacing: normal; }
article h5 { margin: 1.75rem 0 .55rem; scroll-margin-top: 88px; font-size: 16px; font-weight: 650; line-height: 1.4; letter-spacing: normal; }
article h6 { margin: 1.5rem 0 .5rem; scroll-margin-top: 88px; font-size: 14px; font-weight: 700; line-height: 1.45; letter-spacing: normal; }
article h1 + h2, article h2 + h3, article h3 + h4, article h4 + h5, article h5 + h6 { margin-top: 1.5rem; }
article p, article li { color: color-mix(in srgb, var(--text) 89%, var(--muted)); font-weight: 400; line-height: 1.7; }
article img, article video { max-width: 100%; height: auto; border: 0; }
.mpress-theme-image { display: contents; }
.mpress-theme-image-dark { display: none; }
html[data-theme="dark"] .mpress-theme-image-light { display: none; }
html[data-theme="dark"] .mpress-theme-image-dark { display: block; }
@media (prefers-color-scheme: dark) {
  html[data-theme="system"] .mpress-theme-image-light { display: none; }
  html[data-theme="system"] .mpress-theme-image-dark { display: block; }
}
.mpress-image-expand { position: relative; display: block; width: 100%; margin: var(--component-space) 0; padding: 0; overflow: hidden; appearance: none; border: 0; border-radius: var(--component-radius); background: transparent; box-shadow: none; color: inherit; cursor: zoom-in; text-align: left; }
.mpress-image-expand-content, .mpress-image-expand .mpress-theme-image { display: block; overflow: hidden; border-radius: inherit; }
.mpress-image-expand img { display: block; width: 100%; margin: 0; border: 0; border-radius: inherit; }
.mpress-image-expand:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
.mpress-image-lightbox { width: min(96vw, 1600px); max-width: none; height: min(94vh, 1100px); max-height: none; padding: 2.75rem 0 0; overflow: hidden; border: 0; border-radius: 0; outline: 0; background: transparent; box-shadow: none; color: var(--text); }
.mpress-image-lightbox::backdrop { background: rgb(0 0 0 / .78); backdrop-filter: blur(5px); }
.mpress-image-lightbox-content { display: grid; width: 100%; height: 100%; place-items: center; }
.mpress-image-lightbox-content .mpress-image-expand-content { display: contents; }
.mpress-image-lightbox-content img { display: block; width: auto; max-width: 100%; height: auto; max-height: calc(94vh - 3rem); margin: 0; border: 1px solid color-mix(in srgb, var(--border) 72%, transparent); border-radius: 10px; box-shadow: 0 28px 90px rgb(0 0 0 / .48); object-fit: contain; }
.mpress-image-lightbox-close { position: absolute; top: .15rem; right: .15rem; z-index: 1; display: grid; width: 2.25rem; height: 2.25rem; place-items: center; padding: 0; border: 1px solid color-mix(in srgb, var(--border) 76%, transparent); border-radius: 50%; background: color-mix(in srgb, var(--surface-solid) 88%, transparent); color: var(--text); cursor: pointer; backdrop-filter: blur(12px); }
.mpress-image-lightbox-close:hover { background: var(--accent-soft); }
.mpress-image-lightbox-close:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
@media (max-width: 760px) {
  .mpress-image-lightbox { width: calc(100vw - 1rem); height: calc(100dvh - 1rem); padding: 2.75rem 0 0; }
  .mpress-image-lightbox-content img { max-height: calc(100dvh - 3.25rem); }
}
article hr { margin: 3rem 0; border: 0; border-top: 1px solid var(--border); }
article blockquote { margin: 1.7rem 0; padding: .25rem 1.25rem; border-left: 4px solid var(--accent); color: var(--muted); font-size: 1.1rem; }
article table { width: 100%; margin: var(--component-space) 0; border-spacing: 0; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); }
article th, article td { padding: .7rem .85rem; border-bottom: 1px solid var(--border); text-align: left; }
article th { background: var(--panel); font-size: .8rem; letter-spacing: .04em; text-transform: uppercase; }
.mpress-data-table { margin: var(--component-space) 0; }
.mpress-data-table .mpress-table-scroll { overflow-x: auto; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.js .mpress-data-table:has(.mpress-table-toolbar) .mpress-table-scroll { border-radius: 0 0 var(--component-radius) var(--component-radius); }
.mpress-data-table table { min-width: 100%; margin: 0; border: 0; border-radius: 0; }
.mpress-data-table th, .mpress-data-table td { padding: .7rem .85rem; border-bottom: 1px solid var(--border); text-align: left; }
.mpress-data-table th { background: var(--panel); font-size: .8rem; letter-spacing: .04em; text-transform: uppercase; }
.mpress-data-table tbody tr:last-child td { border-bottom: 0; }
.mpress-table-toolbar { display: none; align-items: center; flex-wrap: wrap; gap: .6rem; padding: .65rem; border: 1px solid var(--border); border-bottom: 0; border-radius: var(--component-radius) var(--component-radius) 0 0; background: var(--panel); }
.js .mpress-table-toolbar { display: flex; }
.mpress-table-search { position: relative; display: flex; min-width: min(15rem, 100%); flex: 1 1 15rem; align-items: center; }
.mpress-table-search > svg { position: absolute; left: .7rem; color: var(--muted); pointer-events: none; }
.mpress-table-search input { min-height: 36px; border: 1px solid var(--border); border-radius: 5px; background: var(--surface); color: var(--text); font: 500 .82rem/1.2 ui-sans-serif, system-ui, sans-serif; }
.mpress-table-search input { width: 100%; padding: .5rem .75rem .5rem 2.1rem; }
.mpress-table-search input:focus { border-color: color-mix(in srgb, var(--accent) 58%, var(--border)); outline: 2px solid color-mix(in srgb, var(--accent) 20%, transparent); outline-offset: 1px; }
.mpress-data-table th:has(.mpress-table-heading) { padding: 0; }
.mpress-table-heading { display: flex; min-height: 42px; align-items: stretch; }
.mpress-table-heading-label { display: flex; flex: 1; align-items: center; padding: .7rem .85rem; }
.mpress-data-table [data-table-sort-column] { display: flex; min-width: 0; flex: 1; align-items: center; justify-content: space-between; gap: .5rem; padding: .7rem .85rem; border: 0; background: transparent; color: inherit; cursor: pointer; font: inherit; letter-spacing: inherit; text-align: inherit; text-transform: inherit; }
.mpress-data-table [data-table-sort-column]:hover { background: color-mix(in srgb, var(--text) 5%, transparent); color: var(--text); }
.mpress-data-table [data-table-sort-column]:focus-visible { outline: 2px solid var(--accent); outline-offset: -3px; }
.mpress-data-table [data-table-sort-column] svg { flex: 0 0 auto; color: var(--muted); transition: color .15s, transform .15s; }
.mpress-data-table th[aria-sort="ascending"] [data-table-sort-column] svg { color: var(--accent); transform: rotate(180deg); }
.mpress-data-table th[aria-sort="descending"] [data-table-sort-column] svg { color: var(--accent); }
.mpress-table-filter-trigger.utility-menu-trigger { display: none; width: 36px; min-width: 36px; place-items: center; padding: 0; border: 0; border-radius: 0; background: transparent; color: var(--muted); cursor: pointer; box-shadow: none; }
.mpress-data-table th:not(:last-child) .mpress-table-filter-trigger { border-right: 1px solid var(--border); }
.mpress-table-column-separators th:not(:last-child), .mpress-table-column-separators td:not(:last-child) { border-right: 1px solid var(--border); }
.mpress-table-column-separators th:not(:last-child) .mpress-table-filter-trigger { border-right: 0; }
.js .mpress-table-filter-trigger.utility-menu-trigger { display: grid; }
.mpress-table-filter-trigger:hover, .mpress-table-filter-trigger:focus-visible, .mpress-table-filter-trigger[aria-expanded="true"], .mpress-table-filter-trigger.active { background: color-mix(in srgb, var(--text) 6%, transparent); color: var(--accent); outline: none; }
.mpress-table-filter-menu.utility-menu-panel { min-width: 11rem; }
.mpress-table-filter-menu.utility-menu-panel button { position: relative; display: flex; width: 100%; min-height: 2.2rem; align-items: center; justify-content: space-between; gap: 1rem; padding: .52rem .75rem; border: 0; border-radius: .7rem; background: transparent; color: var(--text); cursor: pointer; font: 500 14px/1.2 ui-sans-serif, system-ui, sans-serif; text-align: left; transition: background .14s ease, color .14s ease; }
.mpress-table-filter-menu.utility-menu-panel button:hover, .mpress-table-filter-menu.utility-menu-panel button:focus-visible { background: color-mix(in srgb, var(--text) 7%, transparent); color: var(--text); outline: none; }
.mpress-table-filter-menu.utility-menu-panel button[aria-current="true"] { padding-right: 2rem; background: color-mix(in srgb, var(--accent) 11%, transparent); color: var(--accent); font-weight: 650; box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 22%, transparent); }
.mpress-table-filter-menu.utility-menu-panel button[aria-current="true"]::after { content: ""; position: absolute; top: 50%; right: .78rem; width: .45rem; height: .45rem; transform: translateY(-50%); border-radius: 999px; background: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 17%, transparent); }
.mpress-table-footer { display: none; min-height: 38px; align-items: center; justify-content: space-between; gap: .75rem; padding: .45rem .15rem 0; }
.js .mpress-table-footer { display: flex; }
.mpress-table-status { margin: 0; color: var(--muted); font-size: .78rem; line-height: 1.4; }
.mpress-table-pagination { display: flex; align-items: center; gap: .5rem; color: var(--muted); font-size: .78rem; }
.mpress-table-pagination button { display: grid; width: 32px; height: 32px; place-items: center; padding: 0; border: 1px solid var(--border); border-radius: 5px; background: var(--surface); color: var(--text); cursor: pointer; }
.mpress-table-pagination button:hover:not(:disabled), .mpress-table-pagination button:focus-visible { border-color: color-mix(in srgb, var(--accent) 45%, var(--border)); background: var(--accent-soft); color: var(--accent); outline: none; }
.mpress-table-pagination button:disabled { opacity: .38; cursor: default; }
article :not(pre) > code { padding: .12em .32em; border: 1px solid var(--border); border-radius: 3px; background: var(--panel); color: var(--accent-text); font-size: 13px; }
article h2 > code, article h3 > code, article h4 > code { padding: .05em .16em; color: inherit; font: inherit; }
pre { position: relative; overflow: auto; margin: 1.35rem 0; padding: 1rem 1.1rem; border: 1px solid var(--border); border-radius: 7px; background: var(--code); color: var(--code-text); font-size: 14px; line-height: 1.75; }
code { font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; }
.mpress-codeframe { position: relative; margin: 1.35rem 0; overflow: hidden; border: 1px solid var(--border); border-radius: 7px; background: var(--code); }
.mpress-codeframe-header { display: flex; min-height: 40px; align-items: center; border-bottom: 1px solid var(--border); background: var(--surface); box-shadow: inset 0 2px var(--accent); }
.mpress-codeframe-title { display: flex; min-width: 0; flex: none; align-items: center; padding: .55rem .7rem .55rem .85rem; color: var(--text); font: 600 .82rem/1.4 ui-sans-serif, system-ui; }
.mpress-codeframe-language { flex: none; padding: 0 0 0 .7rem; border-left: 1px solid var(--border); color: var(--muted); font: 650 .66rem/1.3 ui-monospace, monospace; letter-spacing: .04em; text-transform: uppercase; }
.mpress-codeframe-header > .mpress-codeframe-language:first-child { margin-left: .85rem; padding-left: 0; border-left: 0; }
.mpress-codeframe pre { margin: 0; padding-block: .7rem .9rem; border: 0; border-radius: 0; }
.mpress-code-line { display: block; margin-inline: -1.1rem; padding-inline: 1.1rem; }
.mpress-codeframe-line-numbers code { counter-reset: mpress-code-line; }
.mpress-codeframe-line-numbers .mpress-code-line { position: relative; padding-left: 4rem; }
.mpress-codeframe-line-numbers .mpress-code-line::before { content: counter(mpress-code-line); counter-increment: mpress-code-line; position: absolute; left: 1.1rem; width: 1.8rem; color: color-mix(in srgb, var(--code-text) 42%, transparent); font-variant-numeric: tabular-nums; text-align: right; user-select: none; }
.mpress-codeframe-line-numbers .mpress-code-line-marked::before { color: color-mix(in srgb, var(--accent) 80%, var(--code-text)); }
.mpress-codeframe-line-numbers .mpress-code-line-inserted::before { color: color-mix(in srgb, var(--success) 82%, var(--code-text)); }
.mpress-codeframe-line-numbers .mpress-code-line-deleted::before { color: color-mix(in srgb, var(--danger) 82%, var(--code-text)); }
.mpress-code-line-marked { background: color-mix(in srgb, var(--muted) 17%, transparent); }
.mpress-code-line-inserted { background: color-mix(in srgb, var(--success) 18%, transparent); box-shadow: inset 3px 0 var(--success); }
.mpress-code-line-deleted { background: color-mix(in srgb, var(--danger) 18%, transparent); box-shadow: inset 3px 0 var(--danger); }
.mpress-token-comment { color: var(--code-comment); font-style: italic; }
.mpress-token-string { color: var(--code-string); }
.mpress-token-number { color: var(--code-number); }
.mpress-token-keyword, .mpress-token-operator { color: var(--code-keyword); }
.mpress-token-function { color: var(--code-function); }
.mpress-token-type { color: var(--code-type); }

.mpress-admonition { --notice: var(--accent); position: relative; display: block; margin: var(--component-space) 0; padding: 1rem 1.15rem 1rem 1.3rem; overflow: hidden; border: 1px solid color-mix(in srgb, var(--notice) 24%, var(--border)); border-radius: var(--component-radius); background: color-mix(in srgb, var(--notice) 6%, var(--surface)); }
.mpress-admonition.tip, .mpress-admonition-tip { --notice: var(--accent-2); }
.mpress-admonition.warning, .mpress-admonition.caution, .mpress-admonition-warning, .mpress-admonition-caution { --notice: var(--warning); }
.mpress-admonition.important, .mpress-admonition-important { --notice: var(--warning); }
.mpress-admonition.danger, .mpress-admonition-danger { --notice: var(--danger); }
.mpress-container > .mpress-admonition { min-height: 100%; margin: 0; }
.mpress-admonition > strong { display: block; margin-bottom: .2rem; color: var(--notice); font-size: .79rem; letter-spacing: .07em; text-transform: uppercase; }
.mpress-admonition > :last-child { margin-bottom: 0; }
.mpress-tabs { display: block; margin: var(--component-space) 0; }
.mpress-tabs [role="tablist"] { display: flex; gap: 1.25rem; padding: 0; border-bottom: 1px solid var(--border); color: var(--muted); font-size: 13px; font-weight: 600; }
.mpress-tabs [role="tab"] { padding: .55rem .1rem; border: 0; border-bottom: 2px solid transparent; background: none; color: inherit; cursor: pointer; font: inherit; }
.mpress-tabs [aria-selected="true"] { border-bottom-color: var(--accent); color: var(--text); }
.mpress-tabs [role="tabpanel"] { padding: .8rem 0 0; }
.mpress-file-tabs { overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--panel); }
.mpress-file-tabs > [role="tablist"] { gap: 0; padding: 0 .55rem; background: color-mix(in srgb, var(--surface) 72%, transparent); }
.mpress-file-tabs > [role="tablist"] [role="tab"] { min-height: 38px; padding: .55rem .75rem .48rem; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; font-weight: 600; }
.mpress-file-tabs > [role="tabpanel"] { padding: 0; }
.mpress-file-tabs > [role="tabpanel"] > :first-child { margin-top: 0; }
.mpress-file-tabs > [role="tabpanel"] > :last-child { margin-bottom: 0; }
.mpress-file-tabs pre, .mpress-file-tabs .mpress-code-block, .mpress-file-tabs .mpress-image-expand { margin: 0; border: 0; border-radius: 0; }
.mpress-file-tabs .mpress-image-expand { background: transparent; }
.mpress-video { margin: var(--component-space) 0; }
.mpress-video video { display: block; width: 100%; height: auto; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: #000; }
.mpress-video figcaption { display: flex; flex-wrap: wrap; gap: .35rem; justify-content: space-between; margin-top: .55rem; color: var(--muted); font-size: .85rem; }
.mpress-video figcaption a { color: var(--accent); }
.mpress-card-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; margin: var(--component-space) 0; }
.mpress-cards-1 { grid-template-columns: minmax(0, 1fr); }
.mpress-card { position: relative; display: flex; flex-direction: column; min-height: 132px; padding: 1rem; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); color: var(--text); text-decoration: none; transition: border-color .15s, background .15s, transform .15s; }
.mpress-card::after { content: none; }
.mpress-card:hover { border-color: color-mix(in srgb, var(--accent) 42%, var(--border)); background: var(--accent-soft); text-decoration: none; transform: translateY(-1px); }
.mpress-card h3, .mpress-card strong { margin: 0 0 .35rem; font-size: 1.05rem; }
.mpress-link-card span { max-width: 90%; color: var(--muted); font-size: .92rem; }
.mpress-steps { margin: var(--component-space) 0; }
.mpress-file-tree { margin: 1.6rem 0; padding: 1rem 1.2rem; border: 1px solid var(--border); background: var(--panel); font-family: ui-monospace, monospace; }
.mpress-badge { display: inline-flex; align-items: center; padding: .12rem .45rem; border-radius: 3px; background: var(--accent); color: white; font-size: .75rem; font-weight: 800; }
.mpress-badge-success { background: var(--accent-2); }
.mpress-badge-warning { background: var(--warning); }
.mpress-badge-danger, .mpress-badge-error { background: var(--danger); }
.mpress-button { display: inline-flex; min-height: 40px; align-items: center; gap: .4rem; padding: .5rem .8rem; border: 1px solid var(--accent); border-radius: 6px; background: var(--accent); color: #fff; font-size: .9rem; font-weight: 700; text-decoration: none; }
.mpress-button:hover { filter: brightness(.96); text-decoration: none; }
.mpress-button-secondary { border-color: var(--border); background: var(--surface); color: var(--text); }

/* Text-first Markdown forms. The source stays prose-shaped while the
   generated controls use the same quiet surfaces as the rest of the site. */
.mpress-form { display: grid; gap: 1rem; margin: var(--component-space) 0; padding: 1.25rem; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.mpress-form > :first-child { margin-top: 0; }
.mpress-form > :last-child { margin-bottom: 0; }
.mpress-form-field { display: grid; grid-template-columns: minmax(8rem, .7fr) minmax(0, 1.3fr); gap: 1rem; align-items: center; }
.mpress-form-field > label { display: contents; }
.mpress-form-field > label > span { color: var(--text); font-size: .9rem; font-weight: 600; line-height: 1.35; }
.mpress-form-required { color: var(--accent); font-size: .9em; }
.mpress-form input, .mpress-form select, .mpress-form textarea { width: 100%; min-height: 2.5rem; padding: .55rem .7rem; border: 1px solid var(--border); border-radius: 5px; background: var(--surface-solid); color: var(--text); font: inherit; font-size: .9rem; line-height: 1.4; }
.mpress-form textarea { min-height: 6rem; resize: vertical; }
.mpress-form input:hover, .mpress-form select:hover, .mpress-form textarea:hover { border-color: color-mix(in srgb, var(--accent) 35%, var(--border)); }
.mpress-form input:focus, .mpress-form select:focus, .mpress-form textarea:focus { border-color: var(--accent); outline: 3px solid color-mix(in srgb, var(--accent) 17%, transparent); outline-offset: 0; }
.mpress-form-check { display: flex; align-items: center; }
.mpress-form-check label, .mpress-form-choices label { display: inline-flex; min-height: 2.25rem; align-items: center; gap: .6rem; color: var(--text); cursor: pointer; }
.mpress-form-check input, .mpress-form-choices input { width: 1.05rem; min-height: 1.05rem; accent-color: var(--accent); }
.mpress-form-choice-group { min-width: 0; margin: 0; padding: 0; border: 0; }
.mpress-form-choice-group legend { margin-bottom: .35rem; color: var(--text); font-size: .9rem; font-weight: 600; }
.mpress-form-choices { display: grid; gap: .15rem; }
.mpress-form-actions { display: flex; flex-wrap: wrap; gap: .6rem; justify-content: flex-end; margin-top: .35rem; padding-top: 1rem; border-top: 1px solid var(--border); }
.mpress-form-button { min-height: 2.5rem; padding: .55rem .9rem; border: 1px solid var(--border); border-radius: 5px; background: var(--surface-solid); color: var(--text); cursor: pointer; font: inherit; font-size: .9rem; font-weight: 650; }
.mpress-form-button:hover { border-color: var(--accent); background: var(--accent-soft); color: var(--text); }
.mpress-form-button-primary { border-color: var(--accent); background: var(--accent); color: #fff; }
.mpress-form-button-primary:hover { background: var(--hover); color: #fff; }

/* Progressive enhancement for generated selects. The native control remains
   in the document so form submission and component events keep normal HTML
   behavior. */
.mpress-ddlb { position: relative; display: inline-block; width: 100%; min-width: 0; vertical-align: middle; }
.mpress-ddlb-native { position: absolute !important; width: 1px !important; min-height: 1px !important; margin: 0 !important; padding: 0 !important; overflow: hidden !important; border: 0 !important; clip-path: inset(50%) !important; white-space: nowrap !important; }
.mpress-ddlb-trigger { position: relative; display: flex; width: 100%; min-height: 40px; align-items: center; justify-content: space-between; gap: .7rem; padding: .5rem .65rem; border: 1px solid var(--border); border-radius: 6px; background: color-mix(in srgb, var(--surface-solid) 76%, transparent); color: var(--text); cursor: pointer; font: inherit; font-size: .9rem; line-height: 1.35; text-align: left; box-shadow: inset 0 1px 0 rgb(255 255 255 / .05); -webkit-backdrop-filter: blur(12px); backdrop-filter: blur(12px); }
.mpress-ddlb-trigger:hover, .mpress-ddlb-trigger[aria-expanded="true"] { border-color: color-mix(in srgb, var(--accent) 46%, var(--border)); background: color-mix(in srgb, var(--surface-solid) 88%, transparent); }
.mpress-ddlb-trigger:focus-visible { outline: 2px solid color-mix(in srgb, var(--accent) 58%, transparent); outline-offset: 2px; }
.mpress-ddlb-trigger:disabled { cursor: not-allowed; opacity: .52; }
.mpress-ddlb-value { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mpress-ddlb-trigger > .lucide { flex: 0 0 auto; color: var(--muted); transition: transform .16s ease; }
.mpress-ddlb-trigger[aria-expanded="true"] > .lucide { transform: rotate(180deg); }
.mpress-ddlb-menu[popover] { position: fixed; z-index: 120; inset: auto; width: auto; max-height: min(18rem, calc(100vh - 24px)); margin: 0; padding: .35rem; overflow-y: auto; border: 1px solid color-mix(in srgb, var(--border) 84%, transparent); border-radius: 9px; background: color-mix(in srgb, var(--surface-solid) 92%, transparent); color: var(--text); box-shadow: 0 18px 48px rgb(0 0 0 / .26), 0 2px 10px rgb(0 0 0 / .16); -webkit-backdrop-filter: blur(20px) saturate(1.12); backdrop-filter: blur(20px) saturate(1.12); }
.mpress-ddlb-menu[popover]::backdrop { background: transparent; }
.mpress-ddlb-option { position: relative; display: flex; width: 100%; min-height: 36px; align-items: center; justify-content: space-between; gap: .75rem; padding: .45rem .6rem; border: 0; border-radius: 6px; outline: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: .88rem; text-align: left; }
.mpress-ddlb-option:hover, .mpress-ddlb-option:focus-visible { background: color-mix(in srgb, var(--text) 7%, transparent); color: var(--text); }
.mpress-ddlb-option[aria-selected="true"] { background: color-mix(in srgb, var(--accent) 11%, transparent); color: var(--text); }
.mpress-ddlb-option:disabled { cursor: not-allowed; opacity: .42; }
.mpress-ddlb-option .lucide { flex: 0 0 auto; opacity: 0; color: var(--accent); stroke-width: 2.5; }
.mpress-ddlb-option[aria-selected="true"] .lucide { opacity: 1; }
.mpress-audience-label .mpress-ddlb { width: min(15rem, 100%); margin-left: .35rem; }
@media (max-width: 680px) {
  .mpress-form { padding: 1rem; }
  .mpress-form-field { grid-template-columns: 1fr; gap: .35rem; }
  .mpress-form-actions { justify-content: stretch; }
  .mpress-form-button { flex: 1 1 auto; }
}

/* Component library */
.mpress-terminal { margin: 1rem 0; overflow: hidden; border: 1px solid var(--terminal-border); border-radius: 10px; background: var(--terminal-bg); color: var(--terminal-text); box-shadow: var(--terminal-shadow); text-align: left; }
.mpress-terminal-title { position: relative; display: flex; min-height: 32px; align-items: center; justify-content: center; padding: .5rem 4.5rem; border-bottom: 1px solid var(--terminal-border); background: var(--terminal-title-bg); color: var(--terminal-output); font: 500 .75rem/1 ui-sans-serif, system-ui; }
.mpress-terminal-title::before { content: ""; position: absolute; left: 14px; width: 10px; height: 10px; border-radius: 50%; background: #ff665c; box-shadow: 18px 0 #ffbd44, 36px 0 #00ca4e; }
.mpress-terminal-windows .mpress-terminal-title::before { left: auto; right: 52px; border-radius: 0; background: transparent; box-shadow: 16px 0 0 -4px #aeb8c9, 36px 0 0 -4px #aeb8c9; }
.mpress-terminal-plain .mpress-terminal-title::before { display: none; }
.mpress-terminal pre { margin: 0; min-height: 0; padding: .875rem 1rem; border: 0; border-radius: 0; background: transparent; color: var(--terminal-text); }
.mpress-terminal code { display: block; }
.mpress-terminal code span { white-space: pre-wrap; }
.mpress-prompt { color: var(--terminal-prompt); user-select: none; }
.mpress-cmd { color: var(--terminal-command); }
.mpress-comment { color: var(--terminal-comment); font-style: italic; }
.mpress-output { color: var(--terminal-output); font-style: italic; }
.mpress-terminal .mpress-copy { top: 3px; width: 26px; min-height: 24px; padding: 0; border-color: var(--terminal-copy-border); color: var(--terminal-copy); }
.mpress-terminal .mpress-copy:hover { border-color: var(--terminal-copy); color: var(--terminal-copy-hover); }
.mpress-copy { position: absolute; right: .55rem; top: .28rem; display: inline-grid; width: 32px; min-height: 32px; padding: 0; place-items: center; border: 1px solid var(--code-copy-border); border-radius: 4px; background: transparent; color: var(--code-copy); cursor: pointer; }
.mpress-copy .lucide { grid-area: 1 / 1; width: 15px; height: 15px; pointer-events: none; }
.mpress-copy .lucide-check { display: none; }
.mpress-copy.is-copied .lucide-copy { display: none; }
.mpress-copy.is-copied .lucide-check { display: block; }
.mpress-codeframe-header > .mpress-copy { position: static; height: 28px; width: 28px; min-height: 28px; margin: 0 .4rem 0 auto; background: var(--surface); line-height: 1; transform: translateY(-1px); }
.mpress-copy:hover { border-color: var(--code-copy); color: var(--code-copy-hover); }
.mpress-copy.is-copied, .mpress-copy.is-copied:hover, .mpress-terminal .mpress-copy.is-copied:hover { border-color: color-mix(in srgb, #22c981 55%, var(--code-copy-border)); color: #22c981; }

.mpress-admonition-titlebar { display: flex; align-items: center; gap: .5rem; margin: 0 0 .35rem; color: var(--notice); font-size: .79rem; font-weight: 800; letter-spacing: .07em; text-transform: uppercase; }
.mpress-admonition-body > :first-child { margin-top: 0; }
.mpress-admonition-body > :last-child { margin-bottom: 0; }
.mpress-details { margin: var(--component-space) 0; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.mpress-details summary { display: flex; min-height: 48px; align-items: center; gap: .6rem; padding: .7rem 1rem; cursor: pointer; font-weight: 680; list-style: none; }
.mpress-details summary::-webkit-details-marker { display: none; }
.mpress-disclosure-icon { display: inline-grid; color: var(--muted); }
.mpress-disclosure-icon .lucide { grid-area: 1 / 1; }
.mpress-disclosure-icon .lucide-chevron-down { display: none; }
details[open] > summary .mpress-disclosure-icon .lucide-chevron-right { display: none; }
details[open] > summary .mpress-disclosure-icon .lucide-chevron-down { display: block; }
.mpress-details[open] summary { border-bottom: 1px solid var(--border); }
.mpress-details-content { padding: 1rem; }
.mpress-details-content > :first-child { margin-top: 0; }
.mpress-details-content > :last-child { margin-bottom: 0; }
.mpress-details-content > .mpress-admonition { margin: 1rem 0 0; }
.mpress-details-content > .mpress-admonition:first-child { margin-top: 0; }

.mpress-cards-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.mpress-card-icon { margin-bottom: .65rem; color: var(--accent); font-size: 1.25rem; }
.mpress-card-title { color: var(--text); font-weight: 760; }
.mpress-card-desc { margin-top: .35rem; color: var(--muted); font-size: .92rem; line-height: 1.5; }
.mpress-linkcard { display: flex; align-items: center; gap: .9rem; margin: 1rem 0; padding: 1rem; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); color: var(--text); text-decoration: none; transition: border-color .15s, background .15s; }
.mpress-linkcard:hover { border-color: color-mix(in srgb, var(--accent) 45%, var(--border)); text-decoration: none; }
.mpress-linkcard-icon { color: var(--accent); font-size: 1.25rem; }
.mpress-linkcard-body { flex: 1; min-width: 0; }
.mpress-linkcard-title { display: block; font-weight: 760; }
.mpress-linkcard-desc { display: block; color: var(--muted); font-size: .9rem; }
.mpress-linkcard-arrow { color: var(--muted); }

.mpress-timeline { position: relative; }
.mpress-timeline-entry { position: relative; display: grid; grid-template-columns: 2.1rem minmax(0, 1fr); gap: .8rem; padding-bottom: 1.4rem; }
.mpress-timeline-entry::before { position: absolute; top: 2rem; bottom: 0; left: 1rem; border-left: 1px solid var(--border); content: ""; }
.mpress-timeline-entry:last-child::before { display: none; }
.mpress-timeline-marker, .mpress-step-num { z-index: 1; display: grid; width: 2.1rem; height: 2.1rem; box-sizing: border-box; place-items: center; border: 1px solid var(--border); border-radius: 50%; background: var(--surface); color: var(--accent); font: 800 .78rem/1 ui-monospace, "SFMono-Regular", Consolas, monospace; font-variant-numeric: tabular-nums; }
.mpress-timeline-body { min-width: 0; }
.mpress-step-title { display: flex; min-height: 2.1rem; align-items: center; margin: 0 0 .25rem; font-size: 21px; font-weight: 760; line-height: 1.25; letter-spacing: normal; }
.mpress-step-content { padding-top: .75rem; }
.mpress-step-content > :first-child { margin-top: 0; }
.mpress-filetree { margin: var(--component-space) 0; padding: .75rem 0; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); font: .88rem/1.65 ui-monospace, monospace; }
.mpress-filetree ul { margin: 0; padding: 0; list-style: none; }
.mpress-filetree > ul { padding-left: .9rem; }
.mpress-filetree ul ul { padding-left: 1.6rem; }
.mpress-filetree li { position: relative; }
.mpress-filetree li > ul { position: relative; }
.mpress-filetree li > ul::before { content: ""; position: absolute; top: -.8rem; left: .9rem; height: .8rem; border-left: 1px solid var(--border); }
.mpress-filetree ul ul > li::before { content: ""; position: absolute; top: 0; bottom: 0; left: -.7rem; border-left: 1px solid var(--border); }
.mpress-filetree ul ul > li::after { content: ""; position: absolute; top: .8rem; left: -.7rem; width: 1.15rem; border-top: 1px solid var(--border); }
.mpress-filetree ul ul > li:last-child::before { bottom: auto; height: .8rem; }
.mpress-filetree-row { position: relative; z-index: 1; display: flex; align-items: center; gap: .5rem; min-height: 1.6rem; padding: .05rem 1rem .05rem .45rem; }
.mpress-filetree-row:hover { background: var(--accent-soft); }
.mpress-filetree-icon { width: 1.2rem; color: var(--muted); }
.mpress-filetree-dir .mpress-filetree-name { font-weight: 750; }
.mpress-filetree-desc { margin-left: auto; color: var(--muted); font: .78rem/1.5 ui-sans-serif, system-ui; }

.mpress-api-endpoint { margin: var(--component-space) 0; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.mpress-api-header { display: flex; align-items: center; gap: .7rem; padding: .75rem 1rem; border-bottom: 1px solid var(--border); background: var(--panel); }
.mpress-api-method { padding: .17rem .42rem; border-radius: 4px; background: var(--accent-2); color: white; font: 800 .72rem/1.4 ui-monospace, monospace; }
.mpress-api-path { overflow-wrap: anywhere; font: 650 .88rem/1.5 ui-monospace, monospace; }
.mpress-api-description { padding: 1rem; }
.mpress-api-description > :first-child { margin-top: 0; }
.mpress-api-description > :last-child { margin-bottom: 0; }
.mpress-api-playground .mpress-api-description { padding-bottom: 0; }
.mpress-diff { margin: var(--component-space) 0; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--code); }
.mpress-diff-title { padding: .5rem .75rem; border-bottom: 1px solid var(--border); background: var(--panel); color: var(--muted); font: 600 .75rem/1.4 ui-monospace, monospace; }
.mpress-diff-body { display: grid; }
.mpress-diff-split .mpress-diff-body { grid-template-columns: 1fr 1fr; }
.mpress-diff-side-by-side .mpress-diff-body { grid-template-columns: 1fr 1fr; }
.mpress-diff-pane + .mpress-diff-pane { border-left: 1px solid var(--border); }
.mpress-diff-line { display: block; min-height: 1.5em; padding: .08rem .85rem; color: var(--code-text); font: .84rem/1.55 ui-monospace, monospace; white-space: pre-wrap; }
.mpress-diff pre { margin: 0; padding: .7rem 0; border: 0; border-radius: 0; background: transparent; }
.mpress-diff-added { background: var(--code-added-bg); color: var(--code-added-text); }
.mpress-diff-removed { background: var(--code-removed-bg); color: var(--code-removed-text); }

.mpress-matrix-wrapper { margin: 1.6rem 0; overflow-x: auto; }
.mpress-matrix { width: 100%; margin: 0; }
.mpress-matrix-check, .mpress-matrix-cross, .mpress-matrix-partial, .mpress-matrix-empty { display: inline-flex; align-items: center; justify-content: center; vertical-align: middle; }
.mpress-matrix-check { color: var(--accent-2); }
.mpress-matrix-cross { color: var(--danger); }
.mpress-matrix-partial { color: var(--warning); }
.mpress-calendar { margin: 1.6rem 0; overflow: hidden; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); }
.mpress-calendar-header { display: flex; align-items: center; justify-content: space-between; padding: .75rem; border-bottom: 1px solid var(--border); }
.mpress-calendar-title { font-weight: 760; }
.mpress-calendar-nav { display: grid; width: 40px; height: 40px; place-items: center; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--text); cursor: pointer; }
.mpress-calendar-grid { display: grid; grid-template-columns: repeat(7, 1fr); }
.mpress-calendar-day-header, .mpress-calendar-cell { min-height: 3.5rem; padding: .45rem; border-right: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.mpress-calendar-day-header { min-height: auto; background: var(--panel); color: var(--muted); font-size: .7rem; font-weight: 800; text-align: center; text-transform: uppercase; }
.mpress-calendar-cell.today .mpress-calendar-date { display: inline-grid; width: 1.7rem; height: 1.7rem; place-items: center; border-radius: 50%; background: var(--accent); color: #fff; }
.mpress-calendar-compact .mpress-calendar-cell { min-height: 2.3rem; text-align: center; }
.mpress-calendar-content { padding: .2rem 1rem .8rem; color: var(--muted); font-size: .88rem; }

.mpress-changelog { margin: 2rem 0; }
.mpress-changelog-entry { margin: 0; padding: 0 0 2.75rem; }
.mpress-changelog-entry:last-child { padding-bottom: 0; }
.mpress-changelog-body { margin-left: 2.9rem; }
.mpress-changelog-header { display: flex; flex-wrap: wrap; align-items: baseline; gap: .6rem; }
.mpress-changelog-header { min-height: 2.1rem; align-items: center; justify-content: space-between; margin: 0 0 1.15rem; padding: 0 0 .85rem; border-bottom: 1px solid var(--border); }
.mpress-changelog-version { font-weight: 780; }
.mpress-changelog-version { color: var(--text); font-size: 1.14rem; letter-spacing: -.018em; }
.mpress-changelog-date { color: var(--muted); font-size: .8rem; font-variant-numeric: tabular-nums; }
.mpress-changelog-content { display: grid; gap: 1.15rem; }
.mpress-changelog-section { --changelog-kind: var(--accent); display: grid; grid-template-columns: minmax(6rem, 8rem) minmax(0, 1fr); align-items: start; gap: 1.2rem; }
.mpress-changelog-category { display: flex; align-items: center; gap: .45rem; margin: .16rem 0 0; color: var(--changelog-kind); font-size: .72rem; font-weight: 760; letter-spacing: .055em; line-height: 1.45; text-transform: uppercase; }
.mpress-changelog-category > span { width: .42rem; height: .42rem; flex: 0 0 auto; border-radius: 50%; background: currentColor; }
.mpress-changelog-category-visually-hidden { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
.mpress-changelog-items { display: grid; gap: .48rem; margin: 0; padding: 0; list-style: none; }
.mpress-changelog-items li { position: relative; margin: 0; padding-left: 1rem; color: var(--text); line-height: 1.55; }
.mpress-changelog-items li::before { position: absolute; top: .76em; left: 0; width: .38rem; height: 1px; background: color-mix(in srgb, var(--changelog-kind) 72%, var(--muted)); content: ""; }
.mpress-badge-feature { --changelog-kind: #2aa775; }
.mpress-badge-fix { --changelog-kind: #4f86e8; }
.mpress-badge-breaking { --changelog-kind: #df5f6b; }
.mpress-badge-improvement { --changelog-kind: #9a70dc; }
.mpress-badge-docs { --changelog-kind: #c9813c; }
.mpress-changelog-category { padding: 0; border-radius: 0; background: transparent; }
@media (max-width: 560px) { .mpress-changelog-header { align-items: flex-start; flex-direction: column; gap: .2rem; } .mpress-changelog-body { margin-left: 2.9rem; } .mpress-changelog-section { grid-template-columns: 1fr; gap: .45rem; } .mpress-changelog-category { margin-top: 0; } }
.mpress-release { --release-kind: var(--accent); margin: var(--component-space) 0; padding: 1.25rem; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.mpress-release-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1.25rem; }
.mpress-release-title-group { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; gap: .55rem .75rem; }
.mpress-release-type { padding: .15rem .55rem; border: 1px solid color-mix(in srgb, var(--accent) 58%, var(--border)); border-radius: 999px; background: color-mix(in srgb, var(--accent) 13%, var(--surface)); color: color-mix(in srgb, var(--accent) 72%, var(--text)); font-size: .69rem; font-weight: 650; line-height: 1.45; white-space: nowrap; }
.mpress-release-version { display: flex; min-width: 0; flex-wrap: wrap; gap: .28rem .42rem; margin: 0; color: var(--text); font-size: 1.08rem; font-weight: 720; letter-spacing: -.02em; line-height: 1.4; }
.mpress-release-title { color: var(--text); font-weight: 650; }
.mpress-release-title::before { margin-right: .42rem; color: var(--muted); content: ":"; }
.mpress-release-date { flex: 0 0 auto; padding-top: .22rem; color: var(--muted); font-size: .73rem; font-variant-numeric: tabular-nums; white-space: nowrap; }
.mpress-release-notes { margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--border); }
.mpress-release-section { padding: 0; }
.mpress-release-section + .mpress-release-section { margin-top: 1.15rem; }
.mpress-release-section-title { margin: 0 0 .45rem; color: var(--text); font-size: .84rem; font-weight: 720; letter-spacing: -.005em; line-height: 1.45; text-transform: none; }
.mpress-release-items { display: grid; gap: .36rem; margin: 0; padding-left: 1.15rem; }
.mpress-release-items li { margin: 0; padding-left: .12rem; color: var(--text); line-height: 1.55; }
.mpress-release-items li::marker { color: var(--muted); }
.mpress-release-assets { margin-top: 1.2rem !important; padding-top: 1rem; border-top: 1px solid var(--border); }
.mpress-release-assets .mpress-release-items { gap: .2rem; padding: 0; list-style: none; }
.mpress-release-assets .mpress-release-items li { display: flex; align-items: center; gap: .55rem; padding: .35rem .45rem; border-radius: 4px; color: var(--muted); }
.mpress-release-assets .mpress-release-items li:hover { background: var(--panel); color: var(--text); }
.mpress-release-asset-icon { display: inline-flex; flex: 0 0 auto; }
.mpress-release-assets a { color: inherit; text-decoration: none; }
.mpress-release-assets a:hover { color: var(--accent); text-decoration: none; }
@media (max-width: 620px) { .mpress-release-header { flex-direction: column; gap: .55rem; } .mpress-release-date { padding-top: 0; } }

.mpress-status { display: inline-flex; align-items: center; gap: .45rem; margin-right: .6rem; padding: .3rem .55rem; border: 1px solid var(--border); border-radius: 999px; background: var(--surface); font-size: .82rem; }
.mpress-status-service { font-weight: 680; }
.mpress-status-dot { width: .62rem; height: .62rem; border-radius: 50%; background: var(--accent-2); }
.mpress-status-degraded .mpress-status-dot { background: var(--warning); }
.mpress-status-outage .mpress-status-dot { background: var(--danger); }
.mpress-status-label { color: var(--muted); }
.mpress-status-desc { color: var(--muted); font-size: .83rem; }
.mpress-status-link { font-size: .82rem; }

.mpress-pricing { display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: .8rem; margin: 1.6rem 0; }
.mpress-pricing-card { position: relative; padding: 1.2rem; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); text-align: center; }
.mpress-pricing-card.featured { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
.mpress-pricing-badge { position: absolute; top: -.65rem; left: 50%; padding: .12rem .45rem; border-radius: 3px; background: var(--accent); color: #fff; font-size: .68rem; font-weight: 800; transform: translateX(-50%); }
.mpress-pricing-name { font-weight: 800; }
.mpress-pricing-price { margin: .4rem 0 1rem; font-size: 1.65rem; font-weight: 850; letter-spacing: -.04em; }
.mpress-pricing-features { margin: 0; padding: 0; list-style: none; font-size: .88rem; }
.mpress-pricing-feature { margin: .4rem 0; }
.mpress-pricing-check, .mpress-pricing-cross, .mpress-pricing-partial { display: inline-flex; margin-right: .25rem; align-items: center; vertical-align: text-bottom; }
.mpress-pricing-check { color: var(--accent-2); }
.mpress-pricing-cross { color: var(--danger); }
.mpress-pricing-partial { color: var(--warning); }
.mpress-pricing-cta { display: flex; width: 100%; min-height: 40px; margin-top: 1rem; padding: .5rem .75rem; align-items: center; justify-content: center; border: 1px solid var(--border); border-radius: 5px; color: var(--text); font-weight: 720; text-decoration: none; }
.mpress-pricing-cta:hover { border-color: var(--accent); color: var(--accent); text-decoration: none; }
.mpress-pricing-cta-primary { border-color: var(--accent); background: var(--accent); color: #fff; }
.mpress-pricing-cta-primary:hover { color: #fff; filter: brightness(.96); }

.mpress-testimonials { margin: 1.6rem 0; padding: 1.4rem; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); text-align: center; }
.mpress-testimonial { display: none; margin: 0; padding: 0; border: 0; }
.mpress-testimonial.active { display: block; }
.mpress-testimonial-quote { margin: 0 auto .8rem; max-width: 42rem; color: var(--text); font-size: 1.18rem; }
.mpress-testimonial-author cite { font-style: normal; font-weight: 750; }
.mpress-testimonial-company { display: block; color: var(--muted); font-size: .83rem; }
.mpress-testimonials-nav { display: flex; justify-content: center; gap: .4rem; margin-top: 1rem; }
.mpress-testimonials-dot { display: grid; width: 32px; height: 32px; padding: 0; place-items: center; border: 0; background: transparent; cursor: pointer; }
.mpress-testimonials-dot::before { content: ""; width: .6rem; height: .6rem; border-radius: 50%; background: var(--border); }
.mpress-testimonials-dot[aria-current="true"]::before { background: var(--accent); }

.mpress-carousel { margin: 1.6rem 0; overflow: hidden; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
.mpress-carousel-viewport { min-height: 13rem; }
.mpress-carousel-slide { min-height: inherit; padding: clamp(1.25rem, 3vw, 2rem); }
.mpress-carousel-slide[hidden] { display: none; }
.mpress-carousel-slide > :first-child { margin-top: 0; }
.mpress-carousel-slide > :last-child { margin-bottom: 0; }
.mpress-carousel-controls { display: flex; min-height: 48px; align-items: center; gap: .55rem; padding: .4rem .55rem; border-top: 1px solid var(--border); background: color-mix(in srgb, var(--panel) 62%, transparent); }
.mpress-carousel-controls > button { display: grid; width: 34px; height: 34px; flex: 0 0 auto; place-items: center; padding: 0; border: 0; border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; }
.mpress-carousel-controls > button:hover, .mpress-carousel-controls > button:focus-visible { background: var(--surface); color: var(--text); outline: none; }
.mpress-carousel-controls > span { min-width: 3.2rem; color: var(--muted); font: 650 .7rem/1 ui-monospace, monospace; text-align: center; }
.mpress-carousel-dots { display: flex; flex: 1; align-items: center; justify-content: center; gap: .35rem; }
.mpress-carousel-dots button { display: grid; width: 28px; height: 28px; place-items: center; padding: 0; border: 0; background: transparent; cursor: pointer; }
.mpress-carousel-dots button::before { content: ""; width: 6px; height: 6px; border-radius: 999px; background: var(--border); transition: width .15s ease, background .15s ease; }
.mpress-carousel-dots button[aria-current="true"]::before { width: 18px; background: var(--accent); }

.mpress-tutorial { margin: 1.6rem 0; overflow: hidden; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); }
.mpress-tutorial-header { padding: 1rem; border-bottom: 1px solid var(--border); }
.mpress-tutorial-title { margin: 0 0 .65rem; }
.mpress-tutorial-progress { height: 5px; overflow: hidden; border-radius: 5px; background: var(--panel); }
.mpress-tutorial-progress-bar { height: 100%; background: var(--accent); transition: width .2s; }
.mpress-tutorial-count { display: block; margin-top: .35rem; color: var(--muted); font-size: .75rem; }
.mpress-tutorial-step { padding: 1rem; border-bottom: 1px solid var(--border); }
.mpress-tutorial-step-header { display: flex; align-items: center; gap: .55rem; }
.mpress-tutorial-check input { position: absolute; opacity: 0; }
.mpress-tutorial-check { display: grid; width: 32px; height: 32px; place-items: center; cursor: pointer; }
.mpress-tutorial-checkmark { display: grid; width: 1.25rem; height: 1.25rem; place-items: center; border: 1px solid var(--border); border-radius: 4px; cursor: pointer; }
.mpress-tutorial-check input:checked + .mpress-tutorial-checkmark { border-color: var(--accent-2); background: transparent; }
.mpress-tutorial-checkmark .lucide { width: 13px; height: 13px; opacity: 0; color: var(--accent-2); stroke-width: 3; }
.mpress-tutorial-check input:checked + .mpress-tutorial-checkmark .lucide { opacity: 1; }
.mpress-tutorial-step-num { min-width: 1.15rem; color: var(--muted); font-size: .875rem; font-variant-numeric: tabular-nums; font-weight: 750; line-height: 1; text-align: center; }
.mpress-tutorial-step-title { font-weight: 750; }
.mpress-tutorial-step-content { padding-left: 3rem; }
.mpress-tutorial-step.completed { opacity: .67; }
.mpress-tutorial-footer { padding: .65rem 1rem; text-align: right; }
.mpress-tutorial-reset { min-height: 36px; padding: .4rem .6rem; border: 0; border-radius: 5px; background: none; color: var(--muted); cursor: pointer; font-size: .78rem; }
.mpress-tutorial-reset:hover { background: var(--accent-soft); color: var(--text); }

.mpress-audience-selector { margin: 0 0 1rem; padding: .6rem .75rem; border: 1px solid var(--border); border-radius: 7px; background: var(--panel); }
.mpress-audience-label { font-size: .82rem; font-weight: 700; }
.mpress-audience-select { margin-left: .35rem; padding: .25rem .45rem; border: 1px solid var(--border); border-radius: 4px; background: var(--surface); color: var(--text); }
.mpress-audience { padding: .2rem 1rem; border-left: 2px solid var(--accent); background: var(--accent-soft); }
.mpress-audience > .mpress-admonition { margin: 2rem 0; }
.mpress-reactive-input { display: inline-block; min-width: 180px; margin: .5rem .5rem .5rem 0; }
.mpress-reactive-label { display: grid; gap: .3rem; color: var(--muted); font-size: .78rem; font-weight: 700; }
.mpress-reactive-control { min-width: 0; min-height: 40px; padding: .5rem .6rem; border: 1px solid var(--border); border-radius: 5px; background: var(--surface); color: var(--text); }
.mpress-reactive-range-wrap { display: flex; align-items: center; gap: .6rem; }
.mpress-reactive-range-value { min-width: 2ch; color: var(--text); }
.mpress-reactive-computed { display: inline-flex; gap: .25rem; margin: .5rem 0; padding: .55rem .7rem; border: 1px solid var(--border); border-radius: 5px; background: var(--panel); }
.mpress-reactive-computed-label { color: var(--muted); }
.mpress-reactive-computed-value { color: var(--accent); font-weight: 800; }
.mpress-explained { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(220px, .8fr); margin: var(--component-space) 0; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--code); }
.mpress-explained-code { min-width: 0; overflow: auto; }
.mpress-explained-code pre { height: 100%; margin: 0; border: 0; border-radius: 0; }
.mpress-explained-line { display: block; min-height: 1.75em; margin: 0 -1.1rem; padding: 0 1.1rem; cursor: default; white-space: pre; }
.mpress-explained-line[data-ref] { position: relative; padding-right: 3.2rem; cursor: help; transition: background-color .16s ease, color .16s ease; }
.mpress-explained-line[data-ref]::after { content: attr(data-ref); position: absolute; top: .14rem; right: 1.1rem; display: grid; width: 1.2rem; height: 1.2rem; place-items: center; border: 1px solid color-mix(in srgb, var(--accent) 46%, var(--border)); border-radius: 50%; background: color-mix(in srgb, var(--accent) 14%, var(--code)); color: color-mix(in srgb, var(--accent) 78%, var(--code-text)); font: 750 .66rem/1 ui-sans-serif, system-ui, sans-serif; }
.mpress-explained-line[data-ref]:focus-visible { outline: 0; background: var(--accent-soft); box-shadow: inset 3px 0 var(--accent); }
.mpress-explained-prose { display: grid; align-content: center; gap: .55rem; padding: 1rem; border-left: 1px solid var(--border); background: var(--surface); }
.mpress-explained-prose p { margin: 0; }
.mpress-explained-para { display: flex; align-items: flex-start; gap: .55rem; padding: .5rem; border-radius: 5px; }
.mpress-explained-copy { min-width: 0; }
.mpress-explained-active { background: var(--accent-soft); }
.mpress-explained-ref { display: inline-grid; width: 1.25rem; height: 1.25rem; flex: 0 0 auto; place-items: center; border-radius: 50%; background: var(--accent); color: #fff; font-size: .68rem; font-weight: 800; }
.mpress-explained-enhanced { display: block; overflow: hidden; }
.mpress-explained-enhanced .mpress-explained-code { border-radius: inherit; }
.mpress-explained-enhanced .mpress-explained-prose { position: absolute; width: 1px; height: 1px; margin: -1px; padding: 0; overflow: hidden; border: 0; clip-path: inset(50%); white-space: nowrap; }
.mpress-explained-tooltip { position: fixed; z-index: 1001; display: flex; width: min(21rem, calc(100vw - 1.5rem)); align-items: flex-start; gap: .65rem; padding: .8rem .9rem; border: 1px solid color-mix(in srgb, var(--border) 78%, transparent); border-radius: 9px; background: color-mix(in srgb, var(--surface-solid) 90%, transparent); color: var(--text); box-shadow: 0 18px 50px rgb(0 0 0 / .24), inset 0 1px rgb(255 255 255 / .06); opacity: 0; transform: translateY(5px) scale(.985); transition: opacity .14s ease, transform .14s ease; -webkit-backdrop-filter: blur(18px) saturate(1.25); backdrop-filter: blur(18px) saturate(1.25); pointer-events: none; }
.mpress-explained-tooltip[hidden] { display: none; }
.mpress-explained-tooltip-visible { opacity: 1; transform: translateY(0) scale(1); }
.mpress-explained-tooltip .mpress-explained-ref { margin-top: .08rem; box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 12%, transparent); }
.mpress-explained-tooltip-copy { color: color-mix(in srgb, var(--text) 92%, var(--muted)); font-size: .84rem; line-height: 1.55; }
.mpress-qr { display: inline-grid; gap: .5rem; margin: 1rem 0; padding: .75rem; border: 1px solid var(--border); border-radius: 8px; background: var(--qr-bg); text-align: center; }
.mpress-qr img { border: 0; }
.mpress-qr svg { display: block; }
.mpress-qr svg > rect:first-child { fill: var(--qr-bg); }
.mpress-qr svg > rect:not(:first-child) { fill: var(--qr-foreground); }
.mpress-qr-label { color: var(--qr-foreground); font-size: .78rem; font-weight: 700; }

.mpress-api-playground { margin: var(--component-space) 0; overflow: hidden; border: 1px solid var(--border); border-radius: var(--component-radius); background: var(--surface); }
.mpress-api-pg-body, .mpress-api-pg-response { padding: 1rem; }
.mpress-api-pg-url, .mpress-api-pg-server { display: grid; gap: .35rem; }
.mpress-api-pg-section { margin-top: .85rem; border-top: 1px solid var(--border); }
.mpress-api-pg-section > summary { display: flex; min-height: 44px; align-items: center; gap: .5rem; cursor: pointer; font-weight: 680; list-style: none; }
.mpress-api-pg-section > summary::-webkit-details-marker { display: none; }
.mpress-api-pg-headers { display: grid; gap: .5rem; padding-bottom: .75rem; }
.mpress-api-pg-header-row { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: .5rem; }
.mpress-api-pg-server-select, .mpress-api-pg-url-input, .mpress-api-pg-request, .mpress-api-pg-headers input { width: 100%; min-height: 40px; padding: .55rem .65rem; border: 1px solid var(--border); border-radius: 5px; background: var(--surface); color: var(--text); }
.mpress-api-pg-request { min-height: 100px; resize: vertical; font-family: ui-monospace, monospace; }
.mpress-api-pg-actions { display: flex; align-items: center; gap: .65rem; margin-top: .9rem; }
.mpress-api-pg-send { min-height: 40px; padding: .5rem .8rem; border: 0; border-radius: 5px; background: var(--accent); color: #fff; cursor: pointer; font-weight: 730; }
.mpress-api-pg-response-header { display: flex; gap: .6rem; color: var(--muted); font-size: .78rem; }
.mpress-api-pg-response-body { overflow: auto; min-height: 70px; margin-top: .5rem; padding: .7rem; border-radius: 5px; background: var(--code); color: var(--code-text); font: .8rem/1.55 ui-monospace, monospace; }
.mpress-unsupported { display: inline-block; padding: .15rem .4rem; border: 1px dashed #ef4444; border-radius: 5px; background: rgba(239,68,68,.1); color: #ef4444; }

/* Full-width HTML landing pages. The documentation shell remains unchanged. */
.landing-main { width: 100%; padding: 0; overflow: hidden; }
.landing-main > h1, .landing-main > h2, .landing-main > h3, .landing-main > p { margin-top: 0; }
.landing-main a { font-weight: 700; }
.mpress-site-banner { padding: .55rem 1rem; border: 0; background: var(--accent); color: #fff; text-align: center; font-size: .9rem; font-weight: 650; }
.mpress-site-banner a { color: inherit; }
.mpress-frontmatter-hero { position: relative; min-height: 560px; border-bottom: 1px solid var(--border); }
.mpress-frontmatter-hero-inner { display: flex; width: min(920px, calc(100% - 3rem)); min-height: inherit; margin: 0 auto; padding: 5rem 0; flex-direction: column; align-items: center; justify-content: center; text-align: center; }
.mpress-frontmatter-hero-image { display: block; width: 150px; height: 118px; margin-bottom: 1.4rem; }
.mpress-frontmatter-hero-image img { width: 100%; height: 100%; object-fit: contain; }
.hero-logo-dark { display: none; }
html[data-theme="dark"] .hero-logo-light { display: none; }
html[data-theme="dark"] .hero-logo-dark { display: block; }
@media (prefers-color-scheme: dark) { html[data-theme="system"] .hero-logo-light { display: none; } html[data-theme="system"] .hero-logo-dark { display: block; } }
.mpress-frontmatter-hero-title { max-width: 900px; margin: 0 auto 1rem; font-size: clamp(3rem, 7vw, 6rem); line-height: .98; letter-spacing: -.055em; text-wrap: balance; }
.mpress-frontmatter-hero-tagline { max-width: 760px; margin: 0 auto; color: var(--muted); font-size: clamp(1.05rem, 1.8vw, 1.3rem); line-height: 1.55; }
.mpress-frontmatter-hero-actions { display: flex; flex-wrap: wrap; justify-content: center; gap: .8rem; margin-top: 2rem; }
.mpress-frontmatter-hero-action { display: inline-flex; min-height: 46px; align-items: center; gap: .55rem; padding: .65rem 1rem; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); color: var(--text); text-decoration: none; }
.mpress-frontmatter-hero-action:hover { border-color: var(--accent); text-decoration: none; }
.mpress-frontmatter-hero-action-primary { border-color: var(--accent); background: var(--accent); color: #fff; }
.splash-hero { position: relative; border-bottom: 1px solid var(--border); background: var(--surface); }
.splash-hero::before { content: ""; position: absolute; inset: 0; pointer-events: none; background-image: linear-gradient(var(--border) 1px, transparent 1px), linear-gradient(90deg, var(--border) 1px, transparent 1px); background-size: 40px 40px; opacity: .23; mask-image: linear-gradient(to bottom, #000, transparent 85%); }
.splash-hero-inner { position: relative; display: grid; grid-template-columns: minmax(0, 1.05fr) minmax(360px, .95fr); gap: clamp(3rem, 7vw, 7rem); align-items: center; width: min(1180px, calc(100% - 3rem)); min-height: 660px; margin: 0 auto; padding: 6rem 0; }
.splash-eyebrow { display: flex; align-items: center; gap: .65rem; margin-bottom: 1.4rem; color: var(--accent-2); font: 800 .76rem/1.2 ui-monospace, monospace; letter-spacing: .12em; text-transform: uppercase; }
.splash-eyebrow::before { content: ""; width: 28px; border-top: 2px solid currentColor; }
.splash-hero h1 { max-width: 9ch; margin-bottom: 1.4rem; font-size: clamp(3.4rem, 7vw, 6.3rem); line-height: .93; letter-spacing: -.07em; text-wrap: balance; }
.splash-lede { max-width: 590px; color: var(--muted); font-size: clamp(1.08rem, 1.7vw, 1.32rem); line-height: 1.55; }
.splash-actions { display: flex; flex-wrap: wrap; gap: .75rem; margin-top: 2rem; }
.splash-action { display: inline-flex; min-height: 46px; align-items: center; padding: .65rem 1rem; border: 1px solid var(--accent); border-radius: 5px; background: var(--accent); color: #fff; text-decoration: none; }
.splash-action:hover { filter: brightness(.96); text-decoration: none; }
.splash-action-secondary { border-color: var(--border); background: var(--surface); color: var(--text); }
.splash-action-secondary:hover { border-color: var(--accent); background: var(--accent-soft); }
.splash-hero .splash-proof { margin: 1.4rem 0 0; color: var(--muted); font: .8rem/1.5 ui-monospace, monospace; }
.splash-terminal { overflow: hidden; border: 1px solid #2b3546; border-radius: 10px; background: #090d15; color: #dce5f2; box-shadow: 0 28px 70px rgba(10, 17, 30, .2); transform: rotate(.5deg); }
.splash-terminal-bar { display: flex; min-height: 42px; align-items: center; gap: 7px; padding: 0 15px; border-bottom: 1px solid #273142; background: #161d2a; color: #8995a8; font: .72rem/1 ui-monospace, monospace; }
.splash-terminal-bar::before { content: ""; width: 9px; height: 9px; border-radius: 50%; background: #ff665c; box-shadow: 16px 0 #ffbd44, 32px 0 #28c840; margin-right: 36px; }
.splash-terminal pre { min-height: 290px; margin: 0; padding: 1.6rem; border: 0; border-radius: 0; background: transparent; font-size: .88rem; line-height: 1.75; }
.splash-terminal .prompt { color: #65d8c4; }
.splash-terminal .result { color: #8390a3; }
.splash-strip { border-bottom: 1px solid var(--border); background: var(--bg); }
.splash-strip-inner { display: grid; grid-template-columns: 1.15fr repeat(3, 1fr); width: min(1180px, calc(100% - 3rem)); margin: 0 auto; }
.splash-strip-item { min-height: 132px; padding: 2rem; border-left: 1px solid var(--border); }
.splash-strip-item:last-child { border-right: 1px solid var(--border); }
.splash-strip-item strong { display: block; margin-bottom: .45rem; color: var(--text); font-size: .9rem; }
.splash-strip-item span { display: block; color: var(--muted); font-size: .87rem; line-height: 1.5; }
.splash-section { width: min(1060px, calc(100% - 3rem)); margin: 0 auto; padding: clamp(5rem, 9vw, 8rem) 0; }
.splash-section-heading { display: grid; grid-template-columns: .75fr 1.25fr; gap: clamp(2rem, 7vw, 7rem); margin-bottom: 4rem; }
.splash-kicker { color: var(--accent-2); font: 800 .75rem/1.4 ui-monospace, monospace; letter-spacing: .1em; text-transform: uppercase; }
.splash-section h2 { max-width: 16ch; font-size: clamp(2.3rem, 4vw, 4rem); line-height: 1.02; letter-spacing: -.055em; }
.splash-workflow { border-top: 1px solid var(--border); }
.splash-workflow-row { display: grid; grid-template-columns: 70px 1fr 1fr; gap: 2rem; padding: 2rem 0; border-bottom: 1px solid var(--border); }
.splash-workflow-row > span { color: var(--accent); font: 800 .8rem/1.5 ui-monospace, monospace; }
.splash-workflow-row h3 { margin-bottom: .35rem; font-size: 1.15rem; }
.splash-workflow-row p { color: var(--muted); line-height: 1.6; }
.splash-code-pair { display: grid; grid-template-columns: 1fr 1fr; margin-top: 4.5rem; border: 1px solid var(--border); background: var(--surface); }
.splash-code-pair > div { min-width: 0; padding: 1.5rem; }
.splash-code-pair > div + div { border-left: 1px solid var(--border); }
.splash-code-pair h3 { margin-bottom: 1rem; font: 800 .75rem/1.4 ui-monospace, monospace; letter-spacing: .08em; text-transform: uppercase; }
.splash-code-pair pre { margin: 0; }
.splash-paths { display: grid; grid-template-columns: repeat(3, 1fr); border-top: 1px solid var(--border); border-left: 1px solid var(--border); }
.splash-path { min-height: 230px; padding: 1.6rem; border-right: 1px solid var(--border); border-bottom: 1px solid var(--border); background: var(--surface); color: var(--text); text-decoration: none; }
.splash-path:hover { background: var(--accent-soft); text-decoration: none; }
.splash-path small { display: block; margin-bottom: 2.8rem; color: var(--accent-2); font: 800 .7rem/1.4 ui-monospace, monospace; letter-spacing: .09em; text-transform: uppercase; }
.splash-path strong { display: block; margin-bottom: .55rem; font-size: 1.2rem; }
.splash-path span { color: var(--muted); font-weight: 400; line-height: 1.5; }
.splash-final { padding: clamp(5rem, 9vw, 8rem) 1.5rem; border-top: 1px solid var(--border); background: var(--surface); text-align: center; }
.splash-final h2 { max-width: 720px; margin: 0 auto 1.2rem; font-size: clamp(2.4rem, 5vw, 4.8rem); line-height: 1; letter-spacing: -.06em; }
.splash-final p { max-width: 560px; margin: 0 auto; color: var(--muted); font-size: 1.08rem; }
.splash-final .splash-actions { justify-content: center; }

/* M-Press self-hosted landing page. Content and presentation stay separate so
   the product story can change without disturbing the documentation shell. */
.mp-home-shell { width: min(1180px, calc(100% - 3rem)); margin: 0 auto; }
.mp-home-hero { min-height: 690px; padding: 5rem 0 4.5rem; border-bottom: 1px solid var(--border); background: var(--bg); }
.mp-home-hero-grid { display: grid; grid-template-columns: minmax(0, .95fr) minmax(560px, 1.05fr); gap: clamp(3rem, 5vw, 4.5rem); align-items: center; }
.mp-home-hero-copy { padding: 2rem 0; }
.mp-home-hero h1 { max-width: 14ch; margin: 0 0 1.45rem; font-size: clamp(3.2rem, 4.5vw, 4rem); line-height: .98; letter-spacing: -.065em; text-wrap: balance; }
.mp-home-hero-copy > p { max-width: 560px; margin: 0; color: var(--muted); font-size: 1.12rem; line-height: 1.65; }
.mp-home-actions { display: flex; flex-wrap: wrap; gap: .75rem; margin-top: 2rem; }
.mp-home-button, .landing-page .mpress-button { display: inline-flex; min-height: 46px; align-items: center; justify-content: center; padding: .65rem 1rem; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); color: var(--text); cursor: pointer; font: inherit; font-size: .9rem; font-weight: 720; text-decoration: none; }
.mp-home-button:hover, .landing-page .mpress-button:hover { border-color: var(--accent); background: var(--accent-soft); text-decoration: none; }
.mp-home-button-primary, .landing-page .mpress-button-primary { border-color: var(--accent); background: var(--accent); color: #fff; box-shadow: 0 10px 28px color-mix(in srgb, var(--accent) 18%, transparent); }
.mp-home-button-primary:hover, .landing-page .mpress-button-primary:hover { background: color-mix(in srgb, var(--accent) 88%, white); color: #fff; }
.mp-home-product { overflow: hidden; border: 1px solid var(--border); border-radius: 12px; background: color-mix(in srgb, var(--surface) 78%, var(--bg)); box-shadow: 0 30px 80px rgba(0, 0, 0, .26); }
.mp-home-docbar, .mp-home-showcase-bar { display: flex; min-height: 52px; align-items: center; gap: .65rem; padding: 0 .8rem; border-bottom: 1px solid var(--border); background: var(--surface); color: var(--muted); font-size: .72rem; }
.mp-home-docbar strong, .mp-home-showcase-bar strong { color: var(--text); }
.mp-home-docmark { display: grid; width: 25px; height: 25px; place-items: center; border-radius: 5px; background: var(--accent); color: #fff; font-size: .72rem; font-weight: 850; }
.mp-home-docsearch { margin-left: auto; padding: .35rem .65rem; border: 1px solid var(--border); border-radius: 6px; color: var(--muted); }
.mp-home-docmode { padding: .35rem .55rem; border: 1px solid var(--border); border-radius: 6px; }
.mp-home-docbody { display: grid; grid-template-columns: 155px minmax(0, 1fr); min-height: 360px; }
.mp-home-docbody nav, .mp-home-showcase-body > nav { padding: 1.25rem .8rem; border-right: 1px solid var(--border); }
.mp-home-docbody nav small, .mp-home-showcase-body > nav small { display: block; margin: 1.1rem .55rem .35rem; color: var(--muted); font: .58rem/1.3 ui-monospace, monospace; letter-spacing: .08em; }
.mp-home-docbody nav small:first-child, .mp-home-showcase-body > nav small:first-child { margin-top: 0; }
.mp-home-docbody nav a, .mp-home-showcase-body > nav a { display: block; padding: .38rem .55rem; border-radius: 5px; color: var(--muted); font-size: .7rem; font-weight: 560; }
.mp-home-docbody nav a.active, .mp-home-showcase-body > nav a.active { background: var(--accent-soft); color: color-mix(in srgb, var(--accent) 78%, white); }
.mp-home-docbody article { padding: 1.5rem 1.7rem; }
.mp-home-breadcrumb { margin-bottom: .75rem; color: var(--muted); font-size: .66rem; }
.mp-home-docbody article h2 { margin: 0 0 .7rem; font-size: 1.6rem; letter-spacing: -.035em; }
.mp-home-docbody article h3, .mp-home-showcase article h4 { margin: 1.4rem 0 .5rem; font-size: .9rem; }
.mp-home-docbody article p { margin: 0; color: var(--muted); font-size: .73rem; line-height: 1.6; }
.mp-home-docbody pre, .mp-home-showcase pre { margin: 1.15rem 0 0; padding: .75rem .9rem; border-color: var(--border); border-radius: 7px; background: var(--code); color: var(--code-text); font-size: .72rem; line-height: 1.7; }
.mp-home-docbody pre span, .mp-home-showcase pre span { color: color-mix(in srgb, var(--accent) 78%, white); }
.mp-home-terminal { display: flex; gap: .6rem; padding: .75rem 1rem; border-top: 1px solid var(--border); background: var(--code); color: var(--code-text); font: .69rem/1.4 ui-monospace, monospace; }
.mp-home-terminal .prompt { color: color-mix(in srgb, var(--accent) 78%, white); }
.mp-home-terminal .result { margin-left: auto; color: var(--muted); }
.mp-home-terminal strong { color: color-mix(in srgb, var(--accent) 68%, white); }
.mp-home-site-screenshot { min-width: 0; overflow: visible; border: 0; border-radius: 12px; background: transparent; box-shadow: none; }
.mp-home-site-screenshot .mpress-theme-image { display: block; }
.mp-home-site-screenshot img { display: block; width: 100%; height: 100%; margin: 0; border-radius: 12px; object-fit: contain; object-position: top left; }
.mp-home-site-screenshot .mpress-image-expand { height: 100%; margin: 0; border: 1px solid color-mix(in srgb, var(--text) 16%, var(--border)); border-radius: 12px; background: transparent; box-shadow: 0 18px 48px rgb(0 0 0 / .22), 0 3px 12px rgb(0 0 0 / .16); }
.mp-home-site-screenshot .mpress-image-expand-content { display: block; height: 100%; overflow: hidden; border-radius: inherit; }
.mp-home-site-screenshot-hero { aspect-ratio: 16 / 10; }
.mp-home-site-screenshot-showcase { max-height: 650px; }

.mp-home-section { padding: clamp(5rem, 8vw, 8rem) 0; border-bottom: 1px solid var(--border); }
.mp-home-section-heading { display: grid; grid-template-columns: minmax(0, 1fr) minmax(280px, .7fr); gap: 5rem; align-items: center; margin-bottom: 3rem; }
.mp-home-section-heading h2, .mp-home-column-heading h2, .mp-home-migration h2, .mp-home-final h2 { margin: 0; color: var(--text); font-size: clamp(2.2rem, 3.8vw, 3.8rem); line-height: 1.04; letter-spacing: -.055em; text-wrap: balance; }
.mp-home-section-heading p, .mp-home-column-heading p, .mp-home-migration p, .mp-home-final p { margin: 0; color: var(--muted); line-height: 1.65; }
.mp-home-section-heading h2 { max-width: 19ch; font-size: clamp(2rem, 3vw, 3rem); line-height: 1.08; letter-spacing: -.045em; }
.mp-home-section-heading > .mpress-column:last-child { max-width: 440px; justify-self: end; }

.mp-home-story-section { position: relative; overflow: hidden; background: var(--bg); }
.mp-home-story-section::before { content: ""; position: absolute; width: 420px; height: 420px; border-radius: 50%; background: color-mix(in srgb, var(--accent) 8%, transparent); filter: blur(110px); opacity: .55; pointer-events: none; }
.mp-home-story-section:nth-of-type(odd)::before { top: -260px; right: -210px; }
.mp-home-story-section:nth-of-type(even)::before { bottom: -270px; left: -220px; }
.mp-home-story-components, .mp-home-story-accessibility, .mp-home-story-custom { background: color-mix(in srgb, var(--surface) 32%, var(--bg)); }
.mp-home-story { position: relative; display: grid; grid-template-columns: minmax(0, .88fr) minmax(470px, 1.12fr); gap: clamp(3rem, 7vw, 7rem); align-items: center; }
.mp-home-story-reversed .mp-home-story-copy { order: 2; }
.mp-home-story-reversed .mp-home-story-visual { order: 1; }
.mp-home-story-copy h2 { max-width: 15ch; margin: 0 0 1.35rem; color: var(--text); font-size: clamp(2.3rem, 4.2vw, 4rem); font-weight: 660; letter-spacing: -.055em; line-height: 1.02; text-wrap: balance; }
.mp-home-story-copy > p { max-width: 58ch; margin: 0 0 1.25rem; color: var(--muted); font-size: .98rem; line-height: 1.72; }
.mp-home-story-copy > ul { display: grid; gap: .82rem; margin: 1.8rem 0 0; padding: 0; list-style: none; }
.mp-home-story-copy > ul li { position: relative; padding-left: 1.4rem; color: var(--muted); font-size: .87rem; line-height: 1.55; }
.mp-home-story-copy > ul li::before { content: ""; position: absolute; top: .63em; left: 0; width: 5px; height: 5px; border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 11%, transparent); }
.mp-home-story-copy > ul strong { color: var(--text); }
.mp-home-story-copy > .mpress-button { margin-top: 1.75rem; }
.mp-home-story-copy > .mpress-button + .mpress-button { margin-left: .55rem; }
.mp-home-story-copy > p:has(> .mpress-button) { display: flex; flex-wrap: wrap; gap: .55rem; margin: 1.75rem 0 0; }
.mp-home-story-copy > p:has(> .mpress-button) .mpress-button { margin: 0; }
.mp-home-story-visual { position: relative; min-width: 0; min-height: 390px; padding: 0; overflow: visible; border: 0; border-radius: 0; background: transparent; box-shadow: none; }
.mp-home-story-visual h3 { margin: 0 0 1.3rem; color: var(--text); font-size: .82rem; font-weight: 720; letter-spacing: .01em; }
.mp-home-story-visual > p { color: var(--muted); font-size: .78rem; line-height: 1.6; }
.mp-home-story-visual > p:last-child { margin-bottom: 0; }

.mp-home-stack-visual { display: flex; flex-direction: column; justify-content: center; }
.mp-home-stack-visual .mpress-file-tabs { display: grid; width: 100%; height: 440px; grid-template-rows: auto minmax(0, 1fr); }
.mp-home-stack-visual .mpress-file-tabs > [role="tabpanel"] { min-height: 0; height: 100%; overflow: auto; background: var(--code); }
.mp-home-stack-visual .mpress-file-tabs > [role="tabpanel"] > pre { box-sizing: border-box; min-height: 100%; }
.mp-home-stack-visual .mpress-file-tabs > [role="tabpanel"] > p { height: 100%; margin: 0; }
.mp-home-stack-visual .mpress-file-tabs .mpress-image-expand,
.mp-home-stack-visual .mpress-file-tabs .mpress-image-expand-content { display: block; width: 100%; height: 100%; }
.mp-home-stack-visual .mpress-file-tabs .mpress-image-expand img { width: 100%; height: 100%; object-fit: contain; object-position: center; }
.mp-home-stack-visual pre { position: relative; margin: 0 0 1.2rem; padding: 1.4rem 1.5rem; border: 1px solid color-mix(in srgb, var(--border) 85%, transparent); border-radius: 8px; background: var(--code); color: var(--code-text); box-shadow: inset 0 1px rgb(255 255 255 / .025); font-size: .76rem; line-height: 1.62; }
.mp-home-stack-visual > p:last-child { display: flex; align-items: center; gap: .55rem; margin-top: .1rem; color: var(--text); font: .69rem/1.5 ui-monospace, monospace; }
.mp-home-stack-visual > p:last-child::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: #22c981; box-shadow: 0 0 0 5px rgb(34 201 129 / .1); }

.mp-home-story-image { min-height: 0; padding: 0; overflow: visible; border-radius: 12px; aspect-ratio: 16 / 10; }
.mp-home-story-image > p, .mp-home-site-screenshot > p { height: 100%; margin: 0; }
.mp-home-story-image .mpress-theme-image, .mp-home-story-image picture { display: block; height: 100%; }
.mp-home-story-image img { display: block; width: 100%; height: 100%; margin: 0; border: 0; border-radius: 12px; object-fit: contain; object-position: top left; }
.mp-home-story-image .mpress-image-expand { height: 100%; margin: 0; border: 1px solid color-mix(in srgb, var(--text) 16%, var(--border)); border-radius: 12px; background: transparent; box-shadow: 0 18px 48px rgb(0 0 0 / .22), 0 3px 12px rgb(0 0 0 / .16); }
.mp-home-story-image .mpress-image-expand-content { display: block; height: 100%; overflow: hidden; border-radius: inherit; }

.mp-home-story-components .mp-home-story { grid-template-columns: 1fr; gap: clamp(2.75rem, 5vw, 4.5rem); align-items: start; }
.mp-home-story-components .mp-home-story-copy { display: grid; max-width: none; grid-template-columns: minmax(280px, .82fr) minmax(0, 1.18fr); gap: 0 clamp(3rem, 7vw, 7rem); }
.mp-home-story-components .mp-home-story-copy h2 { grid-column: 1; grid-row: 1; margin: 0; }
.mp-home-story-components .mp-home-story-copy > p:first-of-type { grid-column: 2; grid-row: 1; margin: .15rem 0 0; }
.mp-home-story-components .mp-home-story-copy > ul { grid-column: 2; grid-row: 2; margin-top: 1.4rem; }
.mp-home-story-components .mp-home-story-copy > p:has(> .mpress-button) { grid-column: 1; grid-row: 2; align-self: start; margin-top: 1.8rem; }
.mp-home-story-components .mp-home-story-visual { order: 2; width: 100%; min-height: 0; }
.mp-home-components-visual { min-height: 540px; }
.mp-home-components-visual .mpress-preview-tabs { display: grid; height: 540px; margin: 0; grid-template-rows: auto minmax(0, 1fr); }
.mp-home-components-visual .mp-home-demo-tabs { gap: .15rem 1rem; margin-top: 0; flex-wrap: wrap; }
.mp-home-components-visual .mp-home-demo-tabs [role="tab"] { flex: 0 0 auto; }
.mp-home-components-visual .mpress-preview-tabs > [role="tabpanel"] { min-height: 0; padding-top: 1rem; overflow: auto; }
.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1.25rem; align-content: start; align-items: start; }
.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) > :not(.mpress-preview-source) { min-width: 0; grid-column: 1; }
.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) > .mpress-preview-source { min-width: 0; grid-column: 2; grid-row: 1; margin-top: 0; }
.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) > .mpress-qr { margin-inline: auto; justify-self: center; }
.mp-home-components-visual .mpress-terminal, .mp-home-components-visual .mpress-steps, .mp-home-components-visual .mpress-admonition { margin-bottom: 0; }
.mp-home-components-visual .mpress-codeframe { margin: 1.15rem 0 0; }
.mp-home-components-visual .mpress-codeframe-header { min-height: 34px; }
.mp-home-components-visual .mpress-codeframe pre { max-height: 220px; overflow: auto; font-size: .72rem; line-height: 1.55; }
.mpress-preview-source { margin: 1.15rem 0 0; overflow: hidden; border: 1px solid var(--border); border-radius: 7px; background: var(--code); }
.mpress-preview-source-label { display: flex; height: 34px; align-items: center; padding: 0 .8rem; border-bottom: 1px solid var(--border); background: var(--surface); color: var(--muted); font: 650 .66rem/1.3 ui-monospace, monospace; letter-spacing: .04em; text-transform: uppercase; }
.mpress-preview-source pre { max-height: 220px; margin: 0; padding: 1rem; overflow: auto; border: 0; border-radius: 0; background: var(--code); font-size: .72rem; line-height: 1.55; }
.mp-home-story-components .mpress-preview-source pre { max-height: 390px; }
.mp-home-story-components .mpress-preview-source pre code { white-space: pre-wrap; overflow-wrap: anywhere; }
.mp-home-components-visual .mpress-step-title { font-size: 15px; }
.mp-home-components-visual .mpress-step-content { font-size: 13px; }

.mp-home-speed-visual { display: flex; flex-direction: column; justify-content: center; }
.mp-home-speed-visual > p:first-of-type { display: flex; align-items: center; gap: .55rem; margin: 0 0 1.4rem; color: var(--text); }
.mp-home-speed-visual > p:first-of-type::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: #22c981; box-shadow: 0 0 0 5px rgb(34 201 129 / .1); }
.mp-home-speed-visual .mp-home-capabilities { display: grid; gap: .9rem; margin: .1rem 0 1.7rem; }
.mp-home-speed-visual .mp-home-capabilities > div { position: relative; display: grid; grid-template-columns: minmax(145px, .9fr) 1fr; gap: 1rem; padding: 0 0 .65rem; border-top: 0; border-bottom: 1px solid var(--border); }
.mp-home-speed-visual .mp-home-capabilities > div::after { content: ""; position: absolute; bottom: -1px; left: 0; width: 72%; height: 2px; background: color-mix(in srgb, var(--accent) 68%, white); transform-origin: left; animation: mp-home-progress 4.8s ease-in-out infinite; }
.mp-home-speed-visual .mp-home-capabilities > div:nth-child(2)::after { width: 89%; animation-delay: -.8s; }
.mp-home-speed-visual .mp-home-capabilities > div:nth-child(3)::after { width: 57%; animation-delay: -1.6s; }
.mp-home-speed-visual .mp-home-capabilities > div:nth-child(4)::after { width: 78%; animation-delay: -2.4s; }
.mp-home-speed-visual .mp-home-capabilities dt { font-size: .75rem; }
.mp-home-speed-visual .mp-home-capabilities dd { font-size: .71rem; text-align: right; }
.mp-home-speed-visual > p:nth-last-child(2) { margin: 0 0 .3rem; color: var(--text); }
.mp-home-speed-visual > p:last-child { margin: 0; }
@keyframes mp-home-progress { 0%, 100% { opacity: .62; transform: scaleX(.92); } 50% { opacity: 1; transform: scaleX(1); } }

.mp-home-build-evidence { animation: none; }
.mp-home-build-evidence .mp-home-capabilities { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; }
.mp-home-build-evidence .mp-home-capabilities > div { display: block; padding: 1rem; border: 1px solid var(--border); border-radius: 8px; background: color-mix(in srgb, var(--panel) 62%, transparent); }
.mp-home-build-evidence .mp-home-capabilities > div::after { display: none; }
.mp-home-build-evidence .mp-home-capabilities dt { margin-bottom: .35rem; color: var(--text); font-size: 1rem; font-variant-numeric: tabular-nums; }
.mp-home-build-evidence .mp-home-capabilities dd { color: var(--muted); text-align: left; }

.mp-home-accessibility-visual { display: flex; flex-direction: column; justify-content: center; }
.mp-home-accessibility-visual > p:first-of-type { width: max-content; margin: 0 0 1.45rem; padding: .45rem .65rem; border-bottom: 1px solid var(--accent); color: var(--muted); font-size: .74rem; }
.mp-home-accessibility-visual > p:first-of-type strong { color: var(--text); }
.mp-home-accessibility-visual > ul { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .55rem; margin: 0 0 1.8rem; padding: 0; list-style: none; }
.mp-home-accessibility-visual > ul li { margin: 0; padding: .8rem; border: 1px solid var(--border); border-radius: 7px; background: color-mix(in srgb, var(--panel) 55%, transparent); color: var(--text); font-size: .72rem; }
.mp-home-accessibility-visual input { accent-color: var(--accent); }
.mp-home-accessibility-visual > p:nth-last-child(3) { margin: 0 0 .65rem; color: var(--text); font-size: .72rem; }
.mp-home-accessibility-visual > p:nth-last-child(2) { display: inline-flex; width: max-content; margin: 0 0 1.5rem; padding: .45rem .7rem; border: 1px solid var(--border); border-radius: 6px; background: var(--panel); color: var(--muted); }
.mp-home-accessibility-visual > p:last-child { max-width: 48ch; margin: 0; padding-top: 1rem; border-top: 1px solid var(--border); }

.mp-home-accessibility-carousel { display: flex; min-height: 430px; align-items: stretch; padding: 1rem; }
.mp-home-accessibility-carousel .mpress-carousel { display: flex; width: 100%; margin: 0; flex-direction: column; }
.mp-home-accessibility-carousel .mpress-carousel-viewport { flex: 1; min-height: 0; }
.mp-home-accessibility-carousel .mpress-carousel-slide { display: flex; min-height: 100%; padding: clamp(1.5rem, 4vw, 2.6rem); flex-direction: column; justify-content: center; }
.mp-home-accessibility-carousel .mpress-carousel-slide[hidden] { display: none; }
.mp-home-accessibility-carousel .mpress-carousel-slide h2 { margin: 0 0 .8rem; font-size: clamp(1.55rem, 3vw, 2.2rem); letter-spacing: -.035em; }
.mp-home-accessibility-carousel .mpress-carousel-slide p, .mp-home-accessibility-carousel .mpress-carousel-slide li { color: var(--muted); font-size: .82rem; }
.mp-home-accessibility-carousel .mpress-carousel-slide ul { margin-bottom: 0; padding-left: 1.2rem; }

.mp-home-global-visual { display: flex; flex-direction: column; justify-content: center; }
.mp-home-global-visual .mp-home-capabilities { margin-bottom: 2rem; }
.mp-home-global-visual .mp-home-capabilities > div { grid-template-columns: 125px 1fr; align-items: center; padding: .82rem 0; }
.mp-home-global-visual .mp-home-capabilities dd { position: relative; padding-left: 1.1rem; }
.mp-home-global-visual .mp-home-capabilities dd::before { content: ""; position: absolute; top: .45em; left: 0; width: 6px; height: 6px; border-radius: 50%; background: #22c981; }
.mp-home-global-visual .mp-home-capabilities > div:nth-child(3) dd::before { background: #f6ad3c; }
.mp-home-global-visual .mp-home-capabilities > div:nth-child(4) dd::before { background: var(--accent); animation: mp-home-status-pulse 1.8s ease-in-out infinite; }
.mp-home-global-visual > p:last-child { display: inline-flex; width: max-content; gap: .35rem; padding: .55rem .75rem; border: 1px solid var(--border); border-radius: 6px; background: var(--panel); }
@keyframes mp-home-status-pulse { 50% { box-shadow: 0 0 0 7px color-mix(in srgb, var(--accent) 10%, transparent); } }

.mp-home-global-screenshot, .mp-home-config-screenshot { aspect-ratio: 16 / 10; }
.mp-home-global-screenshot img, .mp-home-config-screenshot img { object-fit: cover; object-position: top left; }

.mp-home-custom-visual { display: grid; grid-template-columns: minmax(150px, .72fr) 1fr; gap: clamp(1.5rem, 4vw, 3rem); align-items: center; }
.mp-home-custom-visual > img { display: block; width: 100%; height: 100%; max-height: 400px; grid-column: 1; grid-row: 1 / 3; align-self: stretch; margin: 0; border: 1px solid var(--border); border-radius: 9px; background: var(--bg); object-fit: cover; object-position: top center; }
.mp-home-custom-visual h3 { grid-column: 2; grid-row: 1; align-self: end; margin-bottom: .8rem; font-size: 1.2rem; letter-spacing: -.025em; }
.mp-home-custom-visual > p:last-child { grid-column: 2; grid-row: 2; align-self: start; margin: 0; font-size: .82rem; }

.mp-home-showcase-section { background: color-mix(in srgb, var(--surface) 35%, var(--bg)); }
.mp-home-showcase { overflow: hidden; border: 1px solid var(--border); border-radius: 12px; background: var(--bg); box-shadow: 0 26px 70px rgba(0, 0, 0, .2); }
.mp-home-showcase-bar { min-height: 58px; padding: 0 1rem; }
.mp-home-showcase-bar label { display: flex; min-width: 300px; margin-left: auto; align-items: center; justify-content: space-between; padding: .48rem .65rem; border: 1px solid var(--border); border-radius: 6px; color: var(--muted); }
.mp-home-showcase-bar kbd { padding: .08rem .3rem; border: 1px solid var(--border); border-radius: 4px; background: var(--panel); font: inherit; }
.mp-home-showcase-bar button { min-width: 44px; padding: .45rem .55rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--muted); font: inherit; }
.mp-home-showcase-body { display: grid; grid-template-columns: 210px minmax(0, 1fr) 190px; min-height: 590px; }
.mp-home-showcase-body > nav { padding: 1.6rem 1rem; }
.mp-home-showcase article { padding: 2.2rem 3rem; }
.mp-home-showcase article h3 { margin: 0 0 .8rem; font-size: 2rem; letter-spacing: -.04em; }
.mp-home-showcase article h4 { margin-top: 2rem; font-size: 1rem; }
.mp-home-showcase article > p { max-width: 620px; color: var(--muted); font-size: .88rem; }
.mp-home-demo-tabs { display: flex; gap: 1.25rem; margin-top: .9rem; border-bottom: 1px solid var(--border); color: var(--muted); font-size: 13px; font-weight: 600; }
.mp-home-demo-tabs button { padding: .55rem .1rem; border: 0; border-bottom: 2px solid transparent; background: none; color: inherit; cursor: pointer; font: inherit; }
.mp-home-demo-tabs button[aria-selected="true"] { border-color: var(--accent); color: var(--text); }
.mpress-preview-tabs [role="tabpanel"] { padding-top: .65rem; }
.mp-home-showcase article aside { display: flex; gap: 1rem; margin-top: 1.4rem; padding: 1rem; border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border)); border-radius: 8px; background: color-mix(in srgb, var(--accent) 7%, var(--surface)); }
.mp-home-showcase article aside strong { color: color-mix(in srgb, var(--accent) 72%, white); font-size: .78rem; }
.mp-home-showcase article aside p { margin: 0; color: var(--muted); font-size: .76rem; }
.mp-home-toc { padding: 2.2rem 1.3rem; border-left: 1px solid var(--border); }
.mp-home-toc strong, .mp-home-toc a { display: block; }
.mp-home-toc strong { margin-bottom: .7rem; font-size: .72rem; }
.mp-home-toc a { padding: .3rem 0 .3rem .7rem; border-left: 1px solid var(--border); color: var(--muted); font-size: .7rem; }
.mp-home-toc a.active { border-color: var(--accent); color: var(--text); }

.mp-home-workflow-section { background: var(--bg); }
.mp-home-two-column { display: grid; grid-template-columns: 1fr 1fr; gap: 0; overflow: hidden; border: 1px solid var(--border); border-radius: 12px; background: var(--surface); }
.mp-home-two-column > div { padding: clamp(2rem, 4vw, 3.5rem); }
.mp-home-two-column > div + div { border-left: 1px solid var(--border); }
.mp-home-column-heading { margin-bottom: 2rem; }
.mp-home-column-heading h2 { margin-bottom: .8rem; font-size: clamp(2rem, 3vw, 2.85rem); }
.mp-home-column-heading p { font-size: .9rem; }
.mp-home-timeline { display: grid; gap: 0; margin: 0; padding: 0; list-style: none; }
.mp-home-timeline .mpress-step-content p { margin-bottom: 0; }
.mp-home-timeline code { padding: .12rem .3rem; border-radius: 4px; background: var(--panel); color: var(--text); }
.mp-home-capabilities { margin: 0; }
.mp-home-capabilities > div { display: grid; grid-template-columns: 155px 1fr; gap: 1.2rem; padding: 1rem 0; border-top: 1px solid var(--border); }
.mp-home-capabilities dt { color: var(--text); font-size: .82rem; font-weight: 720; }
.mp-home-capabilities dd { margin: 0; color: var(--muted); font-size: .76rem; line-height: 1.5; }

.mp-home-migration-section { background: color-mix(in srgb, var(--accent) 4%, var(--surface)); }
.mp-home-migration { display: grid; grid-template-columns: 1fr .8fr; gap: 7rem; align-items: end; }
.mp-home-migration h2 { max-width: 15ch; }
.mp-home-migration p { margin-bottom: 1.4rem; }
.mp-home-migration a { display: inline-flex; min-height: 42px; align-items: center; color: color-mix(in srgb, var(--accent) 78%, white); font-weight: 720; }

.mp-home-resources-section { background: var(--bg); }
.mp-home-resources-section .mp-home-column-heading { max-width: 520px; }
.mp-home-resources { overflow: hidden; border: 1px solid var(--border); border-radius: 10px; }
.mp-home-resources a { display: grid; grid-template-columns: 150px minmax(220px, .7fr) 1fr; gap: 1.5rem; align-items: center; min-height: 82px; padding: 1rem 1.4rem; border-bottom: 1px solid var(--border); background: var(--surface); color: var(--text); text-decoration: none; }
.mp-home-resources a:last-child { border-bottom: 0; }
.mp-home-resources a:hover { background: var(--accent-soft); text-decoration: none; }
.mp-home-resources span { color: var(--accent); font: .68rem/1.3 ui-monospace, monospace; }
.mp-home-resources strong { font-size: .88rem; }
.mp-home-resources em { color: var(--muted); font-size: .78rem; font-style: normal; font-weight: 400; }

/* Layout components respond to the space they receive, not only the viewport.
   This keeps them usable in embeds, previews, side panels, and nested pages. */
.mpress-section { container-type: inline-size; }
@container (max-width: 900px) {
  .mp-home-hero-grid { grid-template-columns: 1fr; gap: 2.25rem; }
  .mp-home-hero-copy { max-width: 680px; }
  .mp-home-product { width: 100%; max-width: 760px; }
  .mp-home-section-heading, .mp-home-story, .mp-home-migration { grid-template-columns: 1fr; }
  .mp-home-resources a { grid-template-columns: minmax(90px, 120px) minmax(0, 1fr); gap: .45rem 1rem; }
  .mp-home-resources em { grid-column: 2; }
}
@container (max-width: 640px) {
  .mp-home-shell { width: min(100% - 1.5rem, 620px); }
  .mp-home-hero { min-height: auto; padding: 2.5rem 0; }
  .mp-home-hero h1 { font-size: clamp(2.25rem, 11cqw, 3.35rem); }
  .mp-home-actions { width: 100%; }
  .mp-home-actions .mpress-button { flex: 1 1 10rem; }
  .mp-home-docbody { grid-template-columns: 1fr; min-height: auto; }
  .mp-home-docbody nav { display: none; }
  .mp-home-docbody article { min-width: 0; padding: 1.1rem; }
  .mp-home-terminal { flex-wrap: wrap; }
  .mp-home-terminal .result { width: 100%; margin-left: 0; }
  .mpress-container[style*="grid-template-columns"], .mpress-card-grid { grid-template-columns: 1fr !important; }
  .mp-home-capabilities > div { grid-template-columns: 1fr; gap: .3rem; }
  .mp-home-build-evidence .mp-home-capabilities { grid-template-columns: 1fr; }
  .mp-home-resources a { grid-template-columns: 1fr; gap: .3rem; min-height: 0; padding: 1rem; }
  .mp-home-resources em { grid-column: auto; }
}
@container (max-width: 440px) {
  .mp-home-docbar { gap: .45rem; }
  .mp-home-docsearch { display: none; }
  .mp-home-docmode { margin-left: auto; }
  .mp-home-terminal { font-size: .65rem; }
}

.mp-home-final { padding: 5rem 0; background: var(--surface); }
.mp-home-final .mp-home-shell { display: flex; align-items: center; justify-content: space-between; gap: 4rem; }
.mp-home-final h2 { max-width: 12ch; margin-bottom: .8rem; font-size: clamp(2.1rem, 3.8vw, 3.7rem); }
.mp-home-final p { max-width: 580px; }
.mp-home-final .mp-home-actions { flex: 0 0 auto; margin-top: 0; }

.blog-layout { display: grid; grid-template-columns: 210px minmax(0, 1fr); width: 100%; min-height: calc(100vh - 58px); }
.blog-sidebar { position: sticky; top: 58px; height: calc(100vh - 58px - var(--mpress-devbar-height, 0px)); padding: 3.25rem 1.15rem; overflow: auto; border-right: 1px solid var(--border); scrollbar-width: thin; }
.blog-sidebar-inner { display: flex; flex-direction: column; gap: .18rem; }
.blog-sidebar-label { margin: 1.25rem .7rem .45rem; color: var(--muted); font-size: 11px; font-weight: 700; letter-spacing: .1em; line-height: 1.2; text-transform: uppercase; }
.blog-sidebar-label:first-child { margin-top: 0; }
.blog-filter { width: 100%; padding: .5rem .7rem; border: 0; border-radius: 5px; background: transparent; color: var(--muted); cursor: pointer; font-size: 14px; line-height: 1.35; text-align: left; transition: background .15s, color .15s; }
.blog-filter:hover { background: var(--surface); color: var(--text); }
.blog-filter.active { background: var(--accent-fill); color: var(--accent-fill-text); font-weight: 650; }
.blog-rss { display: flex; align-items: center; gap: .55rem; margin: 2rem .7rem 0; padding-top: 1rem; border-top: 1px solid var(--border); color: var(--muted); font-size: 13px; text-decoration: none; }
.blog-rss:hover { color: var(--accent); text-decoration: none; }
.blog-article-layout { grid-template-columns: 260px minmax(0, 1fr); }
.blog-archive-home { display: flex; min-height: 38px; align-items: center; gap: .5rem; margin: 0 .45rem 1rem; padding: .45rem .55rem; color: var(--text); font-size: 13px; font-weight: 700; text-decoration: none; }
.blog-archive-home:hover { color: var(--accent); text-decoration: none; }
.blog-archive-home .lucide { flex: 0 0 auto; transition: transform .15s ease; }
.blog-archive-home:hover .lucide { transform: translateX(-2px); }
.blog-period { margin: 0; }
.blog-period + .blog-period { margin-top: 1.15rem; padding-top: .35rem; border-top: 1px solid var(--border); }
.blog-period-posts { display: flex; flex-direction: column; gap: .12rem; }
.blog-article-link { display: flex; min-width: 0; flex-direction: column; gap: .2rem; padding: .58rem .7rem; border-radius: 5px; color: var(--muted); text-decoration: none; transition: background .15s ease, color .15s ease; }
.blog-article-link > span { display: -webkit-box; overflow: hidden; font-size: 13px; font-weight: 600; line-height: 1.35; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.blog-article-link time { color: color-mix(in srgb, var(--muted) 78%, transparent); font-size: 10px; line-height: 1.35; }
.blog-article-link:hover { background: var(--surface); color: var(--text); text-decoration: none; }
.blog-article-link.active { background: var(--accent-fill); color: var(--accent-fill-text); }
.blog-article-link.active time { color: rgb(255 255 255 / .72); }
.blog-article-main { width: min(100%, 960px); min-width: 0; margin: 0 auto; padding: clamp(3.5rem, 7vw, 6rem) clamp(2rem, 7vw, 6rem) 7rem; }
.blog-article { max-width: 720px; margin: 0 auto; }
.blog-article-header { margin-bottom: 3.2rem; padding-bottom: 2rem; border-bottom: 1px solid var(--border); }
.blog-article-kicker { display: inline-block; margin-bottom: 1.15rem; color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .13em; text-decoration: none; text-transform: uppercase; }
.blog-article-kicker:hover { color: var(--hover); text-decoration: none; }
.blog-article-header h1 { max-width: 18ch; margin: 0; font-size: clamp(2.65rem, 6vw, 4.65rem); font-weight: 630; letter-spacing: -.05em; line-height: 1.02; text-wrap: balance; }
.blog-article-header > p { max-width: 54ch; margin: 1.25rem 0 0; color: var(--muted); font-size: 16px; line-height: 1.65; }
.blog-article-header .blog-meta { margin-top: 1.35rem; }
.blog-article-body > :first-child { margin-top: 0; }
.blog-article-body > p { max-width: 68ch; }
.blog-main { width: min(100%, 1190px); min-width: 0; margin: 0 auto; padding: clamp(3.5rem, 7vw, 6.5rem) clamp(2rem, 5vw, 5.5rem) 7rem; }
.blog-intro { display: grid; grid-template-columns: minmax(0, 1fr) minmax(260px, 470px); gap: 1rem 4rem; align-items: end; margin-bottom: 2.75rem; }
.blog-kicker { grid-column: 1 / -1; color: var(--accent-text); font-size: 12px; font-weight: 750; letter-spacing: .12em; line-height: 1; text-transform: uppercase; }
.blog-intro h1 { margin: 0; font-size: clamp(3rem, 6vw, 5.2rem); font-weight: 650; letter-spacing: -.055em; line-height: .98; }
.blog-intro p { max-width: 46ch; margin: 0 0 .35rem; color: var(--muted); font-size: 16px; line-height: 1.65; }
.blog-hero { display: grid; grid-template-columns: minmax(0, 1.12fr) minmax(330px, .88fr); min-height: 420px; margin: 0; overflow: hidden; border: 1px solid var(--border); border-radius: 12px; background: var(--surface); box-shadow: 0 24px 70px color-mix(in srgb, var(--text) 7%, transparent); font-size: inherit; }
.blog-hero-image { position: relative; display: block; min-height: 420px; overflow: hidden; background: var(--blog-image-background, var(--blog-image-default-background)); }
.blog-hero-image::after { content: ""; position: absolute; inset: 0; background: linear-gradient(120deg, transparent 65%, color-mix(in srgb, var(--surface) 18%, transparent)); pointer-events: none; }
.blog-hero[data-blog-image-fit="contain"] .blog-hero-image::after { display: none; }
.blog-hero-image img { width: 100%; height: 100%; border: 0; object-fit: cover; transition: transform .45s ease; }
.blog-hero[data-blog-image-fit="contain"] .blog-hero-image img { object-fit: contain; }
.blog-hero-image:hover img { transform: scale(1.018); }
.blog-hero[data-blog-image-mode="floating"] { grid-template-columns: minmax(360px, 1.08fr) minmax(330px, .92fr); align-items: center; }
.blog-hero[data-blog-image-mode="floating"] .blog-hero-image { --blog-floating-image-gutter: clamp(2.5rem, 4.5vw, 4rem); display: grid; width: min(var(--blog-floating-image-width, 100%), calc(100% - var(--blog-floating-image-gutter))); min-height: 0; margin: clamp(1.25rem, 2.25vw, 2rem) auto; overflow: visible; place-items: center; aspect-ratio: auto; background: transparent; }
.blog-hero[data-blog-image-mode="floating"] .blog-hero-image::after { display: none; }
.blog-hero[data-blog-image-mode="floating"] .blog-hero-image img { width: 100%; height: auto; max-height: 390px; border: 0; border-radius: 10px; object-fit: contain; box-shadow: 0 26px 65px rgb(0 0 0 / .32), 0 8px 22px rgb(0 0 0 / .2); transform: none; }
.blog-hero[data-blog-image-mode="floating"] .blog-hero-image:hover img { transform: translateY(-2px) scale(1.006); }
.blog-hero-image.no-image { display: grid; place-items: center; color: var(--accent); }
.blog-hero-image.no-image span { display: grid; width: 88px; height: 88px; place-items: center; border: 1px solid color-mix(in srgb, var(--accent) 35%, var(--border)); border-radius: 50%; background: var(--accent-soft); }
.blog-hero-copy { display: flex; min-width: 0; padding: clamp(2rem, 4vw, 3.6rem); align-items: flex-start; flex-direction: column; justify-content: center; }
.blog-hero-label-row { display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: 1.2rem; }
.blog-featured-label { color: var(--accent-text); font-size: 11px; font-weight: 800; letter-spacing: .14em; text-transform: uppercase; }
.blog-tags { display: flex; flex-wrap: wrap; gap: .4rem; margin-bottom: .85rem; }
.blog-hero-label-row .blog-tags { justify-content: flex-end; margin-bottom: 0; }
.blog-tags span { padding: .18rem .52rem; border: 1px solid var(--border); border-radius: 999px; color: var(--muted); font-size: 11px; font-weight: 650; line-height: 1.4; }
[data-blog-tags-visible="false"] .blog-tags { display: none; }
.blog-hero h2 { margin: 0; font-size: clamp(1.85rem, 3vw, 2.75rem); font-weight: 620; letter-spacing: -.035em; line-height: 1.1; }
.blog-hero[data-blog-heading-size="compact"] h2 { font-size: clamp(1.55rem, 2.3vw, 2.05rem); }
.blog-hero[data-blog-heading-size="large"] h2 { font-size: clamp(2.2rem, 3.5vw, 3.2rem); }
.blog-hero h2 a, .blog-card h3 a { color: var(--text); text-decoration: none; }
.blog-hero h2 a:hover, .blog-card h3 a:hover { color: var(--accent); }
.blog-hero-copy > p { margin: 1.15rem 0 0; color: var(--muted); font-size: 15px; line-height: 1.7; }
.blog-meta { display: flex; flex-wrap: wrap; gap: .35rem 1rem; margin-top: 1.4rem; color: var(--muted); font-size: 12px; line-height: 1.4; }
.blog-meta > * + * { position: relative; }
.blog-meta > * + *::before { content: ""; position: absolute; top: 50%; left: -.58rem; width: 2px; height: 2px; transform: translateY(-50%); border-radius: 50%; background: currentColor; }
.blog-read { display: inline-flex; align-items: center; gap: .55rem; margin-top: 1.6rem; color: var(--text); font-size: 13px; font-weight: 700; text-decoration: none; }
.blog-read:hover { color: var(--accent); text-decoration: none; }
.blog-read .lucide { transition: transform .15s; }
.blog-read:hover .lucide { transform: translateX(3px); }
.blog-archive { margin-top: 5rem; }
.blog-archive-heading { display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; margin-bottom: 1.4rem; padding-bottom: 1rem; border-bottom: 1px solid var(--border); }
.blog-archive-heading h2 { margin: 0; font-size: 1.4rem; font-weight: 650; letter-spacing: -.02em; }
.blog-archive-heading span { color: var(--muted); font-size: 12px; }
.blog-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.blog-layout[data-blog-landing-style="grid"] .blog-archive, .blog-layout[data-blog-landing-style="list"] .blog-archive { margin-top: 2.5rem; }
.blog-layout[data-blog-landing-style="list"] .blog-grid { grid-template-columns: 1fr; }
.blog-layout[data-blog-landing-style="list"] .blog-card { min-height: 170px; }
.blog-layout[data-blog-landing-style="list"] .blog-card-image { flex-basis: min(30%, 280px); }
.blog-layout[data-blog-landing-style="list"] .blog-card-copy { padding: 1.55rem 1.7rem; }
.blog-layout[data-blog-landing-style="list"] .blog-card-copy > p { max-width: 68ch; -webkit-line-clamp: 2; }
.blog-card { display: flex; min-width: 0; margin: 0; overflow: hidden; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); font-size: inherit; transition: border-color .15s, transform .15s, box-shadow .15s; }
.blog-card:hover { border-color: color-mix(in srgb, var(--accent) 36%, var(--border)); box-shadow: 0 14px 36px color-mix(in srgb, var(--text) 6%, transparent); transform: translateY(-2px); }
.blog-card-image { flex: 0 0 35%; min-width: 130px; overflow: hidden; background: var(--blog-image-background, var(--blog-image-default-background)); }
.blog-card-image img { width: 100%; height: 100%; border: 0; object-fit: cover; transition: transform .35s ease; }
.blog-card[data-blog-image-fit="contain"] .blog-card-image img { object-fit: contain; }
.blog-card:hover .blog-card-image img { transform: scale(1.025); }
.blog-card-copy { display: flex; min-width: 0; padding: 1.35rem; flex: 1; flex-direction: column; }
.blog-card h3 { margin: 0; font-size: 1.12rem; font-weight: 650; letter-spacing: -.015em; line-height: 1.28; }
.blog-card[data-blog-heading-size="compact"] h3 { font-size: .98rem; }
.blog-card[data-blog-heading-size="large"] h3 { font-size: 1.3rem; }
.blog-card-copy > p { display: -webkit-box; margin: .7rem 0 1.1rem; overflow: hidden; color: var(--muted); font-size: 13px; line-height: 1.55; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.blog-card-footer { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-top: auto; }
.blog-card-footer .blog-meta { margin-top: 0; }
.blog-card-footer > a { display: grid; width: 32px; height: 32px; flex: 0 0 auto; place-items: center; border-radius: 50%; color: var(--muted); }
.blog-card-footer > a:hover { background: var(--accent-soft); color: var(--accent); text-decoration: none; }
.blog-card[hidden], .blog-hero[hidden] { display: none; }

@media (max-width: 980px) {
  .mp-home-hero-grid { grid-template-columns: 1fr; }
  .mp-home-hero-copy { max-width: 680px; }
  .mp-home-product { max-width: 760px; }
  .mp-home-site-screenshot-hero { width: min(100%, 760px); }
  .mp-home-section-heading { grid-template-columns: 1fr; gap: 1.4rem; }
  .mp-home-section-heading > .mpress-column:last-child { max-width: 600px; justify-self: start; }
  .mp-home-story { grid-template-columns: 1fr; gap: 2.8rem; }
  .mp-home-story-copy { max-width: 690px; }
  .mp-home-story-components .mp-home-story-copy { display: block; max-width: 760px; }
  .mp-home-story-components .mp-home-story-copy > p:has(> .mpress-button) { margin-top: 1.75rem; }
  .mp-home-story-reversed .mp-home-story-copy, .mp-home-story-reversed .mp-home-story-visual { order: initial; }
  .mp-home-story-visual { width: min(100%, 760px); }
  .mp-home-story-reversed .mp-home-story-visual { justify-self: end; }
  .mp-home-showcase-body { grid-template-columns: 190px 1fr; }
  .mp-home-toc { display: none; }
  .mp-home-two-column { grid-template-columns: 1fr; }
  .mp-home-two-column > div + div { border-top: 1px solid var(--border); border-left: 0; }
  .mp-home-migration { grid-template-columns: 1fr; gap: 2rem; }
  .mp-home-final .mp-home-shell { align-items: flex-start; flex-direction: column; }
  .blog-layout { grid-template-columns: 170px minmax(0, 1fr); }
  .blog-article-layout { grid-template-columns: 220px minmax(0, 1fr); }
  .blog-main { padding-inline: 2rem; }
  .blog-intro { grid-template-columns: 1fr; }
  .blog-hero { grid-template-columns: 1fr; }
  .blog-hero[data-blog-image-mode="floating"] { grid-template-columns: 1fr; }
  .blog-hero[data-blog-image-mode="floating"] .blog-hero-image { --blog-floating-image-gutter: 3rem; width: min(var(--blog-floating-image-width, 100%), calc(100% - var(--blog-floating-image-gutter)), 760px); margin: 1.5rem auto; }
  .blog-hero-image { min-height: 340px; aspect-ratio: 16 / 8; }
  .blog-grid { grid-template-columns: 1fr; }
}

@media (max-width: 700px) {
  .mp-home-shell { width: min(100% - 2rem, 620px); }
  .mp-home-hero { min-height: auto; padding: 3.5rem 0; }
  .mp-home-hero h1 { font-size: clamp(2.35rem, 12vw, 4.3rem); }
  .mp-home-docbody { grid-template-columns: 1fr; min-height: auto; }
  .mp-home-site-screenshot-hero { aspect-ratio: 16 / 10; }
  .mp-home-site-screenshot-showcase { max-height: 440px; }
  .mp-home-docbody nav { display: none; }
  .mp-home-docbody article { padding: 1.3rem; }
  .mp-home-terminal { flex-wrap: wrap; }
  .mp-home-terminal .result { width: 100%; margin-left: 0; }
  .mp-home-section { padding: 4rem 0; }
  .mp-home-story-copy h2 { font-size: clamp(2.15rem, 10vw, 3.15rem); }
  .mp-home-story-copy > .mpress-button, .mp-home-story-copy > .mpress-button + .mpress-button { width: 100%; margin: .65rem 0 0; }
  .mp-home-story-visual { min-height: 0; padding: 1.25rem; border-radius: 9px; animation-name: none; }
  .mp-home-stack-visual .mpress-file-tabs { height: 360px; }
  .mp-home-story-image { padding: 0; aspect-ratio: 1 / .82; }
  .mp-home-story-components .mp-home-story { display: flex; gap: 0; flex-direction: column; }
  .mp-home-story-components .mp-home-story-copy { display: contents; }
  .mp-home-story-components .mp-home-story-copy h2 { order: 1; margin-bottom: 1.15rem; }
  .mp-home-story-components .mp-home-story-copy > p:first-of-type { order: 2; margin-bottom: 2rem; }
  .mp-home-story-components .mp-home-story-visual { width: 100%; padding: 0; }
  .mp-home-story-components .mp-home-story-visual { order: 3; margin-bottom: 2rem; }
  .mp-home-story-components .mp-home-story-copy > ul { order: 4; margin: 0; }
  .mp-home-story-components .mp-home-story-copy > p:has(> .mpress-button) { order: 5; margin-top: 1.6rem; }
  .mp-home-components-visual .mpress-preview-tabs { height: 690px; }
  .mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) { display: block; }
  .mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) > .mpress-preview-source { margin-top: 1.15rem; }
  .mp-home-build-evidence .mp-home-capabilities { grid-template-columns: 1fr; }
  .mp-home-speed-visual .mp-home-capabilities > div { grid-template-columns: 1fr; gap: .25rem; }
  .mp-home-speed-visual .mp-home-capabilities dd { padding-bottom: .1rem; text-align: left; }
  .mp-home-accessibility-visual > ul { grid-template-columns: 1fr; }
  .mp-home-global-visual .mp-home-capabilities > div { grid-template-columns: 105px 1fr; }
  .mp-home-custom-visual { grid-template-columns: minmax(115px, .58fr) 1fr; gap: 1.2rem; }
  .mp-home-showcase-bar label, .mp-home-showcase-bar button { display: none; }
  .mp-home-showcase-body { display: block; min-height: auto; }
  .mp-home-showcase-body > nav { display: none; }
  .mp-home-showcase article { padding: 1.5rem; }
  .mp-home-showcase article h3 { font-size: 1.65rem; }
  .mp-home-two-column { border-radius: 9px; }
  .mp-home-two-column > div { padding: 1.5rem; }
  .mp-home-capabilities > div { grid-template-columns: 1fr; gap: .3rem; }
  .mp-home-resources a { grid-template-columns: 1fr; gap: .35rem; min-height: 0; padding: 1.1rem; }
  .mp-home-actions, .mp-home-final .mp-home-actions { width: 100%; }
  .mp-home-button, .landing-page .mpress-button { flex: 1 1 auto; }
  :is(.blog-index-page, .blog-article-page) #menu { display: none; }
  .blog-layout { display: block; }
  .blog-sidebar { position: static; height: auto; padding: 1rem; overflow-x: auto; border-right: 0; border-bottom: 1px solid var(--border); }
  .blog-sidebar-inner { width: max-content; max-width: none; align-items: center; flex-direction: row; gap: .35rem; }
  .blog-sidebar-label, .blog-rss { display: none; }
  .blog-filter { width: auto; min-height: 40px; padding: .45rem .8rem; white-space: nowrap; }
  .blog-article-page .blog-sidebar-inner { align-items: stretch; }
  .blog-article-page .blog-archive-home { min-height: 40px; flex: 0 0 auto; margin: 0; white-space: nowrap; }
  .blog-article-page .blog-period { display: contents; }
  .blog-article-page .blog-period-posts { flex-direction: row; gap: .35rem; }
  .blog-article-page .blog-article-link { width: 190px; flex: 0 0 auto; padding: .45rem .7rem; }
  .blog-article-page .blog-article-link time { display: none; }
  .blog-article-main { padding: 3.2rem 1.25rem 5rem; }
  .blog-article-header { margin-bottom: 2.4rem; padding-bottom: 1.6rem; }
  .blog-article-header h1 { font-size: clamp(2.35rem, 12vw, 3.5rem); }
  .blog-main { padding: 3.2rem 1rem 5rem; }
  .blog-intro { margin-bottom: 2rem; }
  .blog-intro h1 { font-size: clamp(2.7rem, 16vw, 4.2rem); }
  .blog-intro p { font-size: 15px; }
  .blog-hero { display: block; min-height: 0; }
  .blog-hero[data-blog-image-mode="floating"] .blog-hero-image { --blog-floating-image-gutter: 2.5rem; width: min(var(--blog-floating-image-width, 100%), calc(100% - var(--blog-floating-image-gutter))); margin: 1.25rem auto; }
  .blog-hero-image { min-height: 0; aspect-ratio: 16 / 9; }
  .blog-hero-copy { padding: 1.5rem; }
  .blog-hero h2 { font-size: 1.75rem; }
  .blog-archive { margin-top: 3.5rem; }
  .blog-card { display: block; }
  .blog-card-image { display: block; min-width: 0; aspect-ratio: 16 / 8; }
}
.page-meta { display: flex; min-height: 25px; align-items: center; justify-content: space-between; gap: 1rem; margin-top: 4.5rem; color: var(--muted); font-size: 13px; }
.page-meta a { display: inline-flex; align-items: center; gap: .4rem; color: var(--muted); text-decoration: none; }
.page-meta a:hover { color: var(--text); }
.page-meta p { margin: 0; }
.pager { display: flex; justify-content: space-between; gap: .75rem; margin-top: 1.5rem; padding: 0; border: 0; }
.pager > a { display: flex; min-height: 93px; flex: 1 1 0; flex-direction: column; justify-content: center; gap: .15rem; padding: .85rem 1rem; border: 1px solid var(--border); border-radius: 7px; color: var(--muted); text-decoration: none; }
.pager > a[rel="next"] { align-items: flex-end; text-align: right; }
.pager-direction { display: flex; width: 100%; align-items: center; justify-content: flex-start; gap: .45rem; font-size: 13px; }
.pager > a[rel="next"] .pager-direction { justify-content: flex-end; }
.pager strong { color: var(--text); font-size: 20px; font-weight: 500; line-height: 1.25; }
.pager a:hover { border-color: var(--accent); color: var(--accent); }
body > footer { display: flex; justify-content: space-between; max-width: 1500px; margin: 0 auto; padding: 1.5rem; border-top: 1px solid var(--border); color: var(--muted); font-size: .82rem; }
body > footer a { color: inherit; text-decoration: none; }
body > footer a:hover { text-decoration: underline; text-underline-offset: .18em; }
body > footer strong { color: var(--text); }
body.docs-page { position: relative; }
.docs-page > footer { position: absolute; z-index: 5; right: 0; bottom: var(--mpress-devbar-height, 0px); left: 0; max-width: none; min-height: 72px; margin: 0; background: color-mix(in srgb, var(--bg) 94%, transparent); backdrop-filter: blur(12px); }
.docs-page .sidebar nav::after, .docs-page .toc::after { content: ""; display: block; height: 72px; flex: 0 0 72px; }
.mpress-contribute-trigger { display: inline-flex; min-height: 34px; align-items: center; justify-content: center; gap: .42rem; padding: .38rem .72rem; border: 1px solid var(--border); border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-weight: 600; }
.mpress-contribute-trigger:hover { border-color: color-mix(in srgb, var(--accent) 42%, var(--border)); background: var(--accent-soft); color: var(--text); }
.mpress-contribute-trigger:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.header-contribute { width: 34px; min-width: 34px; height: 34px; padding: 0; border-color: transparent; color: var(--accent); }
.header-contribute span { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
.footer-contribute { min-height: 32px; padding-block: .28rem; }
.mpress-contribute-dialog { width: min(92vw, 720px); padding: 0; overflow: hidden; border: 1px solid var(--border); border-radius: 10px; background: var(--surface-solid); color: var(--text); box-shadow: 0 32px 90px rgb(0 0 0 / .48); }
.mpress-contribute-dialog::backdrop { background: rgb(3 6 12 / .72); backdrop-filter: blur(4px); }
.mpress-contribute-dialog > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; padding: 1.4rem 1.5rem 1.15rem; border-bottom: 1px solid var(--border); }
.mpress-contribute-heading { display: flex; min-width: 0; align-items: center; gap: .9rem; }
.mpress-contribute-mark { display: grid; width: 48px; height: 48px; flex: 0 0 auto; place-items: center; border: 1px solid color-mix(in srgb, var(--accent) 42%, var(--border)); border-radius: 9px; background: var(--accent-soft); color: var(--accent); }
.mpress-contribute-dialog .mpress-contribute-eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase; }
.mpress-contribute-dialog h2 { margin: .25rem 0 0; font-size: 24px; line-height: 1.2; }
.mpress-contribute-dialog .mpress-contribute-intro { margin: 0; padding: 1.25rem 1.5rem .8rem; color: var(--muted); font-size: 15px; line-height: 1.65; }
.mpress-contribute-choices { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .75rem; padding: .45rem 1.5rem .8rem; }
.mpress-contribute-choices-public { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.mpress-contribute-choices[hidden], [data-contribute-setup][hidden] { display: none !important; }
.mpress-contribute-choices > button { display: grid; grid-template-columns: 22px 1fr; gap: .75rem; min-width: 0; padding: 1rem; border: 1px solid var(--border); border-radius: 8px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; text-align: left; }
.mpress-contribute-choices > button:hover { border-color: color-mix(in srgb, var(--accent) 46%, var(--border)); background: var(--accent-soft); color: var(--text); }
.mpress-contribute-choices > button:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.mpress-contribute-choices svg { margin-top: .15rem; color: var(--accent); }
.mpress-contribute-choices span { display: grid; gap: .2rem; }
.mpress-contribute-choices strong { color: var(--text); font-size: 14px; }
.mpress-contribute-choices small { color: var(--muted); font-size: 12px; line-height: 1.45; }
.mpress-contribute-close { display: grid; width: 36px; height: 36px; flex: 0 0 auto; place-items: center; border: 1px solid var(--border); border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; }
.mpress-contribute-close:hover { background: var(--surface); color: var(--text); }
.mpress-contribute-platform { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin: .3rem 1.5rem .55rem; color: var(--muted); font-size: 12px; }
.mpress-contribute-platform strong { color: var(--text); font-size: 13px; }
.mpress-contribute-command { margin: 0 1.5rem; padding: 1rem 1.1rem; overflow-x: auto; border: 1px solid var(--border); border-radius: 7px; background: var(--code); color: var(--code-text); scrollbar-width: thin; }
.mpress-contribute-command code { display: block; overflow-wrap: anywhere; font-size: 13px; line-height: 1.55; white-space: pre-wrap; user-select: all; }
.mpress-contribute-copy { display: flex; width: calc(100% - 3rem); min-height: 46px; align-items: center; justify-content: center; gap: .5rem; margin: .85rem 1.5rem 0; border: 1px solid var(--accent); border-radius: 7px; background: var(--accent); color: var(--accent-contrast); cursor: pointer; font: inherit; font-size: 14px; font-weight: 750; }
.mpress-contribute-copy:hover { filter: brightness(1.08); }
.mpress-contribute-copy:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
.mpress-contribute-explainer { margin: .85rem 1.5rem 0; border-top: 1px solid var(--border); color: var(--muted); font-size: 13px; }
.mpress-contribute-explainer > summary { padding: .9rem 0 .7rem; color: var(--text); cursor: pointer; font-weight: 700; }
.mpress-contribute-explainer ol { margin: 0; padding: 0 0 0 1.25rem; line-height: 1.65; }
.mpress-contribute-explainer p { margin: .7rem 0; }
.mpress-contribute-explainer a { color: var(--accent); }
.mpress-contribute-manual { display: grid; gap: .35rem; margin: .8rem 0 0; padding: .75rem .85rem; border: 1px solid var(--border); border-radius: 7px; background: var(--code); }
.mpress-contribute-manual strong { color: var(--text); font-size: 12px; }
.mpress-contribute-manual code { overflow-wrap: anywhere; color: var(--code-text); font-size: 12px; user-select: all; }
.mpress-contribute-dialog .mpress-contribute-status { min-height: 3.7rem; margin: 0; padding: .8rem 1.5rem 1.2rem; color: var(--muted); font-size: 13px; line-height: 1.5; }
.mpress-edit-segment { border-radius: 3px; transition: background .14s ease, box-shadow .14s ease; }
body.mpress-quick-editing .mpress-edit-segment { cursor: text; box-shadow: 0 1px 0 color-mix(in srgb, var(--accent) 34%, transparent); }
body.mpress-quick-editing .mpress-edit-segment:hover { background: color-mix(in srgb, var(--accent) 8%, transparent); box-shadow: 0 1px 0 var(--accent); }
body.mpress-quick-editing .mpress-edit-segment:focus { outline: 0; background: color-mix(in srgb, var(--accent) 12%, var(--surface)); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 24%, transparent); }
body.mpress-quick-editing .mpress-edit-segment a, body.mpress-quick-editing .mpress-quick-edit-insert a { pointer-events: none; }
.mpress-quick-edit-insert { min-height: 1.75em; border-radius: 4px; color: var(--text); }
.mpress-quick-edit-insert:empty::before { content: "Write a new paragraph"; color: var(--muted); }
.mpress-quick-edit-bar { position: fixed; z-index: 30; right: 1rem; bottom: calc(var(--mpress-devbar-height, 0px) + 1rem); left: 1rem; display: flex; width: min(calc(100% - 2rem), 1080px); min-height: 64px; align-items: center; justify-content: space-between; gap: .8rem; margin-inline: auto; padding: .65rem .75rem; border: 1px solid color-mix(in srgb, var(--border) 78%, transparent); border-radius: 10px; background: color-mix(in srgb, var(--surface-solid) 88%, transparent); box-shadow: 0 18px 60px rgb(0 0 0 / .24); backdrop-filter: blur(18px) saturate(135%); }
.mpress-quick-edit-bar[hidden] { display: none; }
.mpress-quick-edit-state { display: flex; min-width: 0; align-items: center; gap: .65rem; }
.mpress-quick-edit-state > span:last-child { display: grid; min-width: 0; }
.mpress-quick-edit-state strong { color: var(--text); font-size: 13px; }
.mpress-quick-edit-state small { overflow: hidden; color: var(--muted); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.mpress-quick-edit-mark { display: grid; width: 34px; height: 34px; flex: 0 0 auto; place-items: center; border-radius: 7px; background: var(--accent); color: var(--accent-contrast); font-size: 13px; font-weight: 850; }
.mpress-quick-edit-format { position: fixed; z-index: 34; top: 0; left: 0; display: flex; align-items: center; gap: 2px; padding: 4px; border: 1px solid color-mix(in srgb, var(--border) 78%, transparent); border-radius: 8px; background: color-mix(in srgb, var(--surface-solid) 90%, transparent); box-shadow: 0 12px 36px rgb(0 0 0 / .26); transform: translate(-50%, calc(-100% - 10px)); backdrop-filter: blur(18px) saturate(140%); }
.mpress-quick-edit-format[hidden] { display: none; }
.mpress-quick-edit-format.is-below { transform: translate(-50%, 10px); }
.mpress-quick-edit-format > button { display: grid; width: 32px; height: 32px; place-items: center; padding: 0; border: 0; border-radius: 5px; background: transparent; color: var(--muted); cursor: pointer; }
.mpress-quick-edit-format > button:hover, .mpress-quick-edit-format > button[aria-pressed="true"] { background: var(--surface-raised); color: var(--text); }
.mpress-quick-edit-format > button:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; }
.mpress-quick-edit-format > button:disabled { cursor: default; opacity: .38; }
.mpress-quick-edit-link-form { position: absolute; bottom: calc(100% + .65rem); left: 50%; display: grid; width: min(380px, calc(100vw - 2rem)); grid-template-columns: 1fr auto auto; align-items: end; gap: .45rem; padding: .75rem; border: 1px solid var(--border); border-radius: 9px; background: color-mix(in srgb, var(--surface-solid) 94%, transparent); box-shadow: 0 18px 50px rgb(0 0 0 / .28); transform: translateX(-50%); backdrop-filter: blur(18px) saturate(135%); }
.mpress-quick-edit-link-form[hidden] { display: none; }
.mpress-quick-edit-format.is-below .mpress-quick-edit-link-form { top: calc(100% + .65rem); bottom: auto; }
.mpress-quick-edit-link-form label { grid-column: 1 / -1; color: var(--muted); font-size: 11px; font-weight: 750; }
.mpress-quick-edit-link-form input { min-width: 0; height: 36px; padding: 0 .65rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--text); font: inherit; font-size: 13px; }
.mpress-quick-edit-link-form input:focus { border-color: var(--accent); outline: 2px solid color-mix(in srgb, var(--accent) 24%, transparent); }
.mpress-quick-edit-link-form button { display: grid; min-width: 36px; height: 36px; place-items: center; padding: 0 .7rem; border: 1px solid var(--border); border-radius: 6px; background: var(--surface-raised); color: var(--text); cursor: pointer; font: inherit; font-size: 12px; font-weight: 750; }
.mpress-quick-edit-link-form button[type="submit"] { border-color: var(--accent); background: var(--accent); color: var(--accent-contrast); }
.mpress-quick-edit-actions { display: flex; align-items: center; gap: .45rem; }
.mpress-quick-edit-actions button { display: inline-flex; min-height: 38px; align-items: center; justify-content: center; gap: .35rem; padding: .45rem .7rem; border: 1px solid var(--border); border-radius: 6px; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.mpress-quick-edit-actions button:hover { background: var(--surface); color: var(--text); }
.mpress-quick-edit-actions button.primary { border-color: var(--accent); background: var(--accent); color: var(--accent-contrast); }

@media (max-width: 1180px) {
  .docs-stage, .docs-page-wide .docs-stage { display: block; width: min(100%, var(--layout-content-width)); padding-inline: 1.5rem; }
  .docs-page-header::after { right: 0; left: 0; }
  .toc { display: none; }
}

@media (max-width: 1080px) {
  .primary-links { display: none; }
  .splash-hero-inner { grid-template-columns: 1fr 420px; gap: 3rem; }
  .splash-strip-inner { grid-template-columns: 1fr 1fr; }
  .splash-strip-item:nth-child(odd) { border-right: 1px solid var(--border); }
  .splash-strip-item:nth-child(n+3) { border-top: 1px solid var(--border); }
}

@media (max-width: 760px) {
  body > header { height: 58px; padding: 0 .9rem; }
  #menu { display: block; }
  .landing-page #menu { display: none; }
  .brand-mark { width: 32px; height: 32px; }
  .theme-logo { flex-basis: min(var(--mpress-logo-width, 138px), 34vw); width: min(var(--mpress-logo-width, 138px), 34vw); height: 26px; }
  .search { display: block; flex: 0 0 44px; width: 44px; margin-left: auto; }
  .search button { width: 44px; height: 44px; padding: .5rem 0 .5rem 2.4rem; color: transparent; }
  .search-shortcut { display: none; }
  .search:focus-within { position: absolute; z-index: 11; left: .9rem; right: .9rem; width: auto; }
  .search:focus-within button { width: 100%; color: var(--text); }
  .search:focus-within ~ .header-links { visibility: hidden; }
  .mpress-search-overlay { padding: .5rem; }
  .mpress-search-dialog, .mpress-search-dialog.has-results { width: 100%; max-height: calc(100dvh - 1rem); grid-template: auto auto minmax(0, 1fr) / minmax(0, 1fr); border-radius: 9px; }
  .mpress-search-preview, .mpress-search-dialog.has-results .mpress-search-preview { display: none; }
  .mpress-search-query { min-height: 58px; }
  .mpress-search-hint { display: none; }
  article pre, .mpress-codeframe pre, .mpress-terminal pre { scrollbar-width: thin; scrollbar-color: var(--accent) transparent; -webkit-overflow-scrolling: touch; }
  article pre::-webkit-scrollbar, .mpress-codeframe pre::-webkit-scrollbar, .mpress-terminal pre::-webkit-scrollbar { height: 6px; }
  article pre::-webkit-scrollbar-thumb, .mpress-codeframe pre::-webkit-scrollbar-thumb, .mpress-terminal pre::-webkit-scrollbar-thumb { border-radius: 999px; background: color-mix(in srgb, var(--accent) 72%, transparent); }
  .header-links { margin-left: auto; }
  .utility-select:not(.accessibility-select):not(.language-select), .social-links { display: none; }
  .contribution-header { display: flex; }
  .header-contribute { width: 34px; min-width: 34px; padding: 0; border-color: transparent; }
  .header-contribute span { display: none; }
  .header-group.theme-toggle { width: 34px; min-width: 34px; margin-left: 0 !important; padding-left: 0 !important; }
  .header-group.theme-toggle::before { display: none; }
  .layout { display: block; padding: 0; border: 0; }
  .sidebar { position: fixed; z-index: 9; top: 58px; left: 0; bottom: 0; width: min(86vw, 320px); height: auto; padding: 1.25rem; transform: translateX(-105%); border-right: 1px solid var(--border); background: var(--surface-solid); box-shadow: var(--shadow); transition: transform .22s ease; }
  body.nav-open .sidebar { transform: translateX(0); }
  body.nav-open { overflow: hidden; }
  .docs-stage, .docs-page-wide .docs-stage { width: 100%; padding: 0; }
  .docs-stage > main { padding: 2.2rem 1.2rem 5rem; }
  article h1 { font-size: clamp(2rem, 9vw, 2.375rem); overflow-wrap: break-word; }
  article h2 { margin-top: 2.75rem; font-size: 24px; }
  article h3, .mpress-step-title { font-size: 20px; }
  .mpress-card-grid { grid-template-columns: 1fr; }
  .mpress-cards-3, .mpress-diff-split .mpress-diff-body, .mpress-diff-side-by-side .mpress-diff-body, .mpress-explained { grid-template-columns: 1fr; }
  .mpress-diff-pane + .mpress-diff-pane { border-left: 0; border-top: 1px solid #2a3343; }
  .mpress-explained-prose { border-top: 1px solid var(--border); border-left: 0; }
  .mpress-calendar-day-header, .mpress-calendar-cell { min-height: 2.5rem; padding: .25rem; font-size: .75rem; }
  .mpress-calendar-nav { width: 44px; height: 44px; }
  .mpress-button { min-height: 44px; }
  body > footer { align-items: center; flex-wrap: wrap; gap: .75rem; }
  .mpress-contribute-dialog { width: calc(100vw - 1rem); max-height: calc(100dvh - 1rem); }
  .mpress-contribute-dialog > header, .mpress-contribute-dialog .mpress-contribute-intro, .mpress-contribute-dialog .mpress-contribute-status { padding-inline: 1rem; }
  .mpress-contribute-choices { grid-template-columns: 1fr; padding-inline: 1rem; }
  .mpress-contribute-platform, .mpress-contribute-command { margin-inline: 1rem; }
  .mpress-contribute-copy { width: calc(100% - 2rem); margin-inline: 1rem; }
  .mpress-contribute-explainer { margin-inline: 1rem; }
  .mpress-contribute-platform { align-items: flex-start; flex-direction: column; gap: .15rem; }
  .mpress-quick-edit-bar { right: .5rem; bottom: calc(var(--mpress-devbar-height, 0px) + .5rem); left: .5rem; width: calc(100% - 1rem); align-items: stretch; flex-direction: column; }
  .mpress-quick-edit-actions { display: grid; grid-template-columns: auto auto 1fr; }
  .mpress-quick-edit-actions button { padding-inline: .55rem; }
  .mpress-quick-edit-actions button span { display: none; }
  .mpress-filetree-row { align-items: flex-start; }
  .mpress-filetree-desc { white-space: normal; text-align: right; }
  .mpress-table-toolbar { align-items: stretch; flex-direction: column; }
  .mpress-table-footer { align-items: flex-start; flex-direction: column; }
  .mpress-table-pagination button { width: 44px; height: 44px; }
  .mpress-data-table table { display: table; overflow: visible; }
  .mpress-api-pg-header-row { grid-template-columns: 1fr; }
  .mpress-api-pg-server-select, .mpress-api-pg-url-input, .mpress-api-pg-request, .mpress-api-pg-headers input, .mpress-reactive-control { min-height: 44px; }
  .mpress-copy, .mpress-tutorial-reset, .mpress-calendar-nav, .mpress-api-pg-send, .mpress-audience-select, .mpress-pricing-cta { min-height: 44px; }
  .mpress-testimonials-dot, .mpress-tutorial-check { width: 44px; height: 44px; }
  .pager a { display: flex; min-height: 44px; align-items: center; }
  .mpress-container[style*="grid-template-columns"] { grid-template-columns: 1fr !important; }
  article table { display: block; overflow-x: auto; }
  body > footer { display: block; padding: 1.2rem; }
  body > footer span { display: block; }
  .splash-hero-inner { display: block; width: min(100% - 2rem, 680px); min-height: auto; padding: 4.5rem 0; }
  .splash-hero h1 { font-size: clamp(3.2rem, 17vw, 5rem); }
  .splash-terminal { margin-top: 3rem; transform: none; }
  .splash-terminal pre { min-height: 0; padding: 1.2rem; font-size: .75rem; }
  .splash-strip-inner { grid-template-columns: 1fr; width: 100%; }
  .splash-strip-item, .splash-strip-item:nth-child(odd), .splash-strip-item:last-child { min-height: auto; border: 0; border-bottom: 1px solid var(--border); }
  .splash-section { width: min(100% - 2rem, 680px); padding: 4.5rem 0; }
  .splash-section-heading { display: block; margin-bottom: 2.5rem; }
  .splash-kicker { margin-bottom: 1rem; }
  .splash-workflow-row { grid-template-columns: 42px 1fr; gap: 1rem; }
  .splash-workflow-row p { grid-column: 2; }
  .splash-code-pair, .splash-paths { grid-template-columns: 1fr; }
  .splash-code-pair > div + div { border-left: 0; border-top: 1px solid var(--border); }
  .splash-path { min-height: auto; }
  .splash-path small { margin-bottom: 1.3rem; }
}

@media (max-width: 360px) {
  body > header { gap: .25rem; padding-inline: .4rem; }
  .theme-logo { flex-basis: min(var(--mpress-logo-width, 104px), 30vw); width: min(var(--mpress-logo-width, 104px), 30vw); height: 23px; }
  .search { flex-basis: 44px; width: 44px; }
  .search button { width: 44px; height: 44px; }
  .accessibility-select .utility-menu-trigger { width: 34px; min-width: 34px; }
}

@media (max-width: 520px) {
  body:not(.landing-page) > header .brand { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  html { scroll-behavior: auto; }
  *, *::before, *::after { scroll-behavior: auto !important; transition-duration: .01ms !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; }
}

@media (pointer: coarse) {
  .mpress-copy, .mpress-tutorial-reset, .mpress-calendar-nav, .mpress-api-pg-send { min-height: 44px; }
  .mpress-testimonials-dot, .mpress-tutorial-check { width: 44px; height: 44px; }
}
`

const defaultThemeJS = `
(() => {
  const doc = document.documentElement;
  const storage = {
    get(key) { try { return localStorage.getItem(key); } catch (_) { return null; } },
    set(key, value) { try { localStorage.setItem(key, value); } catch (_) {} },
    remove(key) { try { localStorage.removeItem(key); } catch (_) {} }
  };
  const shortcutPlatform = navigator.userAgentData?.platform || navigator.platform || '';
  const shortcutApplePlatform = /(Mac|iPhone|iPad|iPod)/i.test(shortcutPlatform);
  const shortcutTokens = value => String(value || 'None').split('+').map(token => token.trim()).filter(Boolean);
  const shortcutModifiers = new Set(['mod', 'meta', 'control', 'alt', 'shift']);
  const shortcutKey = value => shortcutTokens(value).find(token => !shortcutModifiers.has(token.toLowerCase())) || '';
  const shortcutMatches = (event, value) => {
    if (!value || String(value).toLowerCase() === 'none') return false;
    const tokens = shortcutTokens(value).map(token => token.toLowerCase());
    const wantsMod = tokens.includes('mod');
    const wantsMeta = tokens.includes('meta') || (wantsMod && shortcutApplePlatform);
    const wantsControl = tokens.includes('control') || (wantsMod && !shortcutApplePlatform);
    const wantedKey = shortcutKey(value).toLowerCase();
    const eventKey = event.key === ' ' ? 'space' : event.key.toLowerCase();
    return eventKey === wantedKey && event.metaKey === wantsMeta && event.ctrlKey === wantsControl && event.altKey === tokens.includes('alt') && event.shiftKey === tokens.includes('shift');
  };
  const shortcutARIA = value => shortcutTokens(value).map(token => {
    if (token === 'Mod') return shortcutApplePlatform ? 'Meta' : 'Control';
    return token;
  }).join('+');
  const shortcutLabel = value => shortcutTokens(value).map(token => {
    if (token === 'Mod') return shortcutApplePlatform ? '⌘' : 'Ctrl';
    if (token === 'Meta') return shortcutApplePlatform ? '⌘' : 'Meta';
    if (token === 'Control') return 'Ctrl';
    if (token === 'Alt') return shortcutApplePlatform ? '⌥' : 'Alt';
    if (token === 'Shift') return '⇧';
    return token;
  }).join(' ');
  const shortcutEditableTarget = target => Boolean(target?.closest?.('input, textarea, select, [contenteditable]:not([contenteditable="false"])'));
  window.mpressKeyboard = Object.freeze({matches: shortcutMatches, aria: shortcutARIA, label: shortcutLabel, editableTarget: shortcutEditableTarget});
  const theme = document.querySelector('#theme');
  const themeModes = ['system', 'dark', 'light'];
  const themeLabels = {system: 'System', dark: 'Dark', light: 'Light'};
  const stored = storage.get('mpress-theme');
  if (themeModes.includes(stored)) doc.dataset.theme = stored;
  if (!themeModes.includes(doc.dataset.theme)) doc.dataset.theme = 'system';
  const syncThemeButton = () => {
    if (!theme) return;
    const current = doc.dataset.theme;
    const next = themeModes[(themeModes.indexOf(current) + 1) % themeModes.length];
    theme.dataset.themeMode = current;
    theme.setAttribute('aria-label', 'Theme: ' + themeLabels[current] + '. Switch to ' + themeLabels[next]);
    theme.title = 'Theme: ' + themeLabels[current];
  };
  syncThemeButton();
  theme?.addEventListener('click', () => {
    const current = themeModes.includes(doc.dataset.theme) ? doc.dataset.theme : 'system';
    const next = themeModes[(themeModes.indexOf(current) + 1) % themeModes.length];
    doc.dataset.theme = next;
    storage.set('mpress-theme', next);
    syncThemeButton();
  });

  const menu = document.querySelector('#menu');
  const sidebar = document.querySelector('.sidebar');
  const mobileNavigation = window.matchMedia('(max-width: 760px)');
  const syncNavigation = () => {
    const closedMobile = mobileNavigation.matches && !document.body.classList.contains('nav-open');
    if (sidebar) {
      sidebar.toggleAttribute('inert', closedMobile);
      sidebar.setAttribute('aria-hidden', String(closedMobile));
    }
  };
  const closeNavigation = restoreFocus => {
    document.body.classList.remove('nav-open');
    menu?.setAttribute('aria-expanded', 'false');
    syncNavigation();
    if (restoreFocus) menu?.focus();
  };
  syncNavigation();
  menu?.addEventListener('click', () => {
    const open = document.body.classList.toggle('nav-open');
    menu.setAttribute('aria-expanded', String(open));
    syncNavigation();
    if (open) sidebar?.focus();
  });
  sidebar?.addEventListener('click', event => {
    if (event.target.closest('a') && window.innerWidth <= 760) {
      closeNavigation(false);
    }
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && document.body.classList.contains('nav-open')) closeNavigation(true);
  });
  document.addEventListener('click', event => {
    if (document.body.classList.contains('nav-open') && !event.target.closest('.sidebar') && !event.target.closest('#menu')) closeNavigation(true);
  });
  mobileNavigation.addEventListener?.('change', () => {
    if (!mobileNavigation.matches) closeNavigation(false);
    else syncNavigation();
  });

  const contributeDialog = document.querySelector('#mpress-contribute-dialog');
  if (contributeDialog) {
    const contributionCommand = contributeDialog.querySelector('[data-contribute-command]');
    const contributionManual = contributeDialog.querySelector('[data-contribute-manual]');
    const contributionPlatformLabel = contributeDialog.querySelector('[data-contribute-platform-label]');
    const contributionStatus = contributeDialog.querySelector('[data-contribute-status]');
	const contributionIntro = contributeDialog.querySelector('[data-contribute-intro]');
	const contributionChoices = contributeDialog.querySelector('[data-contribute-choices]');
	const contributionSetup = contributeDialog.querySelector('[data-contribute-setup]');
    const shellQuote = value => "'" + String(value).replaceAll("'", "'\"'\"'") + "'";
    const powerShellQuote = value => "'" + String(value).replaceAll("'", "''") + "'";
    const shellInstallerURL = new URL(contributeDialog.dataset.installerShell, location.href).href;
    const powerShellInstallerURL = new URL(contributeDialog.dataset.installerPowershell, location.href).href;
    const detectedPlatform = navigator.userAgentData?.platform || navigator.platform || navigator.userAgent || '';
    const contributionPlatform = /Windows|Win32|Win64/i.test(detectedPlatform) ? 'powershell' : 'shell';
    contributionPlatformLabel.textContent = contributionPlatform === 'powershell' ? 'Command for Windows PowerShell' : 'Command for macOS and Linux';
	const quickEditDataElement = contributeDialog.querySelector('#mpress-quick-edit-data');
	let quickEditData = null;
	try { quickEditData = quickEditDataElement ? JSON.parse(quickEditDataElement.textContent || 'null') : null; } catch (_) {}
	const quickEditKey = quickEditData ? 'mpress-quick-edit:' + quickEditData.source : '';
	const newDraft = () => ({version: 1, source: quickEditData.source, route: quickEditData.route, revision: quickEditData.revision, changes: []});
	let quickEditDraft = null;
	if (quickEditKey) {
	  try {
		const saved = JSON.parse(storage.get(quickEditKey) || 'null');
		if (saved?.version === 1 && saved.source === quickEditData.source && saved.revision === quickEditData.revision && Array.isArray(saved.changes)) quickEditDraft = saved;
	  } catch (_) {}
	}
	quickEditDraft ||= quickEditData ? newDraft() : null;
	const draftChanges = () => quickEditDraft?.changes || [];
	const saveQuickEditDraft = () => {
	  if (!quickEditKey || !quickEditDraft) return;
	  if (quickEditDraft.changes.length) storage.set(quickEditKey, JSON.stringify(quickEditDraft));
	  else storage.remove(quickEditKey);
	};
	let downloadedDraftName = '';
	let downloadedDraftJSON = '';
	let contributionGoal = '';
	const createDraftFileName = () => {
	  const identifier = globalThis.crypto?.randomUUID
		? globalThis.crypto.randomUUID().replaceAll('-', '').slice(0, 12)
		: Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
	  return 'mpress-draft-' + identifier + '.mpress-draft';
	};
	const downloadQuickEditDraft = () => {
	  if (!quickEditDraft?.changes.length) return '';
	  const portableDraft = {...quickEditDraft, changes: quickEditDraft.changes.map(({html, ...change}) => change)};
	  const json = JSON.stringify(portableDraft);
	  if (downloadedDraftName && downloadedDraftJSON === json) return downloadedDraftName;
	  downloadedDraftName = createDraftFileName();
	  downloadedDraftJSON = json;
	  const blobURL = URL.createObjectURL(new Blob([json + '\n'], {type: 'application/vnd.mpress.draft+json'}));
	  const download = document.createElement('a');
	  download.href = blobURL;
	  download.download = downloadedDraftName;
	  download.hidden = true;
	  document.body.append(download);
	  download.click();
	  download.remove();
	  setTimeout(() => URL.revokeObjectURL(blobURL), 1000);
	  return downloadedDraftName;
	};
	const contributionPageURL = (() => {
	  const pageURL = new URL(location.href);
	  pageURL.hash = '';
	  pageURL.searchParams.delete('contribute');
	  return pageURL.href;
	})();
	const contributionCommands = () => {
	  const shellURL = shellQuote(shellInstallerURL);
	  const shellDownload = shellInstallerURL.startsWith('https:')
		? "{ command -v curl >/dev/null 2>&1 && curl --proto '=https' --tlsv1.2 -fsSL " + shellURL + ' || wget --https-only -qO- ' + shellURL + '; }'
		: '{ command -v curl >/dev/null 2>&1 && curl -fsSL ' + shellURL + ' || wget -qO- ' + shellURL + '; }';
	  const shellArgs = shellQuote(contributionPageURL) + (downloadedDraftName ? ' ' + shellQuote(downloadedDraftName) : " ''") + (contributionGoal ? ' ' + shellQuote(contributionGoal) : '');
	  const powerShellArgs = powerShellQuote(contributionPageURL) + (downloadedDraftName ? ' ' + powerShellQuote(downloadedDraftName) : " ''") + (contributionGoal ? ' ' + powerShellQuote(contributionGoal) : '');
	  return {
		shell: shellDownload + ' | sh -s -- ' + shellArgs,
		powershell: '& ([scriptblock]::Create((irm ' + powerShellQuote(powerShellInstallerURL) + '))) ' + powerShellArgs
	  };
	};
	const refreshContributionCommand = () => {
	  if (!contributionCommand) return;
	  contributionCommand.textContent = contributionCommands()[contributionPlatform];
	  if (contributionManual) contributionManual.textContent = 'mpress contribute ' + (contributionPlatform === 'powershell' ? powerShellQuote(contributionPageURL) : shellQuote(contributionPageURL)) + (contributionGoal ? ' --goal ' + contributionGoal : '');
	};
	const showContributionChoices = () => {
	  contributionChoices.hidden = false;
	  contributionSetup.hidden = true;
	  contributionIntro.textContent = draftChanges().length
		? 'Your browser draft is safe. Continue editing here, or move the changes to a local M-Press checkout.'
		: quickEditData
		? 'Fix this page in the local development preview, translate the documentation, or open the complete project.'
		: 'Translate the documentation or open the complete project in a safe local contribution checkout.';
	  contributionStatus.textContent = draftChanges().length
		? draftChanges().length + ' draft change' + (draftChanges().length === 1 ? '' : 's') + ' saved only in this browser.'
		: 'Nothing is published until you choose to submit your work.';
	};
	const showContributionSetup = (goal = '') => {
	  contributionGoal = goal;
	  const draftFile = downloadQuickEditDraft();
	  contributionChoices.hidden = true;
	  contributionSetup.hidden = false;
	  contributionIntro.textContent = goal === 'translate'
		? 'Run this command once. M-Press will download the project, open the requested page, and take you directly to guided translation.'
		: draftChanges().length
		? 'Run this command to check out the source and apply your browser draft to the exact Markdown page.'
		: 'Run this command to check out the source and open this exact page in the M-Press development server.';
	  contributionStatus.textContent = goal === 'translate'
		? 'Your work stays in a private local branch until you choose to submit it.'
		: draftFile
		? 'Draft downloaded as ' + draftFile + '. Keep it in Downloads, then run the command.'
		: 'Nothing is published until you choose to submit your work.';
	  refreshContributionCommand();
	};
	const quickEditBar = document.querySelector('[data-quick-edit-bar]');
	const quickEditStatus = quickEditBar?.querySelector('[data-quick-edit-status]');
	const quickEditFormat = document.querySelector('[data-quick-edit-format]');
	const quickEditFormatButtons = [...(quickEditFormat?.querySelectorAll('[data-quick-edit-format-command]') || [])];
	const quickEditLinkForm = quickEditFormat?.querySelector('[data-quick-edit-link-form]');
	const quickEditLinkInput = quickEditLinkForm?.querySelector('[data-quick-edit-link-input]');
	const editableSegments = new Map();
	const editableRecords = new WeakMap();
	let lastEditedSegment = null;
	let activeEditable = null;
	let savedEditRange = null;
	const updateQuickEditStatus = () => {
	  if (!quickEditStatus) return;
	  const count = draftChanges().length;
	  quickEditStatus.textContent = count ? count + ' change' + (count === 1 ? '' : 's') + ' saved in this browser' : 'No changes yet';
	};
	const setDraftChange = change => {
	  if (!quickEditDraft) return;
	  downloadedDraftName = '';
	  downloadedDraftJSON = '';
	  const index = quickEditDraft.changes.findIndex(item => item.id === change.id);
	  if (change.markdown === change.original || (change.start === change.end && change.markdown === '')) {
		if (index >= 0) quickEditDraft.changes.splice(index, 1);
	  } else if (index >= 0) quickEditDraft.changes[index] = change;
	  else quickEditDraft.changes.push(change);
	  quickEditDraft.changes.sort((left, right) => left.start - right.start || left.end - right.end);
	  saveQuickEditDraft();
	  updateQuickEditStatus();
	};
	const safeQuickEditLink = value => {
	  const link = String(value || '').trim();
	  if (!link) return '';
	  if (link.startsWith('#') || link.startsWith('/') || link.startsWith('./') || link.startsWith('../')) return link;
	  try {
		const parsed = new URL(link, location.href);
		return ['http:', 'https:', 'mailto:', 'tel:'].includes(parsed.protocol) ? link : '';
	  } catch (_) { return ''; }
	};
	const inlineCodeMarkdown = value => {
	  const mark = String.fromCharCode(96);
	  const runs = String(value).match(new RegExp(mark + '+', 'g')) || [];
	  const delimiter = mark.repeat(Math.max(1, ...runs.map(run => run.length + 1)));
	  const padding = /^\s|\s$/.test(value) ? ' ' : '';
	  return delimiter + padding + value + padding + delimiter;
	};
	const markdownForQuickEditNode = node => {
	  if (node.nodeType === 3) return node.textContent || '';
	  if (node.nodeType !== 1) return '';
	  const content = [...node.childNodes].map(markdownForQuickEditNode).join('');
	  if (node.matches('strong, b')) return '**' + content + '**';
	  if (node.matches('em, i')) return '_' + content + '_';
	  if (node.matches('code')) return inlineCodeMarkdown(node.textContent || '');
	  if (node.matches('a')) {
		const link = safeQuickEditLink(node.getAttribute('href'));
		return link ? '[' + content + '](' + link.replaceAll(' ', '%20').replaceAll(')', '\\)') + ')' : content;
	  }
	  if (node.matches('br')) return '\n';
	  return content;
	};
	const quickEditMarkdown = element => [...element.childNodes].map(markdownForQuickEditNode).join('');
	const restoreQuickEditHTML = (element, html) => {
	  const template = document.createElement('template');
	  template.innerHTML = String(html || '');
	  [...template.content.querySelectorAll('*')].reverse().forEach(item => {
		if (!item.matches('strong, em, code, a')) {
		  item.replaceWith(document.createTextNode(item.textContent || ''));
		  return;
		}
		[...item.attributes].forEach(attribute => {
		  if (item.matches('a') && attribute.name === 'href') return;
		  item.removeAttribute(attribute.name);
		});
		if (item.matches('a')) {
		  const link = safeQuickEditLink(item.getAttribute('href'));
		  if (link) item.setAttribute('href', link);
		  else item.replaceWith(...item.childNodes);
		}
	  });
	  element.replaceChildren(template.content.cloneNode(true));
	};
	const selectionInEditable = () => {
	  const selection = document.getSelection();
	  if (!selection?.rangeCount || !activeEditable) return null;
	  const range = selection.getRangeAt(0);
	  return activeEditable.element.contains(range.startContainer) && activeEditable.element.contains(range.endContainer) ? range : null;
	};
	const formatTags = command => ({bold: 'strong, b', italic: 'em, i', code: 'code', link: 'a'}[command] || '');
	const formatAncestor = (node, command, root) => {
	  const selector = formatTags(command);
	  for (let item = node.nodeType === 1 ? node : node.parentElement; item && item !== root; item = item.parentElement) {
		if (selector && item.matches(selector)) return item;
	  }
	  return null;
	};
	let quickEditFormatFrame = 0;
	const placeQuickEditFormat = () => {
	  if (!quickEditFormat || quickEditFormat.hidden || !activeEditable) return;
	  if (quickEditFormatFrame) cancelAnimationFrame(quickEditFormatFrame);
	  quickEditFormatFrame = requestAnimationFrame(() => {
		quickEditFormatFrame = 0;
		const range = selectionInEditable() || savedEditRange;
		let rect = range?.getBoundingClientRect();
		if (!rect || (!rect.width && !rect.height)) rect = activeEditable.element.getBoundingClientRect();
		const half = quickEditFormat.offsetWidth / 2;
		const centre = Math.max(half + 10, Math.min(rect.left + rect.width / 2, window.innerWidth - half - 10));
		const below = rect.top < quickEditFormat.offsetHeight + 18;
		quickEditFormat.classList.toggle('is-below', below);
		quickEditFormat.style.left = centre + 'px';
		quickEditFormat.style.top = (below ? rect.bottom : rect.top) + 'px';
	  });
	};
	const showQuickEditFormat = () => {
	  if (!quickEditFormat) return;
	  quickEditFormat.hidden = false;
	  placeQuickEditFormat();
	};
	const restoreQuickEditSelection = range => {
	  if (!activeEditable || !range) return;
	  activeEditable.element.focus({preventScroll: true});
	  const selection = document.getSelection();
	  selection.removeAllRanges();
	  selection.addRange(range);
	  savedEditRange = range.cloneRange();
	  showQuickEditFormat();
	};
	const updateQuickEditFormatState = () => {
	  const range = selectionInEditable() || savedEditRange;
	  const valid = Boolean(activeEditable && range && !range.collapsed && activeEditable.element.contains(range.startContainer) && activeEditable.element.contains(range.endContainer));
	  quickEditFormatButtons.forEach(button => {
		const command = button.dataset.quickEditFormatCommand;
		const inherited = activeEditable && formatTags(command) ? activeEditable.element.parentElement?.closest(formatTags(command)) : null;
		button.disabled = !valid || Boolean(inherited);
		button.setAttribute('aria-pressed', String(Boolean(inherited || (valid && formatAncestor(range.startContainer, command, activeEditable.element)))));
	  });
	  placeQuickEditFormat();
	};
	const rememberQuickEditSelection = () => {
	  const range = selectionInEditable();
	  if (range) {
		savedEditRange = range.cloneRange();
		showQuickEditFormat();
	  }
	  updateQuickEditFormatState();
	};
	const syncQuickEditRecord = record => {
	  const body = quickEditMarkdown(record.element);
	  setDraftChange({
		id: record.id, start: record.start, end: record.end, original: record.original,
		markdown: body ? record.prefix + body : '', html: record.element.innerHTML
	  });
	};
	const insertQuickEditPlainText = (element, text) => {
	  const selection = document.getSelection();
	  if (!selection?.rangeCount) return;
	  const range = selection.getRangeAt(0);
	  if (!element.contains(range.startContainer) || !element.contains(range.endContainer)) return;
	  range.deleteContents();
	  const node = document.createTextNode(text);
	  range.insertNode(node);
	  range.setStartAfter(node);
	  range.collapse(true);
	  selection.removeAllRanges();
	  selection.addRange(range);
	};
	const openQuickEditLink = () => {
	  const range = selectionInEditable() || savedEditRange;
	  if (!activeEditable || !range || range.collapsed) {
		if (quickEditStatus) quickEditStatus.textContent = 'Select text before adding a link';
		return;
	  }
	  savedEditRange = range.cloneRange();
	  const existing = formatAncestor(range.startContainer, 'link', activeEditable.element);
	  quickEditLinkForm.hidden = false;
	  quickEditLinkInput.value = existing?.getAttribute('href') || '';
	  quickEditLinkInput.focus();
	  quickEditLinkInput.select();
	};
	const applyQuickEditFormat = command => {
	  if (command === 'link') { openQuickEditLink(); return; }
	  if (!activeEditable || !savedEditRange || savedEditRange.collapsed) return;
	  const range = savedEditRange.cloneRange();
	  if (!activeEditable.element.contains(range.startContainer) || !activeEditable.element.contains(range.endContainer)) return;
	  const selection = document.getSelection();
	  selection.removeAllRanges();
	  selection.addRange(range);
	  const existing = formatAncestor(range.startContainer, command, activeEditable.element);
	  if (existing && existing.contains(range.endContainer)) {
		const first = existing.firstChild;
		const last = existing.lastChild;
		const contents = document.createDocumentFragment();
		while (existing.firstChild) contents.append(existing.firstChild);
		existing.replaceWith(contents);
		if (first && last) {
		  range.setStartBefore(first);
		  range.setEndAfter(last);
		}
	  } else {
		const wrapper = document.createElement(command === 'bold' ? 'strong' : command === 'italic' ? 'em' : 'code');
		wrapper.append(range.extractContents());
		range.insertNode(wrapper);
		range.selectNodeContents(wrapper);
		selection.removeAllRanges();
		selection.addRange(range);
	  }
	  syncQuickEditRecord(activeEditable);
	  restoreQuickEditSelection(range);
	  updateQuickEditFormatState();
	};
	const applyQuickEditLink = value => {
	  const link = safeQuickEditLink(value);
	  if (!link || !activeEditable || !savedEditRange || savedEditRange.collapsed) {
		if (quickEditStatus) quickEditStatus.textContent = link ? 'Select text before adding a link' : 'Enter a valid link address';
		return false;
	  }
	  const range = savedEditRange.cloneRange();
	  const existing = formatAncestor(range.startContainer, 'link', activeEditable.element);
	  if (existing && existing.contains(range.endContainer)) existing.setAttribute('href', link);
	  else {
		const anchor = document.createElement('a');
		anchor.setAttribute('href', link);
		anchor.append(range.extractContents());
		range.insertNode(anchor);
		range.selectNodeContents(anchor);
	  }
	  syncQuickEditRecord(activeEditable);
	  quickEditLinkForm.hidden = true;
	  restoreQuickEditSelection(range);
	  updateQuickEditFormatState();
	  return true;
	};
	const bindQuickEditable = record => {
	  const {element} = record;
	  if (element.dataset.mpressEditBound) return;
	  element.dataset.mpressEditBound = 'true';
	  element.contentEditable = 'true';
	  element.spellcheck = true;
	  element.setAttribute('role', 'textbox');
	  element.setAttribute('aria-label', record.prefix ? 'Edit new paragraph' : 'Edit documentation text');
	  editableRecords.set(element, record);
	  element.addEventListener('pointerdown', () => {
		activeEditable = record;
		if (record.segment) lastEditedSegment = record.segment;
		showQuickEditFormat();
	  });
	  element.addEventListener('focus', () => {
		activeEditable = record;
		if (record.segment) lastEditedSegment = record.segment;
		showQuickEditFormat();
		rememberQuickEditSelection();
	  });
	  element.addEventListener('blur', () => requestAnimationFrame(() => {
		const focused = document.activeElement;
		if (!quickEditFormat?.contains(focused) && !focused?.matches?.('[data-mpress-edit-segment], [data-mpress-edit-insert]')) quickEditFormat.hidden = true;
	  }));
	  element.addEventListener('input', () => { syncQuickEditRecord(record); rememberQuickEditSelection(); });
	  element.addEventListener('mouseup', rememberQuickEditSelection);
	  element.addEventListener('keyup', rememberQuickEditSelection);
	  element.addEventListener('paste', event => {
		event.preventDefault();
		insertQuickEditPlainText(element, event.clipboardData?.getData('text/plain') || '');
		syncQuickEditRecord(record);
		rememberQuickEditSelection();
	  });
	  element.addEventListener('drop', event => event.preventDefault());
	  element.addEventListener('keydown', event => {
		const command = (event.ctrlKey || event.metaKey) && !event.shiftKey
		  ? ({b: 'bold', i: 'italic', k: 'link'}[event.key.toLowerCase()] || '') : '';
		if (command) { event.preventDefault(); applyQuickEditFormat(command); return; }
		if (event.key === 'Enter') event.preventDefault();
	  });
	};
	const installEditableSegments = () => {
	  if (!quickEditData || editableSegments.size) return;
	  const root = document.querySelector('#content article') || document.querySelector('#content');
	  if (!root) return;
	  const nodes = [];
	  const collectTextNodes = node => {
		if (!node) return;
		if (node.nodeType === 3) {
		  const parent = node.parentElement;
		  if (parent && (node.textContent || '').trim() && !parent.closest('a, button, code, pre, script, style, textarea, input, select, [data-mpress-no-edit]') && !parent.matches('article > h1:first-child')) nodes.push(node);
		  return;
		}
		if (node.nodeType !== 1) return;
		[...node.childNodes].forEach(collectTextNodes);
	  };
	  collectTextNodes(root);
	  const matches = new Map();
	  let nodeIndex = 0;
	  let offset = 0;
	  quickEditData.segments.forEach(segment => {
		for (let candidate = nodeIndex; candidate < nodes.length; candidate++) {
		  const searchOffset = candidate === nodeIndex ? offset : 0;
		  const index = (nodes[candidate].textContent || '').indexOf(segment.original, searchOffset);
		  if (index < 0) continue;
		  nodeIndex = candidate;
		  if (!matches.has(nodes[candidate])) matches.set(nodes[candidate], []);
		  matches.get(nodes[candidate]).push({segment, start: index, end: index + segment.original.length});
		  offset = index + segment.original.length;
		  return;
		}
	  });
	  matches.forEach((items, node) => {
		const text = node.textContent || '';
		const fragment = document.createDocumentFragment();
		let cursor = 0;
		items.sort((left, right) => left.start - right.start).forEach(item => {
		  if (item.start > cursor) fragment.append(document.createTextNode(text.slice(cursor, item.start)));
		  const span = document.createElement('span');
		  span.className = 'mpress-edit-segment';
		  span.dataset.mpressEditSegment = item.segment.id;
		  span.append(document.createTextNode(text.slice(item.start, item.end)));
		  fragment.append(span);
		  editableSegments.set(item.segment.id, {element: span, segment: item.segment});
		  cursor = item.end;
		});
		if (cursor < text.length) fragment.append(document.createTextNode(text.slice(cursor)));
		node.parentNode?.replaceChild(fragment, node);
	  });
	};
	const enterQuickEdit = () => {
	  if (!quickEditData) return;
	  installEditableSegments();
	  document.body.classList.add('mpress-quick-editing');
	  quickEditBar.hidden = false;
	  editableSegments.forEach(({element, segment}) => {
		const saved = draftChanges().find(change => change.id === segment.id);
		if (saved?.html) restoreQuickEditHTML(element, saved.html);
		else if (saved) element.textContent = saved.markdown;
		bindQuickEditable({element, segment, id: segment.id, start: segment.start, end: segment.end, original: segment.original, prefix: ''});
	  });
	  updateQuickEditStatus();
	  draftChanges().filter(change => change.id.startsWith('insert:')).forEach(change => insertQuickEditParagraph(change.id.slice(7)));
	  contributeDialog.close();
	  const firstChanged = draftChanges().find(change => editableSegments.has(change.id));
	  const target = firstChanged ? editableSegments.get(firstChanged.id)?.element : editableSegments.values().next().value?.element;
	  if (target) {
		activeEditable = editableRecords.get(target) || activeEditable;
		target.focus();
		showQuickEditFormat();
	  }
	};
	const insertQuickEditParagraph = base => {
	  if (!quickEditData) return;
	  installEditableSegments();
	  const group = quickEditData.segments.filter(segment => segment.id.replace(/-\d+$/, '') === base);
	  if (!group.length) return;
	  const end = Math.max(...group.map(segment => segment.end));
	  const id = 'insert:' + base;
	  const existing = document.querySelector('[data-mpress-edit-insert="' + CSS.escape(id) + '"]');
	  if (existing) { existing.focus(); return; }
	  const anchorElement = editableSegments.get(group.at(-1)?.id)?.element;
	  const block = anchorElement?.closest('p, li, h1, h2, h3, h4, h5, h6, blockquote') || anchorElement;
	  if (!block) return;
	  const paragraph = document.createElement('p');
	  paragraph.className = 'mpress-quick-edit-insert';
	  paragraph.dataset.mpressEditInsert = id;
	  const saved = draftChanges().find(change => change.id === id);
	  if (saved?.html) restoreQuickEditHTML(paragraph, saved.html);
	  else if (saved) paragraph.textContent = saved.markdown.replace(/^\n\n/, '');
	  bindQuickEditable({element: paragraph, segment: group.at(-1), id, start: end, end, original: '', prefix: '\n\n'});
	  block.insertAdjacentElement('afterend', paragraph);
	  paragraph.focus();
	};
	const addQuickEditParagraph = () => {
	  const anchor = lastEditedSegment || [...editableSegments.values()].at(-1)?.segment;
	  if (!anchor) return;
	  insertQuickEditParagraph(anchor.id.replace(/-\d+$/, ''));
	};
    document.querySelectorAll('[data-mpress-contribute]').forEach(trigger => trigger.addEventListener('click', () => {
	  showContributionChoices();
      contributeDialog.showModal();
	  (draftChanges().length ? contributeDialog.querySelector('[data-contribute-quick-edit]') : contributeDialog.querySelector('[data-contribute-quick-edit]'))?.focus();
    }));
	contributeDialog.querySelector('[data-contribute-quick-edit]')?.addEventListener('click', enterQuickEdit);
	contributeDialog.querySelector('[data-contribute-computer]')?.addEventListener('click', () => showContributionSetup(''));
	contributeDialog.querySelector('[data-contribute-translate]')?.addEventListener('click', () => showContributionSetup('translate'));
    contributeDialog.querySelector('[data-contribute-close]').addEventListener('click', () => contributeDialog.close());
    contributeDialog.addEventListener('click', event => {
      if (event.target === contributeDialog) contributeDialog.close();
    });
    contributeDialog.querySelector('[data-contribute-copy]').addEventListener('click', async () => {
      try {
		const draftFile = downloadQuickEditDraft();
		refreshContributionCommand();
		const command = contributionCommands()[contributionPlatform];
		await navigator.clipboard.writeText(command);
		contributionCommand.textContent = command;
		contributionStatus.textContent = draftFile
		  ? 'Command copied. Keep ' + draftFile + ' in Downloads, then run it to apply the draft.'
		  : 'Command copied. Paste it into a terminal to start contributing.';
      } catch (_) {
		contributionStatus.textContent = 'Select the command and copy it from this window.';
      }
    });
	quickEditFormatButtons.forEach(button => {
	  button.addEventListener('mousedown', event => event.preventDefault());
	  button.addEventListener('click', () => applyQuickEditFormat(button.dataset.quickEditFormatCommand));
	});
	quickEditLinkForm?.addEventListener('submit', event => {
	  event.preventDefault();
	  applyQuickEditLink(quickEditLinkInput.value);
	});
	quickEditLinkForm?.querySelector('[data-quick-edit-link-cancel]')?.addEventListener('click', () => {
	  quickEditLinkForm.hidden = true;
	  restoreQuickEditSelection(savedEditRange);
	});
	quickEditLinkInput?.addEventListener('keydown', event => {
	  if (event.key === 'Escape') {
		event.preventDefault();
		quickEditLinkForm.hidden = true;
		restoreQuickEditSelection(savedEditRange);
	  }
	});
	document.addEventListener('selectionchange', () => {
	  if (document.body.classList.contains('mpress-quick-editing')) rememberQuickEditSelection();
	});
	window.addEventListener('scroll', placeQuickEditFormat, {passive: true});
	window.addEventListener('resize', placeQuickEditFormat);
	quickEditBar?.querySelector('[data-quick-edit-add]')?.addEventListener('click', addQuickEditParagraph);
	quickEditBar?.querySelector('[data-quick-edit-discard]')?.addEventListener('click', () => {
	  if (quickEditKey) storage.remove(quickEditKey);
	  location.reload();
	});
	quickEditBar?.querySelector('[data-quick-edit-contribute]')?.addEventListener('click', () => {
	  showContributionSetup('page');
	  contributeDialog.showModal();
	  contributeDialog.querySelector('[data-contribute-copy]')?.focus();
	});
  }

  const toc = document.querySelector('.toc');
  const tocLinks = toc ? [...toc.querySelectorAll('a[href^="#"]')] : [];
  const tocItems = tocLinks.map(link => ({
    link,
    target: document.getElementById(decodeURIComponent(link.hash.slice(1)))
  })).filter(item => item.target);
  if (tocItems.length) {
    let tocFrame = 0;
    const updateTOC = () => {
      tocFrame = 0;
      const marker = window.scrollY + 112;
      let current = tocItems[0];
      for (const item of tocItems) {
        const top = item.target.getBoundingClientRect().top + window.scrollY;
        if (top <= marker) current = item;
        else break;
      }
      tocItems.forEach(item => {
        const active = item === current;
        item.link.classList.toggle('active', active);
        if (active) item.link.setAttribute('aria-current', 'location');
        else item.link.removeAttribute('aria-current');
      });
      if (toc.clientHeight > 0) {
        const tocRect = toc.getBoundingClientRect();
        const linkRect = current.link.getBoundingClientRect();
        if (linkRect.top < tocRect.top + 12) toc.scrollTop -= tocRect.top + 12 - linkRect.top;
        else if (linkRect.bottom > tocRect.bottom - 12) toc.scrollTop += linkRect.bottom - tocRect.bottom + 12;
      }
    };
    const requestTOCUpdate = () => {
      if (!tocFrame) tocFrame = requestAnimationFrame(updateTOC);
    };
    updateTOC();
    document.addEventListener('scroll', requestTOCUpdate, {passive: true});
    window.addEventListener('resize', requestTOCUpdate, {passive: true});
  }

  const utilityMenus = [...document.querySelectorAll('[data-utility-menu], [data-utility-panel]')];
  const menuButton = menu => document.querySelector('[popovertarget="' + menu.id + '"]');
  const menuIsOpen = menu => menu.matches(':popover-open');
  const positionUtilityMenu = menu => {
    const button = menuButton(menu);
    if (!button) return;
    const anchor = button.getBoundingClientRect();
    const inset = 12;
    const width = menu.offsetWidth;
    const height = menu.offsetHeight;
    const left = Math.max(inset, Math.min(window.innerWidth - width - inset, anchor.right - width));
    const below = anchor.bottom + 8;
    const top = below + height <= window.innerHeight - inset ? below : Math.max(inset, anchor.top - height - 8);
    menu.style.left = left + 'px';
    menu.style.top = top + 'px';
  };
  utilityMenus.forEach(menu => {
    const button = menuButton(menu);
    if (!button) return;
    menu.addEventListener('toggle', event => {
      const open = event.newState ? event.newState === 'open' : menuIsOpen(menu);
      button.setAttribute('aria-expanded', String(open));
      if (open) positionUtilityMenu(menu);
    });
    if (menu.hasAttribute('data-utility-menu')) {
      button.addEventListener('keydown', event => {
        if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return;
        event.preventDefault();
        if (!menuIsOpen(menu)) menu.showPopover();
        positionUtilityMenu(menu);
        const items = [...menu.querySelectorAll('[role="menuitem"]')];
        const target = event.key === 'ArrowUp' ? items[items.length - 1] : (menu.querySelector('[aria-current="true"]') || items[0]);
        target?.focus();
      });
      menu.addEventListener('keydown', event => {
        const items = [...menu.querySelectorAll('[role="menuitem"]')];
        const current = items.indexOf(document.activeElement);
        let next = current;
        if (event.key === 'ArrowDown') next = (current + 1) % items.length;
        else if (event.key === 'ArrowUp') next = (current - 1 + items.length) % items.length;
        else if (event.key === 'Home') next = 0;
        else if (event.key === 'End') next = items.length - 1;
        else if (event.key === 'Escape') {
          event.preventDefault();
          menu.hidePopover();
          button.focus();
          return;
        } else return;
        event.preventDefault();
        items[next]?.focus();
      });
    }
  });
  const repositionUtilityMenus = () => utilityMenus.forEach(menu => {
    if (menuIsOpen(menu)) positionUtilityMenu(menu);
  });
  window.addEventListener('resize', repositionUtilityMenus, {passive: true});
  document.addEventListener('scroll', repositionUtilityMenus, {passive: true, capture: true});
  const tabControllers = [...document.querySelectorAll('.mpress-tabs, .mpress-preview-tabs, .mpress-file-tabs')].map(group => ({
    group,
    syncKey: group.dataset.syncKey || '',
    tabs: [...group.querySelectorAll(':scope > [role="tablist"] > [role="tab"]')],
    panels: [...group.querySelectorAll(':scope > [role="tabpanel"]')],
    select: null,
  }));
  const tabValue = tab => (tab.dataset.syncValue || tab.textContent || '').trim();
  tabControllers.forEach(controller => {
    controller.select = (index, propagate = true) => {
      if (index < 0 || index >= controller.tabs.length) return;
      controller.tabs.forEach((item, itemIndex) => item.setAttribute('aria-selected', String(index === itemIndex)));
      controller.tabs.forEach((item, itemIndex) => item.tabIndex = index === itemIndex ? 0 : -1);
      controller.panels.forEach((panel, panelIndex) => panel.hidden = index !== panelIndex);
      if (!propagate || !controller.syncKey) return;
      const value = tabValue(controller.tabs[index]);
      tabControllers.forEach(peer => {
        if (peer === controller || peer.syncKey !== controller.syncKey) return;
        const peerIndex = peer.tabs.findIndex(tab => tabValue(tab) === value);
        if (peerIndex >= 0) peer.select(peerIndex, false);
      });
    };
  });
  tabControllers.forEach(controller => {
    controller.tabs.forEach((tab, index) => {
      tab.addEventListener('click', () => controller.select(index));
      tab.addEventListener('keydown', event => {
        let next = index;
        if (event.key === 'ArrowRight') next = (index + 1) % controller.tabs.length;
        else if (event.key === 'ArrowLeft') next = (index - 1 + controller.tabs.length) % controller.tabs.length;
        else if (event.key === 'Home') next = 0;
        else if (event.key === 'End') next = controller.tabs.length - 1;
        else return;
        event.preventDefault();
        controller.select(next);
        controller.tabs[next].focus();
      });
    });
  });

  const imageLightbox = document.querySelector('#mpress-image-lightbox');
  const imageLightboxContent = imageLightbox?.querySelector('[data-image-lightbox-content]');
  const imageLightboxClose = imageLightbox?.querySelector('[data-image-lightbox-close]');
  document.addEventListener('click', event => {
    const trigger = event.target.closest?.('.mpress-image-expand');
    if (!trigger || !imageLightbox || !imageLightboxContent) return;
    const content = trigger.querySelector('.mpress-image-expand-content');
    if (!content) return;
    imageLightboxContent.replaceChildren(content.cloneNode(true));
    imageLightbox.showModal();
    imageLightboxClose?.focus();
  });
  imageLightboxClose?.addEventListener('click', () => imageLightbox.close());
  imageLightbox?.addEventListener('click', event => {
    if (event.target === imageLightbox) imageLightbox.close();
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && imageLightbox?.open) imageLightbox.close();
  });
  imageLightbox?.addEventListener('close', () => imageLightboxContent?.replaceChildren());

  const blogFilters = [...document.querySelectorAll('[data-blog-filter]')];
  const blogPosts = [...document.querySelectorAll('[data-blog-tags]')];
  blogFilters.forEach(filter => filter.addEventListener('click', () => {
    const selected = filter.dataset.blogFilter;
    blogFilters.forEach(item => {
      const active = item === filter;
      item.classList.toggle('active', active);
      item.setAttribute('aria-pressed', String(active));
    });
    blogPosts.forEach(post => {
      const tags = (post.dataset.blogTags || '').split('|');
      post.hidden = Boolean(selected) && !tags.includes(selected);
    });
  }));

  document.querySelectorAll('.mpress-copy').forEach(button => {
    const defaultLabel = button.dataset.copyLabel || button.getAttribute('aria-label') || 'Copy';
    const copiedLabel = button.dataset.copiedLabel || 'Copied';
    const status = button.querySelector('[data-copy-status]');
    const reset = () => {
      button.classList.remove('is-copied');
      button.setAttribute('aria-label', defaultLabel);
      button.setAttribute('title', defaultLabel);
      if (status) status.textContent = defaultLabel;
    };
    const confirmCopy = () => {
      clearTimeout(button._mpressCopyTimer);
      button.classList.add('is-copied');
      button.setAttribute('aria-label', copiedLabel);
      button.setAttribute('title', copiedLabel);
      if (status) status.textContent = copiedLabel;
      button._mpressCopyTimer = setTimeout(reset, 3000);
    };
    button.addEventListener('click', async () => {
      const container = button.closest('.mpress-terminal, .mpress-codeframe');
      const source = container?.querySelector('pre');
      const value = source?.dataset.commands || source?.dataset.code || '';
      if (!value) return;
      try {
        await navigator.clipboard.writeText(value);
        confirmCopy();
      } catch (_) {
        const field = document.createElement('textarea');
        field.value = value;
        field.style.position = 'fixed';
        field.style.opacity = '0';
        document.body.append(field);
        field.select();
        const copied = document.execCommand('copy');
        field.remove();
        if (copied) confirmCopy();
        else {
          button.setAttribute('aria-label', 'Copy failed');
          if (status) status.textContent = 'Copy failed';
          clearTimeout(button._mpressCopyTimer);
          button._mpressCopyTimer = setTimeout(reset, 3000);
        }
      }
    });
  });

  document.querySelectorAll('.mpress-tutorial').forEach(tutorial => {
    const key = 'mpress-' + tutorial.dataset.tutorialId;
    const inputs = [...tutorial.querySelectorAll('input[type="checkbox"]')];
    let saved = [];
    try { saved = JSON.parse(storage.get(key) || '[]'); } catch (_) {}
    const update = () => {
      const done = inputs.filter(input => input.checked).length;
      tutorial.querySelector('.mpress-tutorial-progress-bar')?.style.setProperty('width', (inputs.length ? done / inputs.length * 100 : 0) + '%');
      const count = tutorial.querySelector('.mpress-tutorial-count');
      if (count) count.textContent = done + '/' + inputs.length + ' completed';
      inputs.forEach(input => input.closest('.mpress-tutorial-step')?.classList.toggle('completed', input.checked));
      storage.set(key, JSON.stringify(inputs.filter(input => input.checked).map(input => input.dataset.stepId)));
    };
    inputs.forEach(input => {
      input.checked = saved.includes(input.dataset.stepId);
      input.addEventListener('change', update);
    });
    tutorial.querySelector('.mpress-tutorial-reset')?.addEventListener('click', () => {
      inputs.forEach(input => input.checked = false);
      update();
    });
    update();
  });

  document.querySelectorAll('.mpress-testimonials').forEach(carousel => {
    const items = [...carousel.querySelectorAll('.mpress-testimonial')];
    const dots = [...carousel.querySelectorAll('.mpress-testimonials-dot')];
    let current = 0;
    const show = index => {
      current = index;
      items.forEach((item, i) => {
        item.classList.toggle('active', i === index);
        item.setAttribute('aria-hidden', String(i !== index));
      });
      dots.forEach((dot, i) => i === index ? dot.setAttribute('aria-current', 'true') : dot.removeAttribute('aria-current'));
    };
    dots.forEach((dot, index) => dot.addEventListener('click', () => show(index)));
    const delay = Number(carousel.dataset.autoplay);
    if (items.length > 1 && delay > 0 && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      let timer = null;
      const stop = () => { if (timer !== null) clearInterval(timer); timer = null; };
      const start = () => { if (timer === null) timer = setInterval(() => show((current + 1) % items.length), delay); };
      start();
      carousel.addEventListener('mouseenter', stop);
      carousel.addEventListener('mouseleave', start);
      carousel.addEventListener('focusin', stop);
      carousel.addEventListener('focusout', start);
    }
  });

  document.querySelectorAll('[data-carousel]').forEach(carousel => {
    const slides = [...carousel.querySelectorAll('[data-carousel-slide]')];
    const dots = [...carousel.querySelectorAll('[data-carousel-dot]')];
    const status = carousel.querySelector('[data-carousel-status]');
    let current = 0;
    const show = index => {
      current = (index + slides.length) % slides.length;
      slides.forEach((slide, slideIndex) => { slide.hidden = slideIndex !== current; });
      dots.forEach((dot, dotIndex) => dotIndex === current ? dot.setAttribute('aria-current', 'true') : dot.removeAttribute('aria-current'));
      if (status) status.textContent = (current + 1) + ' / ' + slides.length;
    };
    carousel.querySelector('[data-carousel-previous]')?.addEventListener('click', () => show(current - 1));
    carousel.querySelector('[data-carousel-next]')?.addEventListener('click', () => show(current + 1));
    dots.forEach(dot => dot.addEventListener('click', () => show(Number(dot.dataset.carouselDot))));
  });

  const audienceBlocks = [...document.querySelectorAll('.mpress-audience[data-audience]')];
  if (audienceBlocks.length) {
    const roles = [...new Set(audienceBlocks.map(block => block.dataset.audience))].sort();
    const selector = document.createElement('div');
    selector.className = 'mpress-audience-selector';
    selector.innerHTML = '<label class="mpress-audience-label">Audience: <select class="mpress-audience-select" aria-label="Select audience role"><option value="all">All</option></select></label>';
    const select = selector.querySelector('select');
    roles.forEach(role => select.add(new Option(role.charAt(0).toUpperCase() + role.slice(1), role)));
    const saved = storage.get('mpress-audience-role');
    select.value = saved && (saved === 'all' || roles.includes(saved)) ? saved : 'all';
    const apply = () => {
      audienceBlocks.forEach(block => block.classList.toggle('mpress-visible', select.value === 'all' || block.dataset.audience === select.value));
      storage.set('mpress-audience-role', select.value);
    };
    select.addEventListener('change', apply);
    audienceBlocks[0].before(selector);
    apply();
  }

  const conditions = [...document.querySelectorAll('.mpress-conditional[data-param]')];
  const params = new URLSearchParams(location.search);
  conditions.forEach(block => {
    const name = block.dataset.param;
    const fromURL = params.get(name);
    if (fromURL) storage.set('mpress-param-' + name, fromURL);
    const value = fromURL || storage.get('mpress-param-' + name) || '';
    block.classList.toggle('mpress-visible', block.dataset.value === value || (!value && block.dataset.default === 'true'));
  });

  const reactiveInputs = [...document.querySelectorAll('[data-reactive]')];
  const reactiveOutputs = [...document.querySelectorAll('.mpress-reactive-computed')];
  if (reactiveInputs.length && reactiveOutputs.length) {
    const store = {};
    const read = input => input.type === 'number' || input.type === 'range' ? Number(input.value) : input.value;
    reactiveInputs.forEach(input => store[input.dataset.reactive] = read(input));
    const evaluate = () => reactiveOutputs.forEach(output => {
      const target = output.querySelector('.mpress-reactive-computed-value');
      try {
        const names = Object.keys(store);
        let result = Function(...names, 'return (' + output.dataset.reactiveExpr + ')')(...names.map(name => store[name]));
        const format = output.dataset.reactiveFormat || '';
        const match = format.match(/%(?:\.(\d+))?([df])/);
        if (match) {
          const value = match[2] === 'd' ? Math.round(Number(result)) : Number(result).toFixed(Number(match[1] || 0));
          result = format.replace(match[0], value);
        }
        target.textContent = String(result);
      } catch (_) { target.textContent = 'Error'; }
    });
    reactiveInputs.forEach(input => input.addEventListener('input', () => {
      store[input.dataset.reactive] = read(input);
      const display = document.querySelector('[data-reactive-display="' + CSS.escape(input.dataset.reactive) + '"]');
      if (display) display.textContent = input.value;
      evaluate();
    }));
    evaluate();
  }

  const tableCollator = new Intl.Collator(undefined, {numeric: true, sensitivity: 'base'});
  document.querySelectorAll('.mpress-data-table').forEach(root => {
    const table = root.querySelector('table');
    const body = table?.tBodies[0];
    if (!table || !body) return;
    const rows = [...body.querySelectorAll('[data-table-row]')];
    const query = root.querySelector('[data-table-query]');
    const filters = [...root.querySelectorAll('[data-table-filter-column]')];
    const sortButtons = [...root.querySelectorAll('[data-table-sort-column]')];
    const status = root.querySelector('.mpress-table-status');
    const pageStatus = root.querySelector('[data-table-page-status]');
    const previousPage = root.querySelector('[data-table-page-prev]');
    const nextPage = root.querySelector('[data-table-page-next]');
    const paginated = root.dataset.tablePaginate === 'true';
    const pageSize = Math.max(1, Number(root.dataset.tablePageSize) || 10);
    let sortColumn = -1;
    let sortDirection = 'none';
    let currentPage = 1;

    const cellValue = (row, column) => row.cells[column]?.dataset.tableValue || row.cells[column]?.textContent?.trim().toLowerCase() || '';
    const apply = () => {
      const needle = query?.value.trim().toLowerCase() || '';
      const matching = rows.filter(row => {
        const matchesSearch = !needle || [...row.cells].some(cell => (cell.dataset.tableValue || cell.textContent || '').toLowerCase().includes(needle));
        const matchesFilters = filters.every(filter => !filter.dataset.tableFilterValue || cellValue(row, Number(filter.dataset.tableFilterColumn)) === filter.dataset.tableFilterValue);
        return matchesSearch && matchesFilters;
      });
      if (sortColumn >= 0) {
        [...rows].sort((left, right) => {
          const compared = tableCollator.compare(cellValue(left, sortColumn), cellValue(right, sortColumn));
          if (compared !== 0) return sortDirection === 'ascending' ? compared : -compared;
          return Number(left.dataset.tableIndex) - Number(right.dataset.tableIndex);
        }).forEach(row => body.append(row));
      }
      const matchingRows = new Set(matching);
      const orderedMatches = [...body.querySelectorAll('[data-table-row]')].filter(row => matchingRows.has(row));
      const totalPages = paginated ? Math.max(1, Math.ceil(orderedMatches.length / pageSize)) : 1;
      currentPage = Math.min(currentPage, totalPages);
      const start = paginated ? (currentPage - 1) * pageSize : 0;
      const end = paginated ? Math.min(start + pageSize, orderedMatches.length) : orderedMatches.length;
      const visible = new Set(orderedMatches.slice(start, end));
      rows.forEach(row => row.hidden = !visible.has(row));
      if (status) {
        if (!paginated) status.textContent = orderedMatches.length === rows.length ? rows.length + (rows.length === 1 ? ' row' : ' rows') : orderedMatches.length + ' of ' + rows.length + ' rows';
        else if (!orderedMatches.length) status.textContent = 'No matching rows';
        else status.textContent = (start + 1) + '–' + end + ' of ' + orderedMatches.length + (orderedMatches.length === rows.length ? ' rows' : ' matching rows');
      }
      if (pageStatus) pageStatus.textContent = 'Page ' + currentPage + ' of ' + totalPages;
      if (previousPage) previousPage.disabled = currentPage <= 1;
      if (nextPage) nextPage.disabled = currentPage >= totalPages;
    };

    query?.addEventListener('input', () => { currentPage = 1; apply(); });
    filters.forEach(filter => filter.querySelectorAll('[data-table-filter-value]').forEach(choice => choice.addEventListener('click', () => {
      const value = choice.dataset.tableFilterValue || '';
      const label = filter.dataset.tableFilterLabel || 'column';
      filter.dataset.tableFilterValue = value;
      filter.querySelectorAll('[data-table-filter-value]').forEach(item => item.setAttribute('aria-current', String(item === choice)));
      const trigger = document.querySelector('[popovertarget="' + filter.id + '"]');
      trigger?.classList.toggle('active', Boolean(value));
      trigger?.setAttribute('aria-label', value ? 'Filter ' + label + ': ' + choice.textContent.trim() : 'Filter ' + label);
      currentPage = 1;
      filter.hidePopover?.();
      apply();
    })));
    sortButtons.forEach(button => button.addEventListener('click', () => {
      const column = Number(button.dataset.tableSortColumn);
      const heading = button.closest('th');
      const current = heading?.getAttribute('aria-sort') || 'none';
      sortColumn = column;
      sortDirection = current === 'ascending' ? 'descending' : 'ascending';
      sortButtons.forEach(item => item.closest('th')?.setAttribute('aria-sort', 'none'));
      heading?.setAttribute('aria-sort', sortDirection);
      currentPage = 1;
      apply();
    }));
    previousPage?.addEventListener('click', () => { if (currentPage > 1) { currentPage--; apply(); } });
    nextPage?.addEventListener('click', () => { currentPage++; apply(); });
    apply();
  });

  document.querySelectorAll('.mpress-explained').forEach((panel, panelIndex) => {
    const lines = [...panel.querySelectorAll('.mpress-explained-line[data-ref]')];
    const explanations = new Map([...panel.querySelectorAll('.mpress-explained-para[data-ref]')].map(item => [item.dataset.ref, item]));
    if (!lines.length || !explanations.size) return;

    const tooltip = document.createElement('div');
    const tooltipRef = document.createElement('span');
    const tooltipCopy = document.createElement('span');
    tooltip.className = 'mpress-explained-tooltip';
    tooltip.id = 'mpress-explained-tooltip-' + panelIndex;
    tooltip.role = 'tooltip';
    tooltip.hidden = true;
    tooltipRef.className = 'mpress-explained-ref';
    tooltipCopy.className = 'mpress-explained-tooltip-copy';
    tooltip.append(tooltipRef, tooltipCopy);
    document.body.append(tooltip);

    let activeLine = null;
    const explanationText = (line, item) => {
      if (line.dataset.explanation?.trim()) return line.dataset.explanation.trim();
      const copy = item.querySelector('.mpress-explained-copy')?.textContent?.trim();
      if (copy) return copy;
      const fallback = item.cloneNode(true);
      fallback.querySelector('.mpress-explained-ref')?.remove();
      return fallback.textContent?.trim() || '';
    };
    const place = (x, y) => {
      const gap = 16;
      const margin = 12;
      const box = tooltip.getBoundingClientRect();
      let left = x + gap;
      let top = y + gap;
      if (left + box.width > window.innerWidth - margin) left = x - box.width - gap;
      if (top + box.height > window.innerHeight - margin) top = y - box.height - gap;
      tooltip.style.left = Math.max(margin, Math.min(left, window.innerWidth - box.width - margin)) + 'px';
      tooltip.style.top = Math.max(margin, Math.min(top, window.innerHeight - box.height - margin)) + 'px';
    };
    const show = (line, x, y) => {
      const explanation = explanations.get(line.dataset.ref);
      if (!explanation) return;
      lines.forEach(item => item.classList.toggle('mpress-explained-active', item === line));
      activeLine = line;
      tooltipRef.textContent = line.dataset.ref;
      tooltipCopy.textContent = explanationText(line, explanation);
      tooltip.hidden = false;
      tooltip.classList.add('mpress-explained-tooltip-visible');
      place(x, y);
    };
    const showAtLine = line => {
      const box = line.getBoundingClientRect();
      show(line, box.right, box.top + Math.min(box.height / 2, 16));
    };
    const hide = () => {
      lines.forEach(item => item.classList.remove('mpress-explained-active'));
      activeLine = null;
      tooltip.classList.remove('mpress-explained-tooltip-visible');
      tooltip.hidden = true;
    };

    lines.forEach(line => {
      const explanation = explanations.get(line.dataset.ref);
      if (!explanation) return;
      const descriptionID = 'mpress-explained-' + panelIndex + '-description-' + line.dataset.ref;
      explanation.id = descriptionID;
      line.setAttribute('aria-describedby', descriptionID);
      line.addEventListener('mouseenter', event => show(line, event.clientX, event.clientY));
      line.addEventListener('mousemove', event => place(event.clientX, event.clientY));
      line.addEventListener('mouseleave', () => {
        if (document.activeElement !== line) hide();
      });
      line.addEventListener('focus', () => showAtLine(line));
      line.addEventListener('click', () => showAtLine(line));
      line.addEventListener('blur', hide);
      line.addEventListener('keydown', event => {
        if (event.key === 'Escape') {
          hide();
          line.blur();
        }
      });
    });
    panel.classList.add('mpress-explained-enhanced');
    window.addEventListener('scroll', hide, {passive: true});
    window.addEventListener('resize', hide);
    document.addEventListener('pointerdown', event => {
      if (activeLine && !panel.contains(event.target)) hide();
    });
  });

  document.querySelectorAll('.mpress-api-playground').forEach(playground => {
    const url = playground.querySelector('.mpress-api-pg-url-input');
    const server = playground.querySelector('.mpress-api-pg-server-select');
    server?.addEventListener('change', () => url.value = server.value.replace(/\/$/, '') + playground.dataset.path);
    playground.querySelector('.mpress-api-pg-send')?.addEventListener('click', async event => {
      const button = event.currentTarget;
      const responsePanel = playground.querySelector('.mpress-api-pg-response');
      const responseBody = responsePanel.querySelector('code');
      const status = responsePanel.querySelector('.mpress-api-pg-response-status');
      const timing = responsePanel.querySelector('.mpress-api-pg-response-time');
      const headers = {};
      playground.querySelectorAll('.mpress-api-pg-header-row').forEach(row => {
        const fields = row.querySelectorAll('input');
        if (fields[0]?.value) headers[fields[0].value] = fields[1]?.value || '';
      });
      const options = {method: playground.dataset.method, headers};
      const body = playground.querySelector('.mpress-api-pg-request')?.value;
      if (body && !['GET', 'HEAD'].includes(options.method)) options.body = body;
      button.disabled = true;
      const started = performance.now();
      try {
        const response = await fetch(url.value, options);
        const text = await response.text();
        status.textContent = response.status + ' ' + response.statusText;
        timing.textContent = Math.round(performance.now() - started) + ' ms';
        try { responseBody.textContent = JSON.stringify(JSON.parse(text), null, 2); } catch (_) { responseBody.textContent = text; }
      } catch (error) {
        status.textContent = 'Request failed';
        responseBody.textContent = error.message;
      }
      responsePanel.hidden = false;
      button.disabled = false;
    });
  });

  document.querySelectorAll('.mpress-calendar').forEach(calendar => {
    const title = calendar.querySelector('.mpress-calendar-title');
    const grid = calendar.querySelector('.mpress-calendar-grid');
    const localDate = date => [date.getFullYear(), String(date.getMonth() + 1).padStart(2, '0'), String(date.getDate()).padStart(2, '0')].join('-');
    const markToday = () => {
      const today = localDate(new Date());
      grid.querySelectorAll('.mpress-calendar-cell[data-date]').forEach(cell => cell.classList.toggle('today', cell.dataset.date === today));
    };
    const draw = (year, month) => {
      calendar.dataset.year = year;
      calendar.dataset.month = month + 1;
      title.textContent = new Intl.DateTimeFormat(undefined, {month: 'long', year: 'numeric'}).format(new Date(year, month, 1));
      [...grid.querySelectorAll('.mpress-calendar-cell')].forEach(cell => cell.remove());
      const first = new Date(year, month, 1);
      const offset = (first.getDay() + 6) % 7;
      const count = new Date(year, month + 1, 0).getDate();
      for (let i = 0; i < offset; i++) {
        const cell = document.createElement('div');
        cell.className = 'mpress-calendar-cell empty';
        grid.append(cell);
      }
      for (let day = 1; day <= count; day++) {
        const cell = document.createElement('div');
        cell.className = 'mpress-calendar-cell';
        cell.dataset.date = [year, String(month + 1).padStart(2, '0'), String(day).padStart(2, '0')].join('-');
        const date = document.createElement('span');
        date.className = 'mpress-calendar-date';
        date.textContent = String(day);
        cell.append(date);
        grid.append(cell);
      }
      markToday();
    };
    calendar.querySelectorAll('.mpress-calendar-nav').forEach(button => button.addEventListener('click', () => {
      const date = new Date(Number(calendar.dataset.year), Number(calendar.dataset.month) - 1 + (button.dataset.dir === 'next' ? 1 : -1), 1);
      draw(date.getFullYear(), date.getMonth());
    }));
    markToday();
  });

  let ddlbID = 0;
  const ddlbChevron = '<svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-chevron-down" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg>';
  const ddlbCheck = '<svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-check" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>';
  const closeDDLB = except => document.querySelectorAll('.mpress-ddlb[data-open="true"]').forEach(wrapper => {
    if (wrapper === except) return;
    const trigger = wrapper.querySelector('.mpress-ddlb-trigger');
    const menu = wrapper.querySelector('.mpress-ddlb-menu');
    wrapper.dataset.open = 'false';
    trigger?.setAttribute('aria-expanded', 'false');
    if (menu?.matches(':popover-open')) menu.hidePopover();
  });
  const installDDLBs = container => {
    if (!container?.querySelectorAll) return;
    const selects = [];
    if (container.matches?.('select:not([multiple]):not([data-mpress-ddlb-ready])')) selects.push(container);
    selects.push(...container.querySelectorAll('select:not([multiple]):not([data-mpress-ddlb-ready])'));
    selects.forEach(select => {
      const wrapper = document.createElement('span');
      const trigger = document.createElement('button');
      const value = document.createElement('span');
      const menu = document.createElement('div');
      const id = 'mpress-ddlb-' + (++ddlbID);
      const menuID = id + '-listbox';
      const label = select.labels?.[0];
      const directLabel = label?.querySelector(':scope > span');
      const labelText = select.getAttribute('aria-label') || directLabel?.textContent?.trim() || select.name || 'Choose an option';
      wrapper.className = 'mpress-ddlb';
      wrapper.dataset.open = 'false';
      trigger.type = 'button';
      trigger.id = id;
      trigger.className = 'mpress-ddlb-trigger';
      trigger.setAttribute('role', 'combobox');
      trigger.setAttribute('aria-haspopup', 'listbox');
      trigger.setAttribute('aria-expanded', 'false');
      trigger.setAttribute('aria-controls', menuID);
      value.className = 'mpress-ddlb-value';
      menu.id = menuID;
      menu.className = 'mpress-ddlb-menu';
      menu.setAttribute('role', 'listbox');
      menu.setAttribute('popover', 'auto');
      select.before(wrapper);
      wrapper.append(select, trigger, menu);
      select.dataset.mpressDdlbReady = 'true';
      select.classList.add('mpress-ddlb-native');
      select.tabIndex = -1;
      select.setAttribute('aria-hidden', 'true');
      if (label?.htmlFor === select.id) label.removeAttribute('for');
      const optionButtons = () => [...menu.querySelectorAll('.mpress-ddlb-option')];
      const position = () => {
        const anchor = trigger.getBoundingClientRect();
        const inset = 12;
        const availableBelow = window.innerHeight - anchor.bottom - inset;
        const availableAbove = anchor.top - inset;
        const height = Math.min(menu.scrollHeight, 288);
        const below = availableBelow >= Math.min(height, 160) || availableBelow >= availableAbove;
        const top = below ? anchor.bottom + 7 : Math.max(inset, anchor.top - height - 7);
        menu.style.left = Math.max(inset, Math.min(window.innerWidth - anchor.width - inset, anchor.left)) + 'px';
        menu.style.top = top + 'px';
        menu.style.width = anchor.width + 'px';
      };
	  wrapper.mpressPositionDDLB = position;
      const sync = () => {
        const selected = select.options[select.selectedIndex];
        value.textContent = selected?.textContent || '';
        trigger.disabled = select.disabled;
        trigger.setAttribute('aria-label', labelText + ': ' + value.textContent);
        optionButtons().forEach((button, index) => button.setAttribute('aria-selected', String(index === select.selectedIndex)));
      };
      const choose = index => {
        if (select.options[index]?.disabled) return;
        select.selectedIndex = index;
        select.dispatchEvent(new Event('input', {bubbles: true}));
        select.dispatchEvent(new Event('change', {bubbles: true}));
        trigger.removeAttribute('aria-invalid');
        sync();
        closeDDLB();
        trigger.focus();
      };
      [...select.options].forEach((option, index) => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'mpress-ddlb-option';
        item.setAttribute('role', 'option');
        item.disabled = option.disabled;
        const text = document.createElement('span');
        text.textContent = option.textContent;
        item.append(text);
        item.insertAdjacentHTML('beforeend', ddlbCheck);
        item.addEventListener('click', event => {
          event.preventDefault();
          choose(index);
        });
        item.addEventListener('keydown', event => {
          const items = optionButtons().filter(button => !button.disabled);
          const current = items.indexOf(item);
          let destination = -1;
          if (event.key === 'ArrowDown') destination = Math.min(items.length - 1, current + 1);
          else if (event.key === 'ArrowUp') destination = Math.max(0, current - 1);
          else if (event.key === 'Home') destination = 0;
          else if (event.key === 'End') destination = items.length - 1;
          else if (event.key === 'Escape') {
            event.preventDefault();
            closeDDLB();
            trigger.focus();
            return;
          } else if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            choose(index);
            return;
          } else return;
          event.preventDefault();
          items[destination]?.focus();
        });
        menu.append(item);
      });
      trigger.append(value);
      trigger.insertAdjacentHTML('beforeend', ddlbChevron);
      const open = () => {
        if (trigger.disabled) return;
        closeDDLB(wrapper);
        wrapper.dataset.open = 'true';
        trigger.setAttribute('aria-expanded', 'true');
        menu.showPopover();
        position();
        const items = optionButtons();
        (items[select.selectedIndex] || items.find(item => !item.disabled))?.focus();
      };
      trigger.addEventListener('click', event => {
        event.preventDefault();
        if (wrapper.dataset.open === 'true') {
          closeDDLB();
          trigger.focus();
        } else open();
      });
      trigger.addEventListener('keydown', event => {
        if (!['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) return;
        event.preventDefault();
        open();
      });
      menu.addEventListener('toggle', event => {
        if (event.newState === 'closed') {
          wrapper.dataset.open = 'false';
          trigger.setAttribute('aria-expanded', 'false');
        }
      });
      select.addEventListener('input', sync);
      select.addEventListener('change', sync);
      select.addEventListener('invalid', event => {
        event.preventDefault();
        trigger.setAttribute('aria-invalid', 'true');
        trigger.focus();
      });
      sync();
    });
  };
  installDDLBs(document);
  const ddlbObserver = new MutationObserver(records => records.forEach(record => record.addedNodes.forEach(node => {
    if (node.nodeType === 1) installDDLBs(node);
  })));
  ddlbObserver.observe(document.body, {childList: true, subtree: true});
  document.addEventListener('pointerdown', event => {
    if (!event.target.closest('.mpress-ddlb')) closeDDLB();
  });
  document.addEventListener('focusin', event => {
    if (!event.target.closest('.mpress-ddlb')) closeDDLB();
  });
  const repositionDDLBs = () => document.querySelectorAll('.mpress-ddlb[data-open="true"]').forEach(wrapper => wrapper.mpressPositionDDLB?.());
  window.addEventListener('resize', repositionDDLBs, {passive: true});
  document.addEventListener('scroll', repositionDDLBs, {passive: true, capture: true});

  const query = document.querySelector('#search');
  if (query) {
    const shortcut = document.querySelector('.search-shortcut');
    const searchShortcut = document.body.dataset.shortcutSearch || 'Mod+K';
    const searchPlaceholder = document.body.dataset.searchPlaceholder || 'Search documentation';
    const searchMaxResults = Math.max(4, Math.min(24, Number.parseInt(document.body.dataset.searchMaxResults || '12', 10) || 12));
    const rememberRecent = document.body.dataset.searchRecent !== 'false';
    if (shortcut) {
      shortcut.textContent = shortcutLabel(searchShortcut);
      shortcut.hidden = searchShortcut === 'None';
    }
    if (searchShortcut !== 'None') query.setAttribute('aria-keyshortcuts', shortcutARIA(searchShortcut));
    let index = [];
    let indexPromise;
    let active = -1;
    let matches = [];
    let overlay;
    let dialog;
    let overlayInput;
    let resultList;
    let resultCount;
    let preview;
    let debounce;
    let suppressSearchFocus = false;
    const recentKey = 'mpress-recent-searches';
    const escapeHTML = value => String(value || '').replace(/[&<>"']/g, character => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[character]));
    const escapePattern = value => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const normalise = value => String(value || '').normalize('NFKD').replace(/[\u0300-\u036f]/g, '').toLowerCase();
    const searchIcon = name => {
      const paths = name === 'clock'
        ? '<circle cx="12" cy="12" r="9"></circle><path d="M12 7v5l3 2"></path>'
        : '<circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path>';
      return '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-' + name + '" aria-hidden="true">' + paths + '</svg>';
    };
    const loadIndex = () => {
      if (indexPromise) return indexPromise;
      indexPromise = fetch(document.body.dataset.search)
        .then(response => {
          if (!response.ok) throw new Error('Search index unavailable');
          return response.json();
        })
        .then(value => index = Array.isArray(value) ? value : [])
        .catch(() => index = []);
      return indexPromise;
    };
    const readRecent = () => {
      if (!rememberRecent) return [];
      try {
        const value = JSON.parse(localStorage.getItem(recentKey) || '[]');
        return Array.isArray(value) ? value.filter(item => typeof item === 'string').slice(0, 5) : [];
      } catch (_) {
        return [];
      }
    };
    const saveRecent = value => {
      if (!rememberRecent) return;
      const clean = value.trim();
      if (!clean) return;
      try {
        localStorage.setItem(recentKey, JSON.stringify([clean, ...readRecent().filter(item => item !== clean)].slice(0, 5)));
      } catch (_) {}
    };
    const levenshtein = (left, right) => {
      if (!left.length) return right.length;
      if (!right.length) return left.length;
      let previous = Array.from({length: left.length + 1}, (_, index) => index);
      for (let row = 1; row <= right.length; row++) {
        const current = [row];
        for (let column = 1; column <= left.length; column++) {
          const cost = left[column - 1] === right[row - 1] ? 0 : 1;
          current[column] = Math.min(current[column - 1] + 1, previous[column] + 1, previous[column - 1] + cost);
        }
        previous = current;
      }
      return previous[left.length];
    };
    const fuzzyMatch = (word, candidate) => {
      if (candidate.includes(word)) return true;
      if (word.length < 4) return false;
      const shorter = Math.min(word.length, candidate.length);
      if (shorter >= 6) {
        let shared = 0;
        while (shared < shorter && word[shared] === candidate[shared]) shared++;
        if (shared >= Math.max(6, Math.min(8, shorter - 1))) return true;
      }
      return levenshtein(word, candidate) <= (word.length >= 6 ? 2 : 1);
    };
    const scoreItem = (item, words, fullQuery) => {
      const title = normalise(item.title);
      const description = normalise(item.description);
      const text = normalise(item.text);
      const headings = (item.headings || []).map(heading => normalise(heading.text || heading));
      const tags = (item.tags || []).map(normalise);
      let score = 0;
      if (title === fullQuery) score += 160;
      else if (title.startsWith(fullQuery)) score += 110;
      else if (title.includes(fullQuery)) score += 75;
      if (headings.some(heading => heading.includes(fullQuery))) score += 45;
      for (const word of words) {
        let wordScore = 0;
        if (title.includes(word)) wordScore = Math.max(wordScore, 34 + (title.startsWith(word) ? 12 : 0));
        if (headings.some(heading => heading.includes(word))) wordScore = Math.max(wordScore, 24);
        if (tags.some(tag => tag.includes(word))) wordScore = Math.max(wordScore, 20);
        if (description.includes(word)) wordScore = Math.max(wordScore, 14);
        if (text.includes(word)) wordScore = Math.max(wordScore, 5);
        const titleWords = title.split(/\s+/);
        const headingWords = headings.flatMap(heading => heading.split(/\s+/));
        if (titleWords.some(candidate => fuzzyMatch(word, candidate))) wordScore = Math.max(wordScore, 80);
        else if (headingWords.some(candidate => fuzzyMatch(word, candidate))) wordScore = Math.max(wordScore, 18);
        if (!wordScore) return -1;
        score += wordScore;
      }
      return score;
    };
    const highlight = (value, words) => {
      let safe = escapeHTML(value);
      const unique = [...new Set(words)].sort((left, right) => right.length - left.length);
      if (!unique.length) return safe;
      const pattern = new RegExp('(' + unique.map(escapePattern).join('|') + ')', 'gi');
      return safe.replace(pattern, '<mark>$1</mark>');
    };
    const snippetFor = (item, words, limit = 190) => {
      const source = String(item.description || item.text || '').replace(/\s+/g, ' ').trim();
      if (!source) return '';
      const lower = normalise(source);
      let start = words.reduce((best, word) => {
        const position = lower.indexOf(word);
        return position >= 0 && (best < 0 || position < best) ? position : best;
      }, -1);
      if (start < 0) start = 0;
      start = Math.max(0, start - 55);
      let excerpt = source.slice(start, start + limit).trim();
      if (start > 0) excerpt = '…' + excerpt;
      if (start + limit < source.length) excerpt += '…';
      return highlight(excerpt, words);
    };
    const ensureOverlay = () => {
      if (overlay) return;
      overlay = document.createElement('div');
      overlay.className = 'mpress-search-overlay';
      overlay.innerHTML = '<section class="mpress-search-dialog" id="mpress-search-dialog" role="dialog" aria-modal="true" aria-label="' + escapeHTML(searchPlaceholder) + '">' +
        '<div class="mpress-search-query">' + searchIcon('search') + '<input type="search" autocomplete="off" spellcheck="false" placeholder="' + escapeHTML(searchPlaceholder) + '" aria-label="' + escapeHTML(searchPlaceholder) + '" role="combobox" aria-expanded="false" aria-controls="mpress-search-listbox" aria-autocomplete="list"><button class="mpress-search-close" type="button" aria-label="Close search">ESC</button></div>' +
        '<div class="mpress-search-meta"><span class="mpress-search-count" aria-live="polite">Type to search</span><span class="mpress-search-hint"><kbd>↑</kbd><kbd>↓</kbd> navigate · <kbd>Enter</kbd> open</span></div>' +
        '<div class="mpress-search-list" id="mpress-search-listbox" role="listbox" aria-label="Search results"></div><aside class="mpress-search-preview" aria-label="Result preview"></aside></section>';
      document.body.append(overlay);
      dialog = overlay.querySelector('.mpress-search-dialog');
      overlayInput = overlay.querySelector('.mpress-search-query input');
      resultList = overlay.querySelector('.mpress-search-list');
      resultCount = overlay.querySelector('.mpress-search-count');
      preview = overlay.querySelector('.mpress-search-preview');
      overlay.addEventListener('click', event => {
        if (event.target === overlay) closeSearch();
      });
      overlay.querySelector('.mpress-search-close').addEventListener('click', closeSearch);
      overlayInput.addEventListener('input', () => {
        clearTimeout(debounce);
        debounce = setTimeout(runSearch, 45);
      });
      overlayInput.addEventListener('keydown', event => {
        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
          event.preventDefault();
          setActive(active + (event.key === 'ArrowDown' ? 1 : -1));
        } else if (event.key === 'Enter' && matches.length) {
          event.preventDefault();
          const target = resultList.querySelector('#mpress-search-option-' + (active < 0 ? 0 : active));
          target?.click();
        }
      });
    };
    const renderPreview = (item, words) => {
      if (!item) {
        preview.replaceChildren();
        return;
      }
      let html = '<h2>' + highlight(item.title, words) + '</h2>';
      if (item.description) html += '<p>' + highlight(item.description, words) + '</p>';
      const headings = (item.headings || []).filter(heading => heading.text).slice(0, 9);
      if (headings.length) {
        html += '<ul class="mpress-search-preview-headings">';
        headings.forEach(heading => {
          html += '<li class="level-' + Number(heading.level || 2) + '"><a href="' + escapeHTML(item.url) + '#' + encodeURIComponent(heading.id || '') + '">' + highlight(heading.text, words) + '</a></li>';
        });
        html += '</ul>';
      }
      const body = snippetFor(item, words, 620);
      if (body) html += '<div class="mpress-search-preview-body">' + body + '</div>';
      preview.innerHTML = html;
    };
    const resultLinks = () => [...resultList.querySelectorAll('a.mpress-search-item')];
    const setActive = value => {
      const links = resultLinks();
      active = links.length ? Math.max(0, Math.min(value, links.length - 1)) : -1;
      links.forEach((link, position) => {
        const selected = position === active;
        link.classList.toggle('active', selected);
        link.setAttribute('aria-selected', String(selected));
      });
      if (active >= 0) {
        overlayInput.setAttribute('aria-activedescendant', links[active].id);
        links[active].scrollIntoView({block: 'nearest'});
        renderPreview(matches[active]?.item, normalise(overlayInput.value).split(/\s+/).filter(Boolean));
      } else {
        overlayInput.removeAttribute('aria-activedescendant');
        renderPreview(null, []);
      }
    };
    const renderRecent = () => {
      ensureOverlay();
      const recent = readRecent();
      matches = [];
      active = -1;
      dialog.classList.remove('has-results');
      overlayInput.setAttribute('aria-expanded', 'false');
      overlayInput.removeAttribute('aria-activedescendant');
      resultCount.textContent = recent.length ? 'Recent searches' : 'Type to search';
      preview.replaceChildren();
      if (!recent.length) {
        resultList.innerHTML = '<div class="mpress-search-empty"><strong>Search this documentation</strong><span>Find pages, headings, concepts, and code terms.</span></div>';
        return;
      }
      resultList.innerHTML = '<div class="mpress-search-recent-heading"><span>Recent</span><button class="mpress-search-clear" type="button">Clear</button></div><div class="mpress-search-recent">' + recent.map((value, position) => '<button class="mpress-search-item" type="button" data-recent="' + position + '">' + searchIcon('clock') + '<span>' + escapeHTML(value) + '</span></button>').join('') + '</div>';
      resultList.querySelector('.mpress-search-clear').addEventListener('click', () => {
        try { localStorage.removeItem(recentKey); } catch (_) {}
        renderRecent();
      });
      resultList.querySelectorAll('[data-recent]').forEach(button => button.addEventListener('click', () => {
        overlayInput.value = recent[Number(button.dataset.recent)] || '';
        runSearch();
      }));
    };
    const renderResults = (queryValue, words) => {
      const resultTotal = matches.length;
      resultCount.textContent = resultTotal ? resultTotal + ' result' + (resultTotal === 1 ? '' : 's') : 'No results';
      dialog.classList.toggle('has-results', resultTotal > 0);
      overlayInput.setAttribute('aria-expanded', String(resultTotal > 0));
      if (!resultTotal) {
        active = -1;
        overlayInput.removeAttribute('aria-activedescendant');
        preview.replaceChildren();
        resultList.innerHTML = '<div class="mpress-search-empty"><strong>No pages found for “' + escapeHTML(queryValue) + '”</strong><span>Try fewer words or check the spelling.</span></div>';
        return;
      }
      resultList.innerHTML = matches.map((match, position) => {
        const item = match.item;
        return '<a class="mpress-search-item" id="mpress-search-option-' + position + '" href="' + escapeHTML(item.url) + '" role="option" aria-selected="false"><div class="mpress-search-item-title"><span>' + highlight(item.title, words) + '</span><span class="mpress-search-item-path">' + escapeHTML(item.url) + '</span></div><div class="mpress-search-item-snippet">' + snippetFor(item, words) + '</div></a>';
      }).join('');
      resultLinks().forEach((link, position) => {
        link.addEventListener('mouseenter', () => setActive(position));
        link.addEventListener('focus', () => setActive(position));
        link.addEventListener('click', () => saveRecent(queryValue));
      });
      setActive(0);
    };
    const runSearch = async () => {
      ensureOverlay();
      const queryValue = overlayInput.value.trim();
      if (!queryValue) {
        renderRecent();
        return;
      }
      await loadIndex();
      if (overlayInput.value.trim() !== queryValue) return;
      const fullQuery = normalise(queryValue);
      const words = fullQuery.split(/\s+/).filter(Boolean);
      matches = index.map(item => ({item, score: scoreItem(item, words, fullQuery)}))
        .filter(match => match.score >= 0)
        .sort((left, right) => right.score - left.score || left.item.title.localeCompare(right.item.title))
        .slice(0, searchMaxResults);
      renderResults(queryValue, words);
    };
    const openSearch = () => {
      ensureOverlay();
      overlay.classList.add('open');
      document.body.classList.add('mpress-search-open');
      query.setAttribute('aria-expanded', 'true');
      renderRecent();
      void loadIndex();
      requestAnimationFrame(() => overlayInput.focus());
    };
    function closeSearch() {
      if (!overlay?.classList.contains('open')) return;
      overlay.classList.remove('open');
      document.body.classList.remove('mpress-search-open');
      query.setAttribute('aria-expanded', 'false');
      query.removeAttribute('aria-activedescendant');
      overlayInput.value = '';
      matches = [];
      active = -1;
      suppressSearchFocus = true;
      query.focus({preventScroll: true});
      requestAnimationFrame(() => suppressSearchFocus = false);
    }
    query.setAttribute('aria-expanded', 'false');
    query.addEventListener('focus', () => {
      if (!suppressSearchFocus) openSearch();
    });
    query.addEventListener('pointerdown', event => {
      event.preventDefault();
      openSearch();
    });
    query.addEventListener('mousedown', openSearch);
    query.addEventListener('click', openSearch);
    document.addEventListener('keydown', event => {
      if (!shortcutEditableTarget(event.target) && shortcutMatches(event, searchShortcut)) {
        event.preventDefault();
        if (overlay?.classList.contains('open')) overlayInput.focus();
        else openSearch();
      } else if (event.key === 'Escape') {
        closeSearch();
      }
    });
  }

})();
`

// themeBootstrapJS applies the user's last theme choice before the blocking
// stylesheet is requested. This keeps a dark or light preference from briefly
// rendering with the site's configured default during a reload.
const themeBootstrapJS = `(function(){try{var t=localStorage.getItem('mpress-theme');if(t==='system'||t==='dark'||t==='light'){document.documentElement.setAttribute('data-theme',t);}}catch(_){}})();`
