(() => {
  const host = document.querySelector('#mpress-devbar-host');
  if (!host || host.shadowRoot) return;
  const root = host.attachShadow({mode: 'open'});
  const themeMedia = matchMedia('(prefers-color-scheme: dark)');
  const syncDevbarTheme = () => {
    const configured = document.documentElement.dataset.theme || 'system';
    host.dataset.theme = configured === 'system' ? (themeMedia.matches ? 'dark' : 'light') : configured;
  };
  syncDevbarTheme();
  new MutationObserver(syncDevbarTheme).observe(document.documentElement, {attributes: true, attributeFilter: ['data-theme']});
  themeMedia.addEventListener?.('change', syncDevbarTheme);
  const pairingURL = new URL(location.href);
  const workspaceMode = host.hasAttribute('data-mpress-workspace');
  const workspaceTool = pairingURL.searchParams.get('tool') || 'page';
  const requestedReturnURL = pairingURL.searchParams.get('return') || '/';
  const returnURL = requestedReturnURL.startsWith('/') && !requestedReturnURL.startsWith('/__mpress') ? requestedReturnURL : '/';
  const returnLocation = new URL(returnURL, location.origin);
  const siteRoute = workspaceMode ? returnLocation.pathname + returnLocation.search : location.pathname + location.search;
  const pairingToken = pairingURL.searchParams.get('token');
  const contributionRequested = pairingURL.searchParams.get('contribute') === '1';
  if (pairingToken) {
    sessionStorage.setItem('mpress-token', pairingToken);
    pairingURL.searchParams.delete('token');
  }
  if (contributionRequested) pairingURL.searchParams.delete('contribute');
  if (pairingToken || contributionRequested) {
    history.replaceState({}, '', pairingURL.pathname + pairingURL.search + pairingURL.hash);
  }
  const resumeWizard = sessionStorage.getItem('mpress-wizard-step') !== null;
  const savedWizardStep = Number.parseInt(sessionStorage.getItem('mpress-wizard-step') || '0', 10);
  const state = {project: null, token: sessionStorage.getItem('mpress-token') || '', menu: workspaceMode, wizardStep: Number.isFinite(savedWizardStep) ? Math.min(4, Math.max(0, savedWizardStep)) : 0, wizardActive: false, pageScrollLock: null, suppressReloadUntil: 0, configPreview: null, drawerDrag: null, previewSize: 'web', previewStage: null, previewBodyOverflow: ''};
  let workspaceHasUnsavedChanges = () => false;
  let canLeaveWorkspace = () => true;
  const resetWorkspaceChangeGuard = () => {
    workspaceHasUnsavedChanges = () => false;
    canLeaveWorkspace = () => true;
  };
  addEventListener('beforeunload', event => {
    if (!workspaceHasUnsavedChanges()) return;
    event.preventDefault();
    event.returnValue = '';
  });
  const api = async (path, options = {}) => {
    const headers = new Headers(options.headers || {});
    if (state.token) headers.set('X-MPress-Token', state.token);
    if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json');
    const response = await fetch('/__mpress/api/' + path, {...options, headers});
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(payload.error || 'M-Press request failed');
      error.status = response.status;
      error.payload = payload;
      throw error;
    }
    return payload;
  };
  const q = selector => root.querySelector(selector);
  const iconHTML = name => q('#icon-' + name)?.innerHTML || '';
  const escapeHTML = value => String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
  const previewWidths = {mobile: '390px', tablet: '768px'};
  const previewURL = () => {
    const url = new URL(location.href);
    url.searchParams.set('__mpress_preview', '1');
    return url.pathname + url.search + url.hash;
  };
  const setPreviewSize = size => {
    const next = Object.hasOwn(previewWidths, size) ? size : 'web';
    state.previewSize = next;
    root.querySelectorAll('[data-preview-size]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.previewSize === next)));
    if (next === 'web') {
      state.previewStage?.remove();
      state.previewStage = null;
      document.body.style.overflow = state.previewBodyOverflow;
      return;
    }
    if (!state.previewStage) {
      state.previewBodyOverflow = document.body.style.overflow;
      const stage = document.createElement('div');
      stage.className = 'mpress-responsive-preview-stage';
      const frame = document.createElement('iframe');
      frame.className = 'mpress-responsive-preview-frame';
      frame.title = 'Responsive site preview';
      frame.src = previewURL();
      stage.append(frame);
      document.body.append(stage);
      document.body.style.overflow = 'hidden';
      state.previewStage = stage;
    }
    state.previewStage.style.setProperty('--mpress-preview-width', previewWidths[next]);
  };
  const contributionFlowKey = () => state.project?.contribution ? `mpress-contribution-flow:${state.project.contribution.branch}` : '';
  const contributionFlow = () => {
    const key = contributionFlowKey();
    return key ? sessionStorage.getItem(key) || state.project?.contribution?.goal || 'choose' : '';
  };
  const rememberContributionFlow = flow => {
    const key = contributionFlowKey();
    if (key) sessionStorage.setItem(key, flow);
    if (workspaceMode) installWorkspaceSidebar();
  };
  const plainGuideText = markdown => String(markdown || '')
    .replace(/^---[\s\S]*?^---\s*/m, '')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/[*_`>#-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  const renderGuideMarkdown = markdown => {
    const lines = String(markdown || '').replace(/\r\n?/g, '\n').split('\n');
    let html = '';
    let list = false;
    let code = false;
    let codeLines = [];
    const closeList = () => { if (list) { html += '</ul>'; list = false; } };
    const flushCode = () => { if (codeLines.length) html += `<pre><code>${escapeHTML(codeLines.join('\n'))}</code></pre>`; codeLines = []; };
    for (const line of lines) {
      if (/^```/.test(line)) {
        closeList();
        if (code) flushCode();
        code = !code;
        continue;
      }
      if (code) { codeLines.push(line); continue; }
      const heading = line.match(/^(#{1,3})\s+(.+)$/);
      if (heading) { closeList(); html += `<h${heading[1].length + 2}>${escapeHTML(heading[2])}</h${heading[1].length + 2}>`; continue; }
      const item = line.match(/^\s*[-*+]\s+(.+)$/);
      if (item) { if (!list) { html += '<ul>'; list = true; } html += `<li>${escapeHTML(item[1])}</li>`; continue; }
      closeList();
      if (line.trim()) html += `<p>${escapeHTML(line.trim())}</p>`;
    }
    closeList();
    flushCode();
    return html;
  };
  const contributorGuideHTML = (guide, qualifier = '') => {
    if (!guide) return '';
    const summary = plainGuideText(guide.markdown).slice(0, 240);
    return `<section class="contributor-guide-summary"><strong>Before you begin</strong><p>${escapeHTML(summary)}${plainGuideText(guide.markdown).length > 240 ? '…' : ''}</p><details class="contributor-guide"><summary>Read complete ${escapeHTML(guide.path)}${qualifier}</summary><div class="contributor-guide-content">${renderGuideMarkdown(guide.markdown)}</div></details></section>`;
  };
  const renderContributionDiff = diff => {
    const rows = String(diff || '').replace(/\r\n?/g, '\n').split('\n').map(line => {
      let kind = 'context';
      if (line.startsWith('@@')) kind = 'hunk';
      else if (/^(diff --git|index |--- |\+\+\+ )/.test(line)) kind = 'meta';
      else if (line.startsWith('+')) kind = 'added';
      else if (line.startsWith('-')) kind = 'removed';
      return `<span class="contribution-diff-line ${kind}">${escapeHTML(line || ' ')}</span>`;
    }).join('');
    return `<details class="contribution-diff" open><summary>Source diff</summary><div class="contribution-diff-code" role="region" aria-label="Source changes"><code>${rows}</code></div></details>`;
  };
  const checkedAttr = value => value ? ' checked' : '';
  const selectedAttr = (value, expected) => value === expected ? ' selected' : '';
  const shortcutModifierAliases = {mod: 'Mod', command: 'Meta', cmd: 'Meta', meta: 'Meta', control: 'Control', ctrl: 'Control', option: 'Alt', alt: 'Alt', shift: 'Shift'};
  const shortcutModifierOrder = ['Mod', 'Control', 'Meta', 'Alt', 'Shift'];
  const shortcutKeyAliases = {esc: 'Escape', escape: 'Escape', enter: 'Enter', space: 'Space', spacebar: 'Space'};
  const normaliseShortcutValue = value => {
    const raw = String(value || '').trim();
    if (!raw) return null;
    if (raw.toLowerCase() === 'none') return 'None';
    const modifiers = new Set();
    let key = '';
    for (const part of raw.split('+').map(token => token.trim()).filter(Boolean)) {
      const modifier = shortcutModifierAliases[part.toLowerCase()];
      if (modifier) {
        if (modifiers.has(modifier)) return null;
        modifiers.add(modifier);
        continue;
      }
      if (key) return null;
      const alias = shortcutKeyAliases[part.toLowerCase()];
      if (alias) key = alias;
      else if (part.length === 1) key = part.toUpperCase();
      else if (/^F(?:[1-9]|1[0-2])$/i.test(part)) key = part.toUpperCase();
      else return null;
    }
    if (!modifiers.size || !key) return null;
    return shortcutModifierOrder.filter(modifier => modifiers.has(modifier)).concat(key).join('+');
  };
  const capturedShortcut = event => {
    if (event.key === 'Backspace' || event.key === 'Delete') return 'None';
    if (['Control', 'Meta', 'Alt', 'Shift'].includes(event.key)) return null;
    const apple = /(Mac|iPhone|iPad|iPod)/i.test(navigator.userAgentData?.platform || navigator.platform || '');
    const modifiers = [];
    if (event.metaKey && event.ctrlKey) modifiers.push('Meta', 'Control');
    else if (event.metaKey) modifiers.push(apple ? 'Mod' : 'Meta');
    else if (event.ctrlKey) modifiers.push(apple ? 'Control' : 'Mod');
    if (event.altKey) modifiers.push('Alt');
    if (event.shiftKey) modifiers.push('Shift');
    if (!modifiers.length) return null;
    const key = event.key === ' ' ? 'Space' : event.key;
    return normaliseShortcutValue(modifiers.concat(key).join('+'));
  };
  const installShortcutCapture = container => {
    const controls = Array.from(container.querySelectorAll('[data-mpress-control="shortcut"]'));
    const labels = new Map([['search.shortcut', 'Search shortcut'], ['accessibility.shortcut', 'Accessibility shortcut']]);
    const setStatus = (control, message = '', error = false) => {
      const status = control.closest('.mpress-form-field')?.querySelector('[data-shortcut-status]');
      if (!status) return;
      status.textContent = message;
      status.hidden = !message;
      status.dataset.state = error ? 'error' : 'hint';
    };
    const finishCapture = (control, keepValue = true) => {
      const wrapper = control.closest('.mpress-shortcut-control');
      const button = wrapper?.querySelector('[data-shortcut-change]');
      if (!keepValue) control.value = control.dataset.shortcutPrevious || '';
      control.readOnly = true;
      control.removeAttribute('data-capturing');
      control.placeholder = control.dataset.shortcutPlaceholder || '';
      delete control.dataset.shortcutPrevious;
      if (button) {
        button.textContent = 'Change key';
        button.setAttribute('aria-pressed', 'false');
      }
    };
    const startCapture = control => {
      controls.forEach(other => {
        if (other.hasAttribute('data-capturing')) finishCapture(other, false);
      });
      const button = control.closest('.mpress-shortcut-control')?.querySelector('[data-shortcut-change]');
      control.dataset.shortcutPrevious = control.value;
      control.dataset.shortcutPlaceholder = control.placeholder;
      control.value = '';
      control.placeholder = 'Press key combination';
      control.readOnly = false;
      control.setAttribute('data-capturing', 'true');
      if (button) {
        button.textContent = 'Cancel';
        button.setAttribute('aria-pressed', 'true');
      }
      setStatus(control, 'Press a modifier and a key. Press Backspace to clear the shortcut.');
      control.focus();
    };
    const sync = () => {
      const seen = new Map();
      controls.forEach(control => {
        if (control.hasAttribute('data-capturing')) return;
        const value = normaliseShortcutValue(control.value);
        control.setCustomValidity('');
        control.removeAttribute('aria-invalid');
        if (control.value.trim() && !value) {
          control.setCustomValidity('Use the shortcut capture instead of typing a shortcut manually.');
          control.setAttribute('aria-invalid', 'true');
          setStatus(control, 'Use a modifier plus a key, or press Backspace to clear.', true);
          return;
        }
        if (!value || value === 'None') {
          setStatus(control);
          return;
        }
        control.value = value;
        const other = seen.get(value.toLowerCase());
        if (other) {
          const message = `Conflicts with ${labels.get(other.name) || 'another M-Press shortcut'}. Choose another shortcut.`;
          control.setCustomValidity(message);
          control.setAttribute('aria-invalid', 'true');
          setStatus(control, message, true);
          return;
        }
        seen.set(value.toLowerCase(), control);
        setStatus(control);
      });
    };
    controls.forEach(control => {
      if (control.dataset.shortcutCaptureInstalled === 'true') return;
      control.dataset.shortcutCaptureInstalled = 'true';
      const change = control.closest('.mpress-shortcut-control')?.querySelector('[data-shortcut-change]');
      change?.addEventListener('click', event => {
        event.preventDefault();
        event.stopPropagation();
        if (control.hasAttribute('data-capturing')) {
          finishCapture(control, false);
          sync();
          return;
        }
        startCapture(control);
      });
      control.addEventListener('keydown', event => {
        if (!control.hasAttribute('data-capturing')) return;
        if (event.isComposing) return;
        if (event.key === 'Escape') {
          event.preventDefault();
          event.stopPropagation();
          finishCapture(control, false);
          sync();
          change?.focus();
          return;
        }
        const captured = capturedShortcut(event);
        event.preventDefault();
        event.stopPropagation();
        if (!captured) {
          setStatus(control, 'Press a modifier plus a key, or press Backspace to clear.', true);
          return;
        }
        control.value = captured;
        finishCapture(control);
        control.dispatchEvent(new Event('input', {bubbles: true}));
        sync();
        change?.focus();
      });
      control.addEventListener('input', sync);
      control.addEventListener('blur', () => {
        if (!control.hasAttribute('data-capturing')) {
          sync();
          return;
        }
        setTimeout(() => {
          if (control.closest('.mpress-shortcut-control')?.contains(document.activeElement)) return;
          finishCapture(control, false);
          sync();
        });
      });
    });
    sync();
  };
  let styledSelectID = 0;
  const closeStyledSelects = except => {
    root.querySelectorAll('.mpress-select[data-open="true"]').forEach(wrapper => {
      if (wrapper === except) return;
      wrapper.dataset.open = 'false';
      wrapper.querySelector('.mpress-select-trigger')?.setAttribute('aria-expanded', 'false');
      const menu = wrapper.querySelector('.mpress-select-menu');
      if (menu) menu.hidden = true;
    });
  };
  const installStyledSelects = container => {
    if (!container) return;
    container.querySelectorAll('select:not([data-mpress-select-ready])').forEach(select => {
      const label = select.closest('label');
      const wrapper = document.createElement('span');
      const trigger = document.createElement('button');
      const value = document.createElement('span');
      const menu = document.createElement('span');
      const id = `mpress-select-${++styledSelectID}`;
      const menuID = `${id}-listbox`;
      wrapper.className = 'mpress-select';
      wrapper.dataset.open = 'false';
      trigger.type = 'button';
      trigger.id = id;
      trigger.className = 'mpress-select-trigger';
      trigger.setAttribute('aria-haspopup', 'listbox');
      trigger.setAttribute('aria-expanded', 'false');
      trigger.setAttribute('aria-controls', menuID);
      value.className = 'mpress-select-value';
      menu.id = menuID;
      menu.className = 'mpress-select-menu';
      menu.setAttribute('role', 'listbox');
      menu.hidden = true;
      select.parentNode.insertBefore(wrapper, select);
      wrapper.append(select, trigger, menu);
      select.dataset.mpressSelectReady = 'true';
      select.classList.add('mpress-select-native');
      select.tabIndex = -1;
      select.setAttribute('aria-hidden', 'true');
      if (label) label.htmlFor = id;
      const sync = () => {
        const selected = select.options[select.selectedIndex];
        value.textContent = selected?.textContent || '';
        trigger.disabled = select.disabled;
        trigger.setAttribute('aria-label', `${label?.querySelector(':scope > span')?.textContent?.trim() || select.name}: ${value.textContent}`);
        menu.querySelectorAll('.mpress-select-option').forEach(option => {
          const active = option.dataset.value === select.value;
          option.dataset.selected = String(active);
          option.setAttribute('aria-selected', String(active));
        });
      };
      Array.from(select.options).forEach(option => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'mpress-select-option';
        item.dataset.value = option.value;
        item.setAttribute('role', 'option');
        item.textContent = option.textContent;
        const choose = () => {
          select.value = option.value;
          select.dispatchEvent(new Event('input', {bubbles: true}));
          select.dispatchEvent(new Event('change', {bubbles: true}));
          sync();
          closeStyledSelects();
          trigger.focus();
        };
        item.addEventListener('click', event => {
          event.preventDefault();
          choose();
        });
        item.addEventListener('keydown', event => {
          const items = Array.from(menu.querySelectorAll('.mpress-select-option'));
          const index = items.indexOf(item);
          const destination = event.key === 'ArrowDown' ? Math.min(items.length - 1, index + 1)
            : event.key === 'ArrowUp' ? Math.max(0, index - 1)
            : event.key === 'Home' ? 0
            : event.key === 'End' ? items.length - 1
            : -1;
          if (destination >= 0) {
            event.preventDefault();
            items[destination].focus();
          } else if (event.key === 'Escape') {
            event.preventDefault();
            closeStyledSelects();
            trigger.focus();
          } else if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            choose();
          }
        });
        menu.append(item);
      });
      trigger.append(value);
      const open = () => {
        if (trigger.disabled) return;
        closeStyledSelects(wrapper);
        wrapper.dataset.open = 'true';
        trigger.setAttribute('aria-expanded', 'true');
        menu.hidden = false;
        const selected = menu.querySelector('.mpress-select-option[data-selected="true"]');
        (selected || menu.querySelector('.mpress-select-option'))?.focus();
      };
      trigger.addEventListener('click', event => {
        event.preventDefault();
        if (wrapper.dataset.open === 'true') {
          closeStyledSelects();
          trigger.focus();
        } else open();
      });
      trigger.addEventListener('keydown', event => {
        if (!['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) return;
        event.preventDefault();
        open();
      });
      select.addEventListener('input', sync);
      select.addEventListener('change', sync);
      select.mpressSelectSync = sync;
      sync();
    });
  };
  const installColorControls = container => {
    if (!container) return;
    const defaults = {
      accentColor: '#5375f6', 'theme.accentColor': '#5375f6',
      hoverColorLight: '#7593ff', 'theme.hoverColorLight': '#7593ff',
      hoverColorDark: '#7593ff', 'theme.hoverColorDark': '#7593ff'
    };
    container.querySelectorAll('input[type="color"]:not([data-mpress-color-ready])').forEach(input => {
      const wrapper = document.createElement('span');
      const text = document.createElement('input');
      const reset = document.createElement('button');
      wrapper.className = 'mpress-color-control';
      text.type = 'text';
      text.className = 'mpress-color-value';
      text.inputMode = 'text';
      text.maxLength = 7;
      text.setAttribute('aria-label', `${input.name || 'Colour'} hexadecimal value`);
      reset.type = 'button';
      reset.className = 'mpress-color-reset';
      reset.textContent = 'Reset';
      input.parentNode.insertBefore(wrapper, input);
      wrapper.append(input, text, reset);
      input.dataset.mpressColorReady = 'true';
      const sync = () => { text.value = input.value.toUpperCase(); };
      input.addEventListener('input', sync);
      text.addEventListener('input', () => {
        const value = text.value.trim();
        const valid = /^#[0-9a-f]{6}$/i.test(value);
        text.setCustomValidity(valid ? '' : 'Enter a six-digit hexadecimal colour, for example #5375F6.');
        if (!valid) return;
        input.value = value;
        input.dispatchEvent(new Event('input', {bubbles: true}));
        input.dispatchEvent(new Event('change', {bubbles: true}));
      });
      reset.addEventListener('click', () => {
        input.value = defaults[input.name] || input.defaultValue || '#5375f6';
        sync();
        input.dispatchEvent(new Event('input', {bubbles: true}));
        input.dispatchEvent(new Event('change', {bubbles: true}));
      });
      sync();
    });
  };
  root.addEventListener('click', event => {
    if (!event.target.closest('.mpress-select')) closeStyledSelects();
  });
  const commonLanguageSuggestions = [
    {code: 'fr', label: 'Français'}, {code: 'es', label: 'Español'}, {code: 'de', label: 'Deutsch'},
    {code: 'pt-BR', label: 'Português (Brasil)'}, {code: 'it', label: 'Italiano'}, {code: 'ja', label: '日本語'},
    {code: 'ko', label: '한국어'}, {code: 'zh-CN', label: '简体中文'}, {code: 'zh-TW', label: '繁體中文'},
    {code: 'ru', label: 'Русский'}, {code: 'id', label: 'Bahasa Indonesia'}
  ];
  const suggestLanguage = existing => {
    const configured = new Set((existing || []).map(value => String(value).toLowerCase()));
    return commonLanguageSuggestions.find(item => !configured.has(item.code.toLowerCase())) || {code: '', label: ''};
  };
  const languageRow = (code = '', label = '') => `<div class="repeat-row language-row"><label class="field"><span class="language-cell-label">Code</span><input data-language-code required placeholder="en" value="${escapeHTML(code)}"></label><label class="field"><span class="language-cell-label">Display label</span><input data-language-label placeholder="English" value="${escapeHTML(label)}"></label><button class="button remove-row language-remove" type="button" data-remove-row aria-label="Remove ${escapeHTML(label || code || 'language')}" title="Remove language">${iconHTML('x')}</button></div>`;
  const headerLinkRow = (label = '', url = '', type = 'link', variant = 'primary', color = '#5375f6') => `<div class="repeat-row header-link-row"><label class="field"><span>Label</span><input data-header-label required placeholder="Documentation" value="${escapeHTML(label)}"></label><label class="field"><span>URL</span><input data-header-url required placeholder="/docs/" value="${escapeHTML(url)}"></label><label class="field"><span>Style</span><select data-header-type><option value="link"${selectedAttr(type || 'link', 'link')}>Link</option><option value="button"${selectedAttr(type, 'button')}>Button</option></select></label><label class="field"><span>Button variant</span><select data-header-variant><option value="primary"${selectedAttr(variant || 'primary', 'primary')}>Primary</option><option value="secondary"${selectedAttr(variant, 'secondary')}>Secondary</option><option value="outline"${selectedAttr(variant, 'outline')}>Outline</option><option value="custom"${selectedAttr(variant, 'custom')}>Custom colour</option></select></label><label class="field"><span>Custom colour</span><input data-header-color type="color" value="${escapeHTML(color || '#5375f6')}"></label><button class="button remove-row" type="button" data-remove-row>Remove</button></div>`;
  const deployTargetRow = (name = '', target = {}) => `<div class="deploy-target repeat-row"><div class="repeat-row-heading"><strong>Deployment target</strong><button class="button remove-row" type="button" data-remove-row>Remove</button></div><div class="settings-grid"><label class="field"><span>Target name</span><input data-deploy-name required placeholder="production" value="${escapeHTML(name)}"></label><label class="field"><span>Provider</span><select data-deploy-provider><option value="cloudflare-pages"${selectedAttr(target.provider || 'cloudflare-pages', 'cloudflare-pages')}>Cloudflare Pages</option><option value="netlify"${selectedAttr(target.provider, 'netlify')}>Netlify</option></select></label><label class="field"><span>Account or team</span><input data-deploy-account value="${escapeHTML(target.accountID)}"></label><label class="field"><span>Project or site</span><input data-deploy-project required placeholder="my-documentation" value="${escapeHTML(target.project)}"></label><label class="field"><span>Production branch</span><input data-deploy-branch placeholder="main" value="${escapeHTML(target.productionBranch)}"></label><label class="field"><span>Custom domain</span><input data-deploy-domain placeholder="docs.example.com" value="${escapeHTML(target.domain)}"></label></div></div>`;
  const visibleDrawerControls = () => Array.from(root.querySelectorAll('#drawer button:not(:disabled), #drawer input:not(:disabled), #drawer textarea:not(:disabled), #drawer select:not(:disabled), #drawer a[href]')).filter(item => item.offsetParent !== null);
  const toast = (message, error = false) => {
    const item = q('#devbar-toast');
    item.textContent = message;
    item.classList.toggle('error', error);
    item.hidden = false;
    clearTimeout(toast.timer);
    toast.timer = setTimeout(() => item.hidden = true, 3600);
  };
  const formatBytes = value => {
    const bytes = Number(value) || 0;
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };
  const setMenu = open => {
    state.menu = workspaceMode || open;
    q('#command-menu').hidden = !state.menu;
    q('#launcher').setAttribute('aria-expanded', String(state.menu));
    if (open && !workspaceMode) q('#command-menu button')?.focus();
  };
  const openWorkspace = tool => {
    const target = new URL('/__mpress/', location.origin);
    target.searchParams.set('tool', tool);
    target.searchParams.set('return', location.pathname + location.search + location.hash);
    location.assign(target.pathname + target.search);
  };
  const syncWorkspaceTop = () => {
    if (!workspaceMode) return;
    const header = document.querySelector('body > header');
    const top = Math.max(0, Math.round(header?.getBoundingClientRect().bottom || 0));
    const sidebar = document.querySelector('.mpress-workspace-sidebar');
    const sidebarWidth = Math.max(0, Math.round(sidebar?.getBoundingClientRect().right || 0));
    host.style.setProperty('--mp-workspace-top', `${top}px`);
    host.style.setProperty('--mp-workspace-sidebar-width', `${sidebarWidth}px`);
  };
  const workspaceToolURL = tool => {
    const target = new URL('/__mpress/', location.origin);
    target.searchParams.set('tool', tool);
    target.searchParams.set('return', returnURL);
    return target.pathname + target.search;
  };
  const workspaceConfigPages = [
    {tool: 'config-site', page: 'site', label: 'Site', icon: 'globe'},
    {tool: 'config-brand', page: 'brand', label: 'Brand', icon: 'image'},
    {tool: 'config-theme', page: 'theme', label: 'Appearance', icon: 'sun'},
    {tool: 'config-languages', page: 'languages', label: 'Languages', icon: 'languages'},
    {tool: 'config-social', page: 'social', label: 'Navigation and links', icon: 'link'},
    {tool: 'config-versioning', page: 'versioning', label: 'Versions', icon: 'versions'},
    {tool: 'config-accessibility', page: 'accessibility', label: 'Accessibility', icon: 'accessibility'},
    {tool: 'config-build', page: 'build', label: 'Project files', icon: 'file'},
    {tool: 'config-layout', page: 'layout', label: 'Layout', icon: 'layout'},
    {tool: 'config-translation', page: 'translation', label: 'Translation provider', icon: 'languages'},
    {tool: 'config-deploy', page: 'deploy', label: 'Deployment targets', icon: 'rocket'}
  ];
  const configPage = page => workspaceConfigPages.find(item => item.page === page);
  const workspaceNavigationGroups = [
    {label: 'Essentials', items: [
      {tool: 'page', label: 'Edit this page', icon: 'pencil'},
      {tool: 'quick', label: 'Quick setup', icon: 'star'},
      configPage('site'), configPage('brand'), configPage('theme')
    ]},
    {label: 'Content', items: [
      configPage('social'), configPage('languages'), {...configPage('versioning'), feature: 'versions'},
      {tool: 'translations', label: 'Translations', icon: 'languages', feature: 'translations'}
    ]},
    {label: 'Blog', feature: 'blog', items: [
      {tool: 'blog-new', label: 'New post', icon: 'pencil'},
      {tool: 'blog-posts', label: 'Posts', icon: 'book-open'},
      {tool: 'blog-preferences', label: 'Preferences', icon: 'settings'}
    ]},
    {label: 'Quality and delivery', items: [
      configPage('accessibility'),
      {tool: 'checks', label: 'Run checks', icon: 'check-circle'},
      {tool: 'lighthouse', label: 'Lighthouse audit', icon: 'gauge'},
      {tool: 'knowledge', label: 'Knowledge base', icon: 'search'},
      {tool: 'publish', label: 'Publish', icon: 'rocket'}
    ]},
    {label: 'Advanced', items: [
      configPage('build'), configPage('layout'), {...configPage('translation'), feature: 'translations'}, {...configPage('deploy'), feature: 'deployment'}
    ]},
    {label: 'Help', items: [{tool: 'onboarding', label: 'Project guide', icon: 'guide'}]}
  ];
  const setWorkspaceActive = tool => {
    if (!workspaceMode) return;
    document.querySelectorAll('[data-workspace-command]').forEach(item => {
      const active = item.dataset.workspaceCommand === tool;
      item.classList.toggle('active', active);
      if (active) item.setAttribute('aria-current', 'page'); else item.removeAttribute('aria-current');
    });
  };
  const installWorkspaceSidebar = () => {
    if (!workspaceMode) return;
    document.body.className = 'docs-page mpress-workspace-document';
    let layout = document.querySelector('.layout');
    let sidebar = layout?.querySelector('.sidebar');
    if (!layout || !sidebar) {
      layout = document.createElement('div');
      layout.className = 'layout mpress-workspace-layout';
      sidebar = document.createElement('aside');
      sidebar.className = 'sidebar';
      sidebar.id = 'mpress-sidebar';
      sidebar.tabIndex = -1;
      layout.append(sidebar);
      const insertionPoint = Array.from(document.body.children).find(element => element !== host && element.tagName !== 'HEADER' && element.tagName !== 'A' && !element.matches('.utility-menu-panel'));
      document.body.insertBefore(layout, insertionPoint || host);
    }
    layout.hidden = false;
    layout.inert = false;
    layout.removeAttribute('aria-hidden');
    layout.style.removeProperty('display');
    Array.from(layout.children).forEach(child => {
      if (child === sidebar) return;
      child.hidden = true;
      child.inert = true;
      child.setAttribute('aria-hidden', 'true');
    });
    sidebar.classList.add('mpress-workspace-sidebar');
    sidebar.setAttribute('aria-label', 'M-Press project tools');
    const groupKey = group => `mpress-workspace-nav:${group.label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;
    const groupIsOpen = group => {
      const saved = sessionStorage.getItem(groupKey(group));
      if (saved !== null) return saved === 'open';
      return group.label === 'Essentials' || group.items.some(item => item.tool === workspaceTool) || (group.label === 'Blog' && workspaceTool === 'config-blog');
    };
    const featureEnabled = feature => !feature || state.project?.features?.[feature] === true;
    let navigationGroups = workspaceNavigationGroups;
    if (state.project?.contribution) {
      const flow = contributionFlow();
      const contributionItems = [
        {tool: 'contribute', label: 'Contribution guide', icon: 'guide'},
        {tool: 'page', label: 'Edit this page', icon: 'pencil'},
        {tool: 'translations', label: 'Translations', icon: 'languages'},
        {tool: 'checks', label: 'Run checks', icon: 'check-circle'},
        {tool: 'review', label: 'Review changes', icon: 'file'}
      ];
      navigationGroups = [{label: 'Contribution', items: contributionItems}];
      if (flow === 'config') {
        navigationGroups.push({label: 'Site setup', items: [configPage('site'), configPage('brand'), configPage('theme'), configPage('social'), configPage('accessibility')]});
      }
    } else if (state.project?.onboarding && (workspaceTool === 'onboarding' || state.wizardActive)) {
      navigationGroups = [{label: 'Getting started', items: [
        {tool: 'onboarding', label: 'Project guide', icon: 'guide'},
        {tool: 'quick', label: 'Quick setup', icon: 'star'}
      ]}];
    }
    const visibleGroups = navigationGroups.map(group => ({...group, items: group.items.filter(item => featureEnabled(item.feature))})).filter(group => featureEnabled(group.feature) && group.items.length);
    const navigation = visibleGroups.map((group, index) => {
      const panelID = `mpress-workspace-group-${index}`;
      const open = groupIsOpen(group);
      return `<section class="mpress-workspace-group"><button class="mpress-workspace-group-toggle" type="button" aria-expanded="${open}" aria-controls="${panelID}" data-workspace-group-toggle="${escapeHTML(groupKey(group))}"><span>${escapeHTML(group.label)}</span>${iconHTML('chevron-down')}</button><div class="mpress-workspace-group-links" id="${panelID}"${open ? '' : ' hidden'}>${group.items.map(item => `<a class="mpress-workspace-link" href="${escapeHTML(workspaceToolURL(item.tool))}" data-workspace-command="${item.tool}">${iconHTML(item.icon)}<span>${escapeHTML(item.label)}</span></a>`).join('')}</div></section>`;
    }).join('');
    sidebar.innerHTML = `<nav>
      <a class="mpress-workspace-back mpress-workspace-link" href="${escapeHTML(returnURL)}">${iconHTML('arrow-left')}<span>Back to site</span></a>
      ${navigation}
    </nav>`;
    sidebar.querySelector('.mpress-workspace-back')?.addEventListener('click', event => {
      if (canLeaveWorkspace()) return;
      event.preventDefault();
    });
    sidebar.querySelectorAll('[data-workspace-group-toggle]').forEach(toggle => toggle.addEventListener('click', () => {
      const panel = document.getElementById(toggle.getAttribute('aria-controls'));
      const open = toggle.getAttribute('aria-expanded') !== 'true';
      toggle.setAttribute('aria-expanded', String(open));
      panel.hidden = !open;
      sessionStorage.setItem(toggle.dataset.workspaceGroupToggle, open ? 'open' : 'closed');
    }));
    sidebar.querySelectorAll('[data-workspace-command]').forEach(link => link.addEventListener('click', event => {
      event.preventDefault();
      if (!canLeaveWorkspace()) return;
      resetWorkspaceChangeGuard();
      const tool = link.dataset.workspaceCommand;
      const target = new URL(location.href);
      target.searchParams.set('tool', tool);
      history.replaceState({}, '', target.pathname + target.search + target.hash);
      command(tool);
    }));
    syncWorkspaceTop();
  };
  const prepareWorkspaceDocument = () => {
    if (!workspaceMode) return;
    for (const element of Array.from(document.body.children)) {
      if (element === host || element.tagName === 'HEADER' || element.tagName === 'SCRIPT' || element.tagName === 'STYLE' || element.matches('.skip, .utility-menu-panel, .mpress-contribute-dialog')) continue;
      element.hidden = true;
      element.inert = true;
      element.setAttribute('aria-hidden', 'true');
      element.style.setProperty('display', 'none', 'important');
    }
    installWorkspaceSidebar();
  };
  const lockPageScroll = () => {
    if (state.pageScrollLock) return;
    state.pageScrollLock = {html: document.documentElement.style.overflow, body: document.body.style.overflow};
    document.documentElement.style.overflow = 'hidden';
    document.body.style.overflow = 'hidden';
  };
  const unlockPageScroll = () => {
    if (!state.pageScrollLock) return;
    document.documentElement.style.overflow = state.pageScrollLock.html;
    document.body.style.overflow = state.pageScrollLock.body;
    state.pageScrollLock = null;
  };
  const restoreConfigPreview = () => {
    const preview = state.configPreview;
    if (!preview) return;
    if (!preview.committed) {
      document.documentElement.dataset.theme = preview.theme;
      if (preview.accent) document.documentElement.style.setProperty('--accent', preview.accent); else document.documentElement.style.removeProperty('--accent');
      if (preview.hover) document.documentElement.style.setProperty('--hover', preview.hover); else document.documentElement.style.removeProperty('--hover');
      if (preview.hoverLight) document.documentElement.style.setProperty('--hover-light', preview.hoverLight); else document.documentElement.style.removeProperty('--hover-light');
      if (preview.hoverDark) document.documentElement.style.setProperty('--hover-dark', preview.hoverDark); else document.documentElement.style.removeProperty('--hover-dark');
      if (preview.wailsAccentStrong) document.documentElement.style.setProperty('--wails-accent-strong', preview.wailsAccentStrong); else document.documentElement.style.removeProperty('--wails-accent-strong');
      if (preview.wailsAccentLink) document.documentElement.style.setProperty('--wails-accent-link', preview.wailsAccentLink); else document.documentElement.style.removeProperty('--wails-accent-link');
      if (preview.workspaceAccent) host.style.setProperty('--mp-accent', preview.workspaceAccent); else host.style.removeProperty('--mp-accent');
      document.title = preview.documentTitle;
      if (preview.descriptionMeta) preview.descriptionMeta.setAttribute('content', preview.description);
      preview.logoImages.forEach((image, index) => image.alt = preview.logoAlts[index]);
      if (preview.brandTitle) preview.brandTitle.textContent = preview.brandText;
      if (preview.searchContainer) preview.searchContainer.hidden = preview.searchHidden;
      if (preview.searchButton) {
        preview.searchButton.textContent = preview.searchText;
        preview.searchButton.setAttribute('aria-label', preview.searchLabel);
      }
    }
    state.configPreview = null;
  };
  const beginConfigPreview = siteTitle => {
    restoreConfigPreview();
    const descriptionMeta = document.querySelector('meta[name="description"]');
    const logoImages = Array.from(document.querySelectorAll('.brand img'));
    const brandTitle = document.querySelector('.brand > span:not(.brand-mark):not(.theme-logo)');
    const searchContainer = document.querySelector('.search');
    const searchButton = searchContainer?.querySelector('#search');
    state.configPreview = {
      committed: false,
      siteTitle,
      theme: document.documentElement.dataset.theme || 'system',
      accent: document.documentElement.style.getPropertyValue('--accent'),
      hover: document.documentElement.style.getPropertyValue('--hover'),
      hoverLight: document.documentElement.style.getPropertyValue('--hover-light'),
      hoverDark: document.documentElement.style.getPropertyValue('--hover-dark'),
      wailsAccentStrong: document.documentElement.style.getPropertyValue('--wails-accent-strong'),
      wailsAccentLink: document.documentElement.style.getPropertyValue('--wails-accent-link'),
      documentTitle: document.title,
      descriptionMeta,
      description: descriptionMeta?.getAttribute('content') || '',
      logoImages,
      logoAlts: logoImages.map(image => image.alt),
      brandTitle,
      brandText: brandTitle?.textContent || '',
      searchContainer,
      searchButton,
      searchHidden: searchContainer?.hidden || false,
      searchText: searchButton?.textContent || '',
      searchLabel: searchButton?.getAttribute('aria-label') || 'Search',
      workspaceAccent: host.style.getPropertyValue('--mp-accent')
    };
  };
  const applyConfigPreview = values => {
    const preview = state.configPreview;
    if (!preview) return;
    const scheme = String(values.colorScheme || '').trim();
    if (scheme === 'system' || scheme === 'light' || scheme === 'dark') document.documentElement.dataset.theme = scheme;
    const accent = String(values.accentColor || '').trim();
    const hoverLight = String(values.hoverColorLight || values.hoverColor || '').trim();
    const hoverDark = String(values.hoverColorDark || values.hoverColor || '').trim();
    if (accent) {
      document.documentElement.style.setProperty('--accent', accent);
      document.documentElement.style.setProperty('--wails-accent-strong', accent);
      document.documentElement.style.setProperty('--wails-accent-link', accent);
    }
    if (accent && workspaceMode) host.style.setProperty('--mp-accent', accent);
    if (hoverLight) document.documentElement.style.setProperty('--hover-light', hoverLight);
    if (hoverDark) document.documentElement.style.setProperty('--hover-dark', hoverDark);
    const title = String(values.title || '').trim();
    if (title) {
      document.title = preview.siteTitle && preview.documentTitle.includes(preview.siteTitle) ? preview.documentTitle.replace(preview.siteTitle, title) : title;
      preview.logoImages.forEach(image => image.alt = title);
      if (preview.brandTitle) preview.brandTitle.textContent = title;
    }
    if (preview.descriptionMeta && values.description !== undefined) preview.descriptionMeta.setAttribute('content', String(values.description));
    if (preview.searchContainer && values.searchEnabled !== undefined) preview.searchContainer.hidden = !values.searchEnabled;
    const searchPlaceholder = String(values.searchPlaceholder || '').trim();
    if (preview.searchButton && searchPlaceholder) {
      preview.searchButton.textContent = searchPlaceholder;
    }
  };
  const showCurrentPage = async () => {
    openDrawer('Project', 'Edit this page', '<div class="result-box"><strong>Finding the source file</strong>Please wait.</div>', 'tool', 'Continue from the page you were viewing without losing its source context.');
    try {
      const result = await api('route?url=' + encodeURIComponent(siteRoute));
      const source = result.path;
      const checkout = state.project?.repository?.path || '';
      const absolute = checkout ? checkout.replace(/\/$/, '') + '/' + source : source;
      const build = state.project?.state || {};
      setDrawerBody('Edit this page', `
        <div class="workspace-context">
          <span class="wizard-kicker">Current page</span>
          <div class="result-box"><strong>Source file</strong><code>${escapeHTML(source)}</code></div>
          <div class="result-box"><strong>Local checkout</strong><code>${escapeHTML(checkout)}</code></div>
          <div class="check-item ${build.status === 'ready' ? 'pass' : 'warn'}">${iconHTML(build.status === 'ready' ? 'check' : 'triangle-alert')}<div><strong>${build.status === 'ready' ? 'Live preview is ready' : 'Build needs attention'}</strong><span>${Number(build.pages) || 0} pages · ${Number(build.durationMs) || 0}ms</span></div></div>
          <p class="intro">Open the source file in your editor and save it. M-Press rebuilds this page and reloads the browser automatically.</p>
          <div class="actions"><button class="button" type="button" data-page-copy>Copy file path</button><button class="button" type="button" data-page-settings>Site settings</button><button class="button primary" type="button" data-page-checks>Run checks</button></div>
        </div>`, 'M-Press preserved the exact page you were viewing.');
      q('[data-page-copy]').addEventListener('click', async () => {
        try { await navigator.clipboard.writeText(absolute); toast('File path copied'); }
        catch { toast('Could not copy the file path', true); }
      });
      q('[data-page-settings]').addEventListener('click', () => command('quick'));
      q('[data-page-checks]').addEventListener('click', () => command('checks'));
    } catch (error) {
      setDrawerBody('Edit this page', `<div class="result-box"><strong>Source file not found</strong>${escapeHTML(error.message)}</div><div class="actions"><button class="button" type="button" data-page-settings>Open site settings</button><button class="button primary" type="button" data-page-back>Back to site</button></div>`, 'This route does not map to a documentation source file.');
      q('[data-page-settings]').addEventListener('click', () => command('quick'));
      q('[data-page-back]').addEventListener('click', exitWorkspace);
    }
  };
  const exitWorkspace = () => {
    restoreConfigPreview();
    location.assign(returnURL);
  };
  const closeDrawer = () => {
    if (workspaceMode) {
      restoreConfigPreview();
      command('quick');
      return;
    }
    if (q('#drawer').dataset.mode === 'configuration') restoreConfigPreview();
    q('#drawer').hidden = true;
    q('#scrim').hidden = true;
    unlockPageScroll();
    q('#launcher').focus();
    if (state.wizardActive) sessionStorage.removeItem('mpress-wizard-step');
    state.wizardActive = false;
  };
  const setDrawerBody = (title, html, description = '') => {
    q('#drawer-title').textContent = title;
    const body = q('#drawer-body');
    body.innerHTML = workspaceMode ? `<article class="workspace-page"><header class="workspace-page-header"><h1>${escapeHTML(title)}</h1>${description ? `<p>${escapeHTML(description)}</p>` : ''}</header>${html}</article>` : html;
    body.scrollTop = 0;
  };
  const openDrawer = (eyebrow, title, html, mode = 'tool', description = '') => {
    setMenu(workspaceMode);
    const floating = mode === 'configuration' && !workspaceMode;
    if (floating) unlockPageScroll(); else lockPageScroll();
    q('#drawer-eyebrow').textContent = eyebrow;
    setDrawerBody(title, html, description);
    q('#drawer').dataset.mode = mode;
    q('#drawer').setAttribute('aria-modal', String(!floating));
    q('#drawer').style.removeProperty('left');
    q('#drawer').style.removeProperty('top');
    q('#drawer').style.removeProperty('right');
    q('#drawer').style.removeProperty('bottom');
    root.querySelectorAll('[data-command]').forEach(item => item.classList.toggle('active', item.dataset.command === ({Configuration: 'config', 'Project health': 'checks', 'Site quality': 'lighthouse', Export: 'export', Translations: 'translations', Deployment: 'deploy', 'Project guide': 'onboarding'}[eyebrow] || '')));
    setWorkspaceActive(({Configuration: 'quick', 'Project health': 'checks', 'Site quality': 'lighthouse', Export: 'publish', Translations: 'translations', Deployment: 'publish', Publish: 'publish', 'Project guide': 'onboarding'}[eyebrow] || ''));
    if (floating && innerWidth > 720) {
      try {
        const position = JSON.parse(sessionStorage.getItem('mpress-config-position') || 'null');
        if (position && Number.isFinite(position.left) && Number.isFinite(position.top)) {
          const estimatedWidth = Math.min(760, innerWidth - 40);
          const estimatedHeight = Math.min(760, innerHeight - 82);
          q('#drawer').style.left = `${Math.max(8, Math.min(innerWidth - estimatedWidth - 8, position.left))}px`;
          q('#drawer').style.top = `${Math.max(8, Math.min(innerHeight - 42 - estimatedHeight - 8, position.top))}px`;
          q('#drawer').style.right = 'auto';
          q('#drawer').style.bottom = 'auto';
        }
      } catch (_) {}
    }
    q('#drawer').hidden = false;
    q('#scrim').hidden = floating;
    q('#drawer-body input, #drawer-body button, #drawer-body textarea')?.focus();
  };
  const installDrawerDrag = () => {
    if (workspaceMode) return;
    const drawer = q('#drawer');
    const handle = drawer.querySelector(':scope > header');
    const move = event => {
      const drag = state.drawerDrag;
      if (!drag || event.pointerId !== drag.pointerId) return;
      const maxLeft = Math.max(8, innerWidth - drag.width - 8);
      const maxTop = Math.max(8, innerHeight - 42 - drag.height - 8);
      drawer.style.left = `${Math.min(maxLeft, Math.max(8, drag.left + event.clientX - drag.x))}px`;
      drawer.style.top = `${Math.min(maxTop, Math.max(8, drag.top + event.clientY - drag.y))}px`;
      drawer.style.right = 'auto';
      drawer.style.bottom = 'auto';
    };
    const finish = event => {
      if (!state.drawerDrag || event.pointerId !== state.drawerDrag.pointerId) return;
      state.drawerDrag = null;
      drawer.classList.remove('dragging');
      handle.releasePointerCapture?.(event.pointerId);
      const rect = drawer.getBoundingClientRect();
      sessionStorage.setItem('mpress-config-position', JSON.stringify({left: Math.round(rect.left), top: Math.round(rect.top)}));
    };
    handle.addEventListener('pointerdown', event => {
      if (drawer.dataset.mode !== 'configuration' || innerWidth <= 720 || event.button !== 0 || event.target.closest('button, a, input, select, textarea')) return;
      const rect = drawer.getBoundingClientRect();
      state.drawerDrag = {pointerId: event.pointerId, x: event.clientX, y: event.clientY, left: rect.left, top: rect.top, width: rect.width, height: rect.height};
      drawer.classList.add('dragging');
      handle.setPointerCapture?.(event.pointerId);
      event.preventDefault();
    });
    handle.addEventListener('pointermove', move);
    handle.addEventListener('pointerup', finish);
    handle.addEventListener('pointercancel', finish);
  };
  const updateBuild = build => {
    if (!build) return;
    const status = q('.status');
    status.classList.toggle('ready', build.status === 'ready');
    status.classList.toggle('error', build.status === 'error');
    q('#status-label').textContent = build.status === 'ready' ? 'Ready' : build.status === 'building' ? 'Building' : 'Needs attention';
    q('#build-label').textContent = build.status === 'ready' ? `${build.pages || 0} pages · ${build.durationMs || 0}ms` : build.error || 'Build in progress';
    const errors = (build.diagnostics || []).filter(item => item.severity === 'error').length + (build.error ? 1 : 0);
    q('#check-label').textContent = errors ? `${errors} error${errors === 1 ? '' : 's'}` : 'No build errors';
  };
  const enhanceBlogImages = () => {
    const closeImageMenu = menu => {
      if (!menu || menu.hidden) return Promise.resolve(true);
      menu.hidden = true;
      menu.previousElementSibling?.setAttribute('aria-expanded', 'false');
      return Promise.resolve(menu.commitDraft?.());
    };
    const closeImageMenus = except => Promise.all(Array.from(document.querySelectorAll('.mpress-dev-image-menu'), menu => {
      if (menu !== except) return closeImageMenu(menu);
      return true;
    }));
    document.querySelectorAll('[data-blog-style-editor]').forEach(article => {
      if (article.querySelector(':scope > .mpress-dev-image-edit')) return;
      const articleLink = article.querySelector('.blog-hero-image, .blog-card-image, h2 a, h3 a');
      if (!articleLink) return;
      const edit = document.createElement('button');
      edit.className = 'mpress-dev-image-edit';
      edit.type = 'button';
      edit.title = 'Edit article appearance';
      edit.setAttribute('aria-label', 'Edit article appearance');
      edit.setAttribute('aria-haspopup', 'true');
      edit.setAttribute('aria-expanded', 'false');
      edit.innerHTML = iconHTML('pencil');
      const menu = document.createElement('div');
      menu.className = 'mpress-dev-image-menu';
      menu.hidden = true;
      menu.setAttribute('role', 'group');
      menu.setAttribute('aria-label', 'Article appearance');
      menu.innerHTML = '<div class="mpress-dev-image-controls" data-image-controls><span class="mpress-dev-image-section" data-image-mode-label>Image layout</span><button type="button" data-image-mode="panel">Panel</button><button type="button" data-image-mode="floating">Floating</button><div class="mpress-dev-image-panel-controls" data-image-panel-controls><span class="mpress-dev-image-section">Image fit</span><button type="button" data-image-fit="cover">Cover</button><button type="button" data-image-fit="contain">Fit</button><label class="mpress-dev-image-background"><span>Background</span><input type="color" aria-label="Image background colour"></label><button class="mpress-dev-image-reset" type="button" data-image-background-reset>Default</button></div></div><span class="mpress-dev-image-section">Tags</span><button type="button" data-tags-visible="true">Show</button><button type="button" data-tags-visible="false">Hide</button><span class="mpress-dev-image-section">Heading size</span><div class="mpress-dev-image-heading-options"><button type="button" data-heading-size="compact">Compact</button><button type="button" data-heading-size="default">Default</button><button type="button" data-heading-size="large">Large</button></div>';
      const route = new URL(articleLink.href, location.href).pathname;
      const backgroundInput = menu.querySelector('input[type="color"]');
      const backgroundReset = menu.querySelector('[data-image-background-reset]');
      const panelControls = menu.querySelector('[data-image-panel-controls]');
      const resizeHandle = document.createElement('button');
      resizeHandle.className = 'mpress-dev-image-resize';
      resizeHandle.type = 'button';
      resizeHandle.title = 'Drag to resize the floating image';
      resizeHandle.setAttribute('aria-label', 'Resize floating image');
      resizeHandle.setAttribute('aria-valuemin', '50');
      resizeHandle.setAttribute('aria-valuemax', '100');
      resizeHandle.innerHTML = iconHTML('grip-vertical');
      menu.querySelector('[data-image-controls]').hidden = !article.hasAttribute('data-blog-image-editor');
      const modeAvailable = article.classList.contains('blog-hero');
      menu.querySelector('[data-image-mode-label]').hidden = !modeAvailable;
      menu.querySelectorAll('[data-image-mode]').forEach(option => option.hidden = !modeAvailable);
      const positionResizeHandle = () => {
        if (resizeHandle.hidden || !resizeHandle.isConnected) return;
        const articleBounds = article.getBoundingClientRect();
        const imageBounds = articleLink.getBoundingClientRect();
        resizeHandle.style.left = `${imageBounds.right - articleBounds.left}px`;
        resizeHandle.style.top = `${imageBounds.top - articleBounds.top + imageBounds.height / 2}px`;
      };
      const defaultBackground = () => document.documentElement.dataset.theme === 'dark' || document.documentElement.dataset.theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches ? '#000000' : '#ffffff';
      const syncMode = mode => {
        article.dataset.blogImageMode = mode;
        panelControls.hidden = mode === 'floating';
        resizeHandle.hidden = !modeAvailable || mode !== 'floating';
        menu.querySelectorAll('[data-image-mode]').forEach(option => {
          const active = option.dataset.imageMode === mode;
          option.classList.toggle('active', active);
          option.setAttribute('aria-pressed', String(active));
        });
        requestAnimationFrame(positionResizeHandle);
      };
      const syncFit = fit => {
        article.dataset.blogImageFit = fit;
        menu.querySelectorAll('[data-image-fit]').forEach(option => {
          const active = option.dataset.imageFit === fit;
          option.classList.toggle('active', active);
          option.setAttribute('aria-pressed', String(active));
        });
      };
      const syncBackground = background => {
        if (background) {
          article.dataset.blogImageBackground = background;
          article.style.setProperty('--blog-image-background', background);
        } else {
          delete article.dataset.blogImageBackground;
          article.style.removeProperty('--blog-image-background');
        }
        backgroundInput.value = background || defaultBackground();
        backgroundReset.classList.toggle('active', !background);
      };
      const syncTags = visible => {
        article.dataset.blogTagsVisible = String(visible);
        menu.querySelectorAll('[data-tags-visible]').forEach(option => {
          const active = option.dataset.tagsVisible === String(visible);
          option.classList.toggle('active', active);
          option.setAttribute('aria-pressed', String(active));
        });
      };
      const syncHeading = size => {
        article.dataset.blogHeadingSize = size;
        menu.querySelectorAll('[data-heading-size]').forEach(option => {
          const active = option.dataset.headingSize === size;
          option.classList.toggle('active', active);
          option.setAttribute('aria-pressed', String(active));
        });
      };
      const normaliseImageWidth = value => Math.min(100, Math.max(50, Math.round(Number(value) || 100)));
      let savedImageWidth = normaliseImageWidth(article.dataset.blogImageWidth);
      let imageWidth = savedImageWidth;
      const syncImageWidth = width => {
        imageWidth = normaliseImageWidth(width);
        article.dataset.blogImageWidth = String(imageWidth);
        article.style.setProperty('--blog-floating-image-width', `${imageWidth}%`);
        resizeHandle.setAttribute('aria-valuenow', String(imageWidth));
        resizeHandle.setAttribute('aria-valuetext', `${imageWidth} percent`);
        requestAnimationFrame(positionResizeHandle);
      };
      const setBusy = busy => menu.querySelectorAll('button, input').forEach(control => control.disabled = busy);
      let savedMode = article.dataset.blogImageMode || 'panel';
      let savedFit = article.dataset.blogImageFit || 'cover';
      let savedBackground = article.dataset.blogImageBackground || '';
      let savedShowTags = article.dataset.blogTagsVisible !== 'false';
      let savedHeadingSize = article.dataset.blogHeadingSize || 'default';
      let draftMode = savedMode;
      let draftFit = savedFit;
      let draftBackground = savedBackground;
      let draftShowTags = savedShowTags;
      let draftHeadingSize = savedHeadingSize;
      let dirty = false;
      let savePromise = null;
      const syncDirty = () => {
        dirty = draftMode !== savedMode || draftFit !== savedFit || draftBackground !== savedBackground || draftShowTags !== savedShowTags || draftHeadingSize !== savedHeadingSize;
        menu.dataset.dirty = String(dirty);
      };
      const commitImageDraft = () => {
        if (savePromise) return savePromise;
        if (!dirty) return Promise.resolve(true);
        const nextMode = draftMode;
        const nextFit = draftFit;
        const nextBackground = nextMode === 'floating' ? '' : draftBackground;
        const nextShowTags = draftShowTags;
        const nextHeadingSize = draftHeadingSize;
        const previousMode = savedMode;
        const previousFit = savedFit;
        const previousBackground = savedBackground;
        const previousShowTags = savedShowTags;
        const previousHeadingSize = savedHeadingSize;
        const change = {};
        if (nextMode !== previousMode) change.mode = nextMode;
        if (nextFit !== previousFit) change.fit = nextFit;
        if (nextBackground !== previousBackground) change.background = nextBackground;
        if (nextShowTags !== previousShowTags) change.showTags = nextShowTags;
        if (nextHeadingSize !== previousHeadingSize) change.headingSize = nextHeadingSize;
        dirty = false;
        setBusy(true);
        state.suppressReloadUntil = Date.now() + 15000;
        savePromise = (async () => {
          try {
            await api('blog-image', {method: 'PUT', body: JSON.stringify({route, ...change}), keepalive: true});
            savedMode = nextMode;
            savedFit = nextFit;
            savedBackground = nextBackground;
            savedShowTags = nextShowTags;
            savedHeadingSize = nextHeadingSize;
            draftMode = nextMode;
            draftFit = nextFit;
            draftBackground = nextBackground;
            draftShowTags = nextShowTags;
            draftHeadingSize = nextHeadingSize;
            syncBackground(nextBackground);
            syncDirty();
            toast('Image appearance saved');
            return true;
          } catch (error) {
            state.suppressReloadUntil = 0;
            savedMode = previousMode;
            savedFit = previousFit;
            savedBackground = previousBackground;
            savedShowTags = previousShowTags;
            savedHeadingSize = previousHeadingSize;
            draftMode = previousMode;
            draftFit = previousFit;
            draftBackground = previousBackground;
            draftShowTags = previousShowTags;
            draftHeadingSize = previousHeadingSize;
            syncMode(previousMode);
            syncFit(previousFit);
            syncBackground(previousBackground);
            syncTags(previousShowTags);
            syncHeading(previousHeadingSize);
            syncDirty();
            toast(error.message, true);
            return false;
          } finally {
            savePromise = null;
            setBusy(false);
          }
        })();
        return savePromise;
      };
      let widthSavePromise = null;
      const commitImageWidth = () => {
        if (widthSavePromise) return widthSavePromise;
        if (imageWidth === savedImageWidth) return Promise.resolve(true);
        const nextWidth = imageWidth;
        const previousWidth = savedImageWidth;
        resizeHandle.disabled = true;
        state.suppressReloadUntil = Date.now() + 15000;
        widthSavePromise = (async () => {
          try {
            await api('blog-image', {method: 'PUT', body: JSON.stringify({route, width: nextWidth}), keepalive: true});
            savedImageWidth = nextWidth;
            toast('Image size saved');
            return true;
          } catch (error) {
            state.suppressReloadUntil = 0;
            syncImageWidth(previousWidth);
            toast(error.message, true);
            return false;
          } finally {
            widthSavePromise = null;
            resizeHandle.disabled = false;
          }
        })();
        return widthSavePromise;
      };
      let resizePointer = null;
      let resizeCenter = 0;
      let resizeTrackWidth = 0;
      const beginImageResize = event => {
        if (resizeHandle.hidden || event.button !== 0) return;
        event.preventDefault();
        event.stopPropagation();
        const articleBounds = article.getBoundingClientRect();
        const imageBounds = articleLink.getBoundingClientRect();
        const copyBounds = article.querySelector('.blog-hero-copy')?.getBoundingClientRect();
        const desktopTrackWidth = copyBounds && copyBounds.left > articleBounds.left + 2 ? copyBounds.left - articleBounds.left : articleBounds.width;
        resizeCenter = imageBounds.left + imageBounds.width / 2;
        resizeTrackWidth = Math.max(imageBounds.width, desktopTrackWidth);
        resizePointer = event.pointerId;
        resizeHandle.classList.add('active');
        resizeHandle.setPointerCapture?.(event.pointerId);
      };
      const moveImageResize = event => {
        if (event.pointerId !== resizePointer) return;
        event.preventDefault();
        const nextWidth = Math.abs(event.clientX - resizeCenter) * 2 / resizeTrackWidth * 100;
        syncImageWidth(nextWidth);
      };
      const finishImageResize = event => {
        if (event.pointerId !== resizePointer) return;
        event.preventDefault();
        event.stopPropagation();
        resizeHandle.releasePointerCapture?.(event.pointerId);
        resizePointer = null;
        resizeHandle.classList.remove('active');
        void commitImageWidth();
      };
      resizeHandle.addEventListener('pointerdown', beginImageResize);
      addEventListener('pointermove', moveImageResize);
      addEventListener('pointerup', finishImageResize);
      addEventListener('pointercancel', event => {
        if (event.pointerId !== resizePointer) return;
        resizePointer = null;
        resizeHandle.classList.remove('active');
        syncImageWidth(savedImageWidth);
      });
      resizeHandle.addEventListener('keydown', event => {
        if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
        event.preventDefault();
        event.stopPropagation();
        syncImageWidth(imageWidth + (event.key === 'ArrowRight' ? 2 : -2));
      });
      resizeHandle.addEventListener('keyup', event => {
        if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') void commitImageWidth();
      });
      articleLink.querySelector('img')?.addEventListener('load', positionResizeHandle);
      addEventListener('resize', positionResizeHandle);
      menu.commitDraft = commitImageDraft;
      article.append(edit, menu, resizeHandle);
      syncMode(savedMode);
      syncFit(savedFit);
      syncBackground(savedBackground);
      syncTags(savedShowTags);
      syncHeading(savedHeadingSize);
      syncImageWidth(savedImageWidth);
      syncDirty();
      edit.addEventListener('click', event => {
        event.preventDefault();
        event.stopPropagation();
        const opening = menu.hidden;
        if (opening) {
          void closeImageMenus(menu);
          menu.hidden = false;
          edit.setAttribute('aria-expanded', 'true');
          menu.querySelector('.active')?.focus();
        } else {
          void closeImageMenu(menu);
        }
      });
      menu.addEventListener('click', event => {
        const modeOption = event.target.closest('[data-image-mode]');
        if (modeOption) {
          event.preventDefault();
          event.stopPropagation();
          const mode = modeOption.dataset.imageMode;
          if (mode === draftMode) return;
          draftMode = mode;
          syncMode(mode);
          syncDirty();
          return;
        }
        const option = event.target.closest('[data-image-fit]');
        if (option) {
          event.preventDefault();
          event.stopPropagation();
          const fit = option.dataset.imageFit;
          if (fit === draftFit) return;
          draftFit = fit;
          syncFit(fit);
          syncDirty();
          return;
        }
        const tagsOption = event.target.closest('[data-tags-visible]');
        if (tagsOption) {
          event.preventDefault();
          event.stopPropagation();
          const visible = tagsOption.dataset.tagsVisible === 'true';
          if (visible === draftShowTags) return;
          draftShowTags = visible;
          syncTags(visible);
          syncDirty();
          return;
        }
        const headingOption = event.target.closest('[data-heading-size]');
        if (!headingOption) return;
        event.preventDefault();
        event.stopPropagation();
        const size = headingOption.dataset.headingSize;
        if (size === draftHeadingSize) return;
        draftHeadingSize = size;
        syncHeading(size);
        syncDirty();
      });
      backgroundInput.addEventListener('input', event => {
        draftBackground = event.target.value;
        syncBackground(draftBackground);
        syncDirty();
      });
      backgroundInput.addEventListener('change', event => {
        draftBackground = event.target.value;
        syncBackground(draftBackground);
        syncDirty();
      });
      backgroundReset.addEventListener('click', event => {
        event.preventDefault();
        event.stopPropagation();
        if (!draftBackground) return;
        draftBackground = '';
        syncBackground('');
        syncDirty();
      });
      menu.addEventListener('keydown', event => {
        if (event.key !== 'Escape') return;
        event.preventDefault();
        void closeImageMenu(menu);
        edit.focus();
      });
    });
    addEventListener('click', event => {
      if (event.target.closest?.('.mpress-dev-image-edit, .mpress-dev-image-menu')) return;
      const link = event.target.closest?.('a[href]');
      const hasDirtyMenu = document.querySelector('.mpress-dev-image-menu[data-dirty="true"]');
      if (link && hasDirtyMenu && !link.hasAttribute('download') && link.target !== '_blank' && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) {
        event.preventDefault();
        void closeImageMenus(null).then(results => {
          if (results.every(Boolean)) location.assign(link.href);
        });
        return;
      }
      void closeImageMenus(null);
    });
  };
  const showConfig = async (fromWizard = false, initialPage = '', initialBlogView = '') => {
    const configPages = {
      quick: {title: 'Quick setup', description: 'Change the site details and appearance that you use most often.'},
      site: {title: 'Site details', description: 'Set the site name, public address, and keyboard shortcuts.'},
      languages: {title: 'Languages', description: 'Add languages and choose what happens when a translation is missing.'},
      brand: {title: 'Brand', description: 'Upload the images that identify the site in light and dark mode.'},
      social: {title: 'Navigation and links', description: 'Manage header links, public channels, and reader contributions.'},
      build: {title: 'Project files', description: 'Change where M-Press finds source files and writes the generated site.'},
      blog: {title: 'Blog posts', description: 'Create and edit Markdown articles.'},
      theme: {title: 'Appearance', description: 'Set the colour scheme, interaction colours, and documentation search.'},
      accessibility: {title: 'Accessibility', description: 'Give every visitor local controls for reading, focus, motion, contrast, and colour.'},
      layout: {title: 'Layout', description: 'Choose a documentation layout preset or configure custom dimensions.'},
      versioning: {title: 'Versions', description: 'Create fixed documentation releases and keep older versions available.'},
      translation: {title: 'Translation provider', description: 'Connect the provider that M-Press uses for machine translation.'},
      deploy: {title: 'Deployment targets', description: 'Configure saved Cloudflare Pages and Netlify destinations.'}
    };
    const activeConfig = configPages[initialPage || 'quick'];
    state.wizardActive = fromWizard;
    if (fromWizard) sessionStorage.setItem('mpress-wizard-step', '2');
    try {
      const data = await api('config');
      const cfg = data.config;
      beginConfigPreview(data.title);
      const quickFormTemplate = await api('config-form');
      const formSections = ['site', 'languages', 'brand', 'social', 'build', 'blog', 'theme', 'accessibility', 'layout', 'versioning', 'translation', 'deploy'];
      const settingsFormTemplates = Object.fromEntries(await Promise.all(formSections.map(async section => [section, await api(`config-form?section=${section}`)])));
      // Site fields are supplied by the Markdown form template. Keep the
      // complete-form field names explicit here for integrations that inspect
      // the development bundle before the template is hydrated.
      // name="search.shortcut" name="accessibility.shortcut" complete.search.shortcut complete.accessibility.shortcut
      const languages = (cfg.site.languages || []).map(code => languageRow(code, cfg.site.languageLabels?.[code] || '')).join('');
      const headerLinks = (cfg.site.headerLinks || []).map(link => headerLinkRow(link.label, link.url, link.type, link.variant, link.color)).join('');
      const deployTargets = Object.entries(cfg.deploy?.targets || {}).map(([name, target]) => deployTargetRow(name, target)).join('');
      openDrawer('Configuration', activeConfig.title, `
        <div class="config-intro"><p class="intro">Use quick setup for everyday changes, or open all settings to configure every option M-Press supports. Every field is validated before the site is rebuilt.</p><span class="live-preview-note">${workspaceMode ? 'Theme changes preview across this workspace' : 'Changes preview on the site behind this window'} · Save changes to rebuild the site.</span></div>
        <div class="config-tabs" role="tablist" aria-label="Configuration view">
          <button class="config-tab active" type="button" role="tab" aria-selected="true" data-config-tab="guided">Quick setup</button>
          <button class="config-tab" type="button" role="tab" aria-selected="false" data-config-tab="complete">All settings</button>
        </div>
        <div class="config-panel" data-config-panel="guided">
          ${quickFormTemplate.html}
          <div class="settings-actions settings-actions-guided"><div><button class="button" id="config-reset-guided" type="button">Revert changes</button><button class="button primary" type="submit" form="config-form">Save and rebuild</button></div></div>
        </div>
        <div class="config-panel" data-config-panel="complete" hidden>
          <div class="settings-workspace">
            <nav class="settings-nav" role="tablist" aria-label="Configuration sections">
              <button class="settings-nav-button active" type="button" role="tab" aria-selected="true" aria-controls="settings-site" data-settings-tab="site">Site</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-languages" data-settings-tab="languages">Languages</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-brand" data-settings-tab="brand">Brand</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-social" data-settings-tab="social">Links</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-build" data-settings-tab="build">Files</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-blog" data-settings-tab="blog">Blog</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-theme" data-settings-tab="theme">Theme</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-accessibility" data-settings-tab="accessibility">Accessibility</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-layout" data-settings-tab="layout">Layout</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-versioning" data-settings-tab="versioning">Versions</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-translation" data-settings-tab="translation">Translation</button>
              <button class="settings-nav-button" type="button" role="tab" aria-selected="false" aria-controls="settings-deploy" data-settings-tab="deploy">Deploy</button>
            </nav>
            <form id="config-complete-form" class="form-grid complete-settings">
            <section class="settings-section" id="settings-site" role="tabpanel" data-settings-page="site"><div class="settings-grid mpress-settings-form-fields settings-form-site">${settingsFormTemplates.site.html}</div></section>

            <section class="settings-section" id="settings-languages" role="tabpanel" data-settings-page="languages" hidden><div class="settings-grid mpress-settings-form-fields settings-form-languages">${settingsFormTemplates.languages.html}</div><div class="repeat-section language-section"><div class="repeat-heading"><h2>Configured languages</h2><button class="button" id="add-language" type="button">Add language</button></div><div class="language-table"><div class="language-table-head" aria-hidden="true"><span>Code</span><span>Display label</span><span></span></div><div class="repeat-list" id="language-list" aria-label="Configured languages">${languages}</div></div></div></section>

            <section class="settings-section" id="settings-brand" role="tabpanel" data-settings-page="brand" hidden><h2 class="settings-section-title">Logos and images</h2><div class="brand-preview settings-wide"><div><span>Light mode</span><img data-brand-preview="light" alt="Light-mode logo preview"></div><div><span>Dark mode</span><img data-brand-preview="dark" alt="Dark-mode logo preview"></div></div><div class="settings-grid mpress-settings-form-fields settings-form-brand">${settingsFormTemplates.brand.html}</div></section>

            <section class="settings-section" id="settings-social" role="tabpanel" data-settings-page="social" hidden><div class="settings-grid mpress-settings-form-fields settings-form-social">${settingsFormTemplates.social.html}</div><div class="repeat-section"><div class="repeat-heading"><h2>Header links</h2><button class="button" id="add-header-link" type="button">Add header link</button></div><div class="language-table header-link-table"><div class="language-table-head header-link-table-head" aria-hidden="true"><span>Label</span><span>URL</span><span></span></div><div class="repeat-list" id="header-link-list" aria-label="Header links">${headerLinks || '<p class="empty-repeat">No header links yet.</p>'}</div></div></div></section>

            <section class="settings-section" id="settings-build" role="tabpanel" data-settings-page="build" hidden><aside class="settings-notice" role="note"><strong>Advanced settings</strong><span>Changing these paths can stop the site from building. Move the matching files before you save.</span></aside><div class="settings-grid mpress-settings-form-fields settings-form-build">${settingsFormTemplates.build.html}</div></section>

            <section class="settings-section" id="settings-blog" role="tabpanel" data-settings-page="blog" hidden><div data-blog-settings-panel="posts"><div class="settings-section-toolbar settings-section-toolbar-actions"><button class="button primary" id="blog-new-post" type="button">New post</button></div><div class="content-list" id="blog-post-list" aria-live="polite"><p class="empty-repeat">Loading blog posts.</p></div><div id="blog-post-editor" hidden></div></div><div data-blog-settings-panel="preferences" hidden><div class="settings-section-heading"><h2 class="settings-section-title">Blog presentation</h2><p class="settings-section-copy">Set the archive defaults. A post can override them in its frontmatter.</p></div><div class="settings-grid mpress-settings-form-fields settings-form-blog">${settingsFormTemplates.blog.html}</div></div></section>

            <section class="settings-section" id="settings-theme" role="tabpanel" data-settings-page="theme" hidden><div class="settings-grid mpress-settings-form-fields settings-form-theme">${settingsFormTemplates.theme.html}</div></section>

            <section class="settings-section" id="settings-accessibility" role="tabpanel" data-settings-page="accessibility" hidden><div class="settings-grid mpress-settings-form-fields settings-form-accessibility">${settingsFormTemplates.accessibility.html}</div><div class="feature-summary"><strong>Visitor controls</strong><span>The menu includes reading focus, motion, contrast, colour, type, and spacing preferences. Each visitor's settings stay in their browser.</span></div></section>

            <section class="settings-section" id="settings-layout" role="tabpanel" data-settings-page="layout" hidden><div class="settings-grid mpress-settings-form-fields settings-form-layout">${settingsFormTemplates.layout.html}</div></section>

            <section class="settings-section" id="settings-versioning" role="tabpanel" data-settings-page="versioning" hidden><div class="settings-grid mpress-settings-form-fields settings-form-versioning">${settingsFormTemplates.versioning.html}</div><div id="version-status" class="feature-summary" aria-live="polite"><strong>Loading version status</strong><span>M-Press is checking the captured builds.</span></div><div class="repeat-section version-manager"><div class="settings-section-toolbar"><div><h2>Captured versions</h2><p>Each version is a fixed copy of a successful build.</p></div><button class="button primary" id="version-new" type="button">Create version</button></div><div class="inline-create-form" id="version-create-form" hidden><label class="field"><span>Version label</span><input data-version-label pattern="[A-Za-z0-9][A-Za-z0-9._-]*" placeholder="3.0"></label><div><button class="button" type="button" id="version-create-cancel">Cancel</button><button class="button primary" type="button" id="version-create-submit">Capture build</button></div><p class="inline-form-error error-copy" id="version-create-error" role="alert" hidden></p></div><div class="content-list" id="version-list" aria-live="polite"><p class="empty-repeat">Loading versions.</p></div></div></section>

            <section class="settings-section" id="settings-translation" role="tabpanel" data-settings-page="translation" hidden><div class="settings-grid mpress-settings-form-fields settings-form-translation"><div class="translation-workspace-entry settings-wide"><div><strong>Translation workspace</strong><span>Add a language, update translated text, or record a human review.</span></div><button class="button" id="open-translation-workspace" type="button">Open translation tools</button></div>${settingsFormTemplates.translation.html}</div></section>

            <section class="settings-section" id="settings-deploy" role="tabpanel" data-settings-page="deploy" hidden><aside class="settings-notice" role="note"><strong>Credentials stay outside this file</strong><span>M-Press stores target names and project identifiers in YAML. It reads deployment credentials from your environment.</span></aside><div class="settings-grid mpress-settings-form-fields settings-form-deploy">${settingsFormTemplates.deploy.html}</div><div class="repeat-section"><div class="repeat-heading"><h2>Saved targets</h2><button class="button" id="add-deploy-target" type="button">Add target</button></div><div class="repeat-list" id="deploy-target-list">${deployTargets || '<p class="empty-repeat">No deployment targets yet.</p>'}</div></div></section>

            <div class="settings-actions"><span class="settings-step-label" id="settings-step-label">Site details · 1 of 12</span><div><button class="button" id="config-reset-complete" type="button">Revert changes</button><button class="button" id="settings-previous" type="button" disabled>Previous</button><button class="button" id="settings-next" type="button">Next</button><button class="button primary" type="submit">Save and rebuild</button></div></div>
            </form>
          </div>
        </div>`, 'configuration', activeConfig.description);
      const quickForm = q('#config-form');
      if (quickForm) {
        const setQuickValue = (name, value) => {
          const field = quickForm.elements.namedItem(name);
          if (field && value !== undefined && value !== null) field.value = String(value);
        };
        setQuickValue('title', data.title);
        setQuickValue('description', data.description);
        setQuickValue('baseURL', data.baseURL);
        setQuickValue('colorScheme', data.colorScheme);
        setQuickValue('accentColor', data.accentColor || '#5375f6');
        setQuickValue('hoverColorLight', data.hoverColorLight || data.hoverColor || '#7593ff');
        setQuickValue('hoverColorDark', data.hoverColorDark || data.hoverColor || '#7593ff');
      }
      const siteSettings = q('#settings-site');
      if (siteSettings) {
        const setSiteValue = (name, value) => {
          const field = siteSettings.querySelector(`[name="${name}"]`);
          if (field && value !== undefined && value !== null) field.value = String(value);
        };
        setSiteValue('site.title', cfg.site.title);
        setSiteValue('site.description', cfg.site.description);
        setSiteValue('site.baseURL', cfg.site.baseURL);
        setSiteValue('site.logoLight', cfg.site.logoLight);
        setSiteValue('site.logoDark', cfg.site.logoDark);
        setSiteValue('site.logoWidth', cfg.site.logoWidth || '160px');
        setSiteValue('accessibility.shortcut', cfg.accessibility?.shortcut || 'Mod+A');
      }
      const setSettingsValue = (section, name, value) => {
        const field = q(`[data-settings-page="${section}"] [name="${name}"]`);
        if (field && value !== undefined && value !== null) field.value = String(value);
      };
      const setSettingsChecked = (section, name, value) => {
        const field = q(`[data-settings-page="${section}"] [name="${name}"]`);
        if (field) field.checked = Boolean(value);
      };
      setSettingsValue('languages', 'site.defaultLanguage', cfg.site.defaultLanguage);
      setSettingsValue('languages', 'site.missingTranslation', cfg.site.missingTranslation);
      setSettingsChecked('languages', 'site.defaultLanguageAtRoot', cfg.site.defaultLanguageAtRoot);
      setSettingsValue('brand', 'site.logoLight', cfg.site.logoLight);
      setSettingsValue('brand', 'site.logoDark', cfg.site.logoDark);
      setSettingsValue('brand', 'site.favicon', cfg.site.favicon);
      setSettingsValue('brand', 'site.socialImage', cfg.site.socialImage || '');
      const brandPreviewURL = value => {
        const source = String(value || '').trim();
        if (!source) return '';
        if (/^(?:https?:|data:|blob:)/i.test(source) || source.startsWith('/')) return source;
        return `/${source.replace(/^\/+/, '')}`;
      };
      const setBrandPreview = (mode, source) => {
        const image = q(`[data-settings-page="brand"] [data-brand-preview="${mode}"]`);
        if (!image) return;
        const url = brandPreviewURL(source);
        image.src = url;
        image.hidden = !url;
      };
      const imageAssetName = value => {
        const clean = String(value || 'image').split('/').pop().replace(/\.[^.]+$/, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'image';
        return `${clean}-edited.png`;
      };
      const loadImageSource = source => new Promise((resolve, reject) => {
        const image = new Image();
        image.onload = () => resolve(image);
        image.onerror = () => reject(new Error('M-Press could not read this image.'));
        image.src = source;
      });
      const openImageEditor = async ({file, source = '', category = 'brand', suggestedName = '', title = 'Edit image', onSave}) => {
        let sourceURL = file ? URL.createObjectURL(file) : source;
        if (!sourceURL) throw new Error('Choose an image first.');
        let image = await loadImageSource(sourceURL);
        const overlay = document.createElement('div');
        overlay.className = 'image-editor-overlay';
        overlay.innerHTML = `<div class="image-editor-dialog" role="dialog" aria-modal="true" aria-labelledby="image-editor-title"><div class="image-editor-top"><div><span class="drawer-eyebrow">Image editor</span><h2 id="image-editor-title">${escapeHTML(title)}</h2></div><button class="icon-button" type="button" data-image-editor-close aria-label="Close image editor">${iconHTML('x')}</button></div><div class="image-editor-layout"><div class="image-editor-stage"><canvas width="1200" height="675" aria-label="Edited image preview"></canvas></div><div class="image-editor-controls"><label class="field"><span>Source image</span><input type="file" data-image-source accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif"></label><label class="field"><span>Crop</span><select data-image-crop><option value="original">Original shape</option><option value="16:9" selected>Wide, 16:9</option><option value="4:3">Standard, 4:3</option><option value="1:1">Square, 1:1</option></select></label><label class="field"><span>Size</span><input data-image-zoom type="range" min="100" max="220" value="100"><small>Zoom the image inside the crop.</small></label><div class="form-row"><label class="field"><span>Horizontal position</span><input data-image-x type="range" min="-100" max="100" value="0"></label><label class="field"><span>Vertical position</span><input data-image-y type="range" min="-100" max="100" value="0"></label></div><label class="field"><span>Brightness</span><input data-image-brightness type="range" min="50" max="150" value="100"><small>Darken or lighten the complete image.</small></label><div class="form-row"><label class="field"><span>Text overlay</span><input data-image-text type="text" placeholder="Optional caption"></label><label class="field"><span>Text colour</span><input data-image-text-color type="color" value="#ffffff"></label></div><label class="field"><span>Save as</span><input data-image-filename value="${escapeHTML(suggestedName || imageAssetName(file?.name || source))}" spellcheck="false"></label><div class="image-editor-conflict" data-image-conflict hidden><strong>This filename already exists.</strong><span>Save the suggested copy, or confirm that you want to replace the existing image.</span></div></div></div><div class="image-editor-actions"><button class="button" type="button" data-image-editor-close>Cancel</button><button class="button danger-quiet" type="button" data-image-overwrite hidden>Overwrite existing</button><button class="button primary" type="button" data-image-save>Save image</button></div></div>`;
        root.append(overlay);
        overlay.querySelector('[data-image-crop]').value = category === 'blog' ? '16:9' : 'original';
        lockPageScroll();
        const canvas = overlay.querySelector('canvas');
        const context = canvas.getContext('2d');
        const value = name => overlay.querySelector(`[data-image-${name}]`)?.value || '';
        const draw = () => {
          const ratios = {'16:9': 16 / 9, '4:3': 4 / 3, '1:1': 1};
          const ratio = ratios[value('crop')] || image.naturalWidth / image.naturalHeight || 16 / 9;
          const width = 1200;
          const height = Math.max(400, Math.round(width / ratio));
          canvas.width = width;
          canvas.height = height;
          context.clearRect(0, 0, width, height);
          const zoom = Number(value('zoom')) / 100;
          const scale = Math.max(width / image.naturalWidth, height / image.naturalHeight) * zoom;
          const drawWidth = image.naturalWidth * scale;
          const drawHeight = image.naturalHeight * scale;
          const overflowX = Math.max(0, drawWidth - width);
          const overflowY = Math.max(0, drawHeight - height);
          const x = (width - drawWidth) / 2 - (Number(value('x')) / 100) * overflowX / 2;
          const y = (height - drawHeight) / 2 - (Number(value('y')) / 100) * overflowY / 2;
          context.filter = `brightness(${Number(value('brightness')) || 100}%)`;
          context.drawImage(image, x, y, drawWidth, drawHeight);
          context.filter = 'none';
          const caption = value('text').trim();
          if (caption) {
            context.font = '700 54px system-ui, sans-serif';
            context.textBaseline = 'bottom';
            context.shadowColor = 'rgba(0,0,0,.72)';
            context.shadowBlur = 18;
            context.fillStyle = value('text-color');
            context.fillText(caption, 56, height - 48, width - 112);
            context.shadowBlur = 0;
          }
        };
        overlay.querySelectorAll('input, select').forEach(control => control.addEventListener('input', draw));
        installStyledSelects(overlay);
        const close = () => { overlay.remove(); unlockPageScroll(); if (sourceURL.startsWith('blob:')) URL.revokeObjectURL(sourceURL); };
        overlay.querySelectorAll('[data-image-editor-close]').forEach(button => button.addEventListener('click', close));
        overlay.addEventListener('click', event => { if (event.target === overlay) close(); });
        overlay.querySelector('[data-image-source]').addEventListener('change', async event => {
          const replacement = event.target.files?.[0];
          if (!replacement) return;
          if (sourceURL.startsWith('blob:')) URL.revokeObjectURL(sourceURL);
          sourceURL = URL.createObjectURL(replacement);
          try { image = await loadImageSource(sourceURL); overlay.querySelector('[data-image-filename]').value = imageAssetName(replacement.name); draw(); }
          catch (error) { toast(error.message, true); }
        });
        let conflictName = '';
        const save = async overwrite => {
          const saveButton = overlay.querySelector('[data-image-save]');
          const overwriteButton = overlay.querySelector('[data-image-overwrite]');
          const filename = value('filename').trim();
          if (!/^[A-Za-z0-9][A-Za-z0-9._-]*\.(?:png|jpe?g|webp)$/i.test(filename)) { overlay.querySelector('[data-image-filename]').focus(); toast('Use a valid PNG, JPEG, or WebP filename', true); return; }
          saveButton.disabled = true;
          overwriteButton.disabled = true;
          saveButton.textContent = 'Saving';
          const mimeType = /\.webp$/i.test(filename) ? 'image/webp' : /\.jpe?g$/i.test(filename) ? 'image/jpeg' : 'image/png';
          const blob = await new Promise(resolve => canvas.toBlob(resolve, mimeType, .92));
          const body = new FormData();
          body.append('category', category);
          body.append('filename', filename);
          body.append('overwrite', String(overwrite));
          body.append('file', blob, filename);
          try {
            state.suppressReloadUntil = Date.now() + 15000;
            const result = await api('image-asset', {method: 'POST', body});
            close();
            onSave?.(result.path);
            toast('Image saved');
          } catch (error) {
            if (error.status === 409) {
              conflictName = error.payload?.filename || filename;
              overlay.querySelector('[data-image-conflict]').hidden = false;
              overwriteButton.hidden = false;
              overlay.querySelector('[data-image-filename]').value = error.payload?.suggestedName || imageAssetName(filename);
              saveButton.textContent = 'Save suggested copy';
            } else {
              toast(error.message, true);
              saveButton.textContent = 'Save image';
            }
            saveButton.disabled = false;
            overwriteButton.disabled = false;
          }
        };
        overlay.querySelector('[data-image-save]').addEventListener('click', () => void save(false));
        overlay.querySelector('[data-image-overwrite]').addEventListener('click', () => {
          if (conflictName) overlay.querySelector('[data-image-filename]').value = conflictName;
          void save(true);
        });
        draw();
        overlay.querySelector('[data-image-crop]')?.mpressSelectSync?.();
      };
      setBrandPreview('light', cfg.site.logoLight);
      setBrandPreview('dark', cfg.site.logoDark);
      const installBrandUpload = ({inputName, pathName, title, suggestedName, previewMode = ''}) => {
        const input = q(`[data-settings-page="brand"] [name="${inputName}"]`);
        const pathField = q(`[data-settings-page="brand"] [name="${pathName}"]`);
        if (!input || !pathField) return;
        const field = input.closest('.mpress-form-field');
        const status = document.createElement('small');
        status.className = 'brand-upload-status';
        status.textContent = pathField.value ? `Current image: ${pathField.value}` : 'Choose an image to add it to the project.';
        const editCurrent = document.createElement('button');
        editCurrent.type = 'button';
        editCurrent.className = 'button image-field-action';
        editCurrent.textContent = 'Edit current image';
        editCurrent.hidden = !pathField.value;
        field?.append(editCurrent, status);
        const edit = async ({file, source = '', name = suggestedName} = {}) => {
          if (!file && !source) source = brandPreviewURL(pathField.value);
          if (!file && !source) {
            input.click();
            return;
          }
          if (file && previewMode) setBrandPreview(previewMode, URL.createObjectURL(file));
          try {
            await openImageEditor({file, source, category: 'brand', suggestedName: name, title, onSave: savedPath => {
              pathField.value = savedPath;
              pathField.dispatchEvent(new Event('input', {bubbles: true}));
              if (previewMode) setBrandPreview(previewMode, savedPath);
              editCurrent.hidden = false;
              status.dataset.state = 'success';
              status.textContent = `Saved ${savedPath}`;
            }});
          } catch (error) {
            status.dataset.state = 'error';
            status.textContent = error.message;
            toast(error.message, true);
          }
        };
        input.addEventListener('change', async () => {
          const file = input.files?.[0];
          if (!file) return;
          await edit({file});
          input.value = '';
        });
        editCurrent.addEventListener('click', () => void edit());
        return {edit, field};
      };
      installBrandUpload({inputName: 'site.logoLightUpload', pathName: 'site.logoLight', title: 'Edit light-mode logo', suggestedName: 'logo-light.png', previewMode: 'light'});
      installBrandUpload({inputName: 'site.logoDarkUpload', pathName: 'site.logoDark', title: 'Edit dark-mode logo', suggestedName: 'logo-dark.png', previewMode: 'dark'});
      installBrandUpload({inputName: 'site.faviconUpload', pathName: 'site.favicon', title: 'Edit favicon', suggestedName: 'favicon.png'});
      const socialUpload = installBrandUpload({inputName: 'site.socialImageUpload', pathName: 'site.socialImage', title: 'Edit social sharing image', suggestedName: 'social-image.png'});
      if (socialUpload) {
        const generate = document.createElement('button');
        generate.type = 'button';
        generate.className = 'button image-field-action';
        generate.textContent = 'Generate from site';
        socialUpload.field?.append(generate);
        generate.addEventListener('click', async () => {
          const canvas = document.createElement('canvas');
          canvas.width = 1200;
          canvas.height = 630;
          const context = canvas.getContext('2d');
          const title = q('[name="site.title"]')?.value.trim() || cfg.site.title || 'Documentation';
          const description = q('[name="site.description"]')?.value.trim() || cfg.site.description || '';
          const accent = q('[name="theme.accentColor"]')?.value || cfg.theme.accentColor || '#5375f6';
          context.fillStyle = '#0b111b';
          context.fillRect(0, 0, canvas.width, canvas.height);
          context.fillStyle = accent;
          context.fillRect(0, 0, 18, canvas.height);
          context.fillRect(72, 82, 96, 8);
          context.fillStyle = '#f5f7fb';
          context.font = '700 64px system-ui, sans-serif';
          context.fillText(title, 72, 320, 1056);
          context.fillStyle = '#aeb9c9';
          context.font = '400 30px system-ui, sans-serif';
          const words = description.split(/\s+/).filter(Boolean);
          let line = '';
          let y = 382;
          for (const word of words) {
            const next = line ? `${line} ${word}` : word;
            if (context.measureText(next).width > 1000 && line) {
              context.fillText(line, 72, y);
              line = word;
              y += 43;
              if (y > 440) break;
            } else line = next;
          }
          if (line && y <= 440) context.fillText(line, 72, y);
          const logoSource = brandPreviewURL(cfg.site.logoDark || cfg.site.logoLight);
          if (logoSource) {
            try {
              const logo = await loadImageSource(logoSource);
              const scale = Math.min(300 / logo.naturalWidth, 72 / logo.naturalHeight, 1);
              context.drawImage(logo, 72, 120, logo.naturalWidth * scale, logo.naturalHeight * scale);
            } catch (_) {}
          }
          await socialUpload.edit({source: canvas.toDataURL('image/png'), name: 'social-image.png'});
        });
      }
      for (const [name, value] of Object.entries({
        'social.github': cfg.social.github, 'social.discord': cfg.social.discord, 'social.reddit': cfg.social.reddit,
        'social.x': cfg.social.x, 'social.rss': cfg.social.rss, 'social.sponsor': cfg.social.sponsor,
        'social.editURL': cfg.social.editURL, 'contribution.repository': cfg.contribution?.repository || '',
        'contribution.branch': cfg.contribution?.branch || 'main', 'contribution.guide': cfg.contribution?.guide || 'CONTRIBUTING.md'
      })) setSettingsValue('social', name, value);
      setSettingsChecked('social', 'contribution.enabled', cfg.contribution?.enabled);
      setSettingsChecked('social', 'contribution.quickEdit', cfg.contribution?.quickEdit);
      const editURLField = q('[data-settings-page="social"] [name="social.editURL"]')?.closest('.mpress-form-field');
      if (editURLField && !editURLField.querySelector('small')) {
        const hint = document.createElement('small');
        hint.textContent = 'Optional. M-Press appends each source path to create the Edit page link.';
        editURLField.append(hint);
      }
      for (const [name, value] of Object.entries({
        'build.contentDir': cfg.build.contentDir, 'build.staticDir': cfg.build.staticDir,
        'build.outputDir': cfg.build.outputDir, 'build.navFile': cfg.build.navFile, 'build.customCSS': cfg.build.customCSS
      })) setSettingsValue('build', name, value);
      setSettingsChecked('build', 'knowledge.enabled', cfg.knowledge?.enabled !== false);
      setSettingsChecked('blog', 'blog.showTags', cfg.blog?.showTags !== false);
      for (const [name, value] of Object.entries({
        'blog.landingStyle': cfg.blog?.landingStyle || 'featured', 'blog.tagline': cfg.blog?.tagline || '',
        'blog.headingSize': cfg.blog?.headingSize || 'default', 'blog.imageMode': cfg.blog?.imageMode || 'panel',
        'blog.imageFit': cfg.blog?.imageFit || 'cover', 'blog.imageWidth': cfg.blog?.imageWidth || 100,
        'blog.imageBackground': cfg.blog?.imageBackground || ''
      })) setSettingsValue('blog', name, value);
      for (const [name, value] of Object.entries({
        'theme.colorScheme': cfg.theme.colorScheme, 'theme.accentColor': cfg.theme.accentColor || '#5375f6',
        'theme.hoverColorLight': cfg.theme.hoverColorLight || cfg.theme.hoverColor || '#7593ff',
        'theme.hoverColorDark': cfg.theme.hoverColorDark || cfg.theme.hoverColor || '#7593ff',
        'search.placeholder': cfg.search?.placeholder || 'Search documentation',
        'search.maxResults': cfg.search?.maxResults || 12,
        'search.shortcut': cfg.search?.shortcut || 'Mod+K'
      })) setSettingsValue('theme', name, value);
      setSettingsChecked('theme', 'search.enabled', cfg.search.enabled);
      setSettingsChecked('theme', 'search.rememberRecent', cfg.search?.rememberRecent !== false);
      installShortcutCapture(q('#config-complete-form'));
      setSettingsChecked('accessibility', 'accessibility.enabled', cfg.accessibility?.enabled !== false);
      for (const [name, value] of Object.entries({
        'theme.layout.preset': cfg.theme.layout?.preset || 'starlight',
        'theme.layout.contentWidth': cfg.theme.layout?.contentWidth || '45rem',
        'theme.layout.wideContentWidth': cfg.theme.layout?.wideContentWidth || '64rem',
        'theme.layout.sidebarWidth': cfg.theme.layout?.sidebarWidth || '18.75rem',
        'theme.layout.tocWidth': cfg.theme.layout?.tocWidth || '16rem',
        'theme.layout.contentTocGap': cfg.theme.layout?.contentTocGap || '2.5rem',
        'theme.layout.alignment': cfg.theme.layout?.alignment || 'cluster',
        'theme.layout.toc': cfg.theme.layout?.toc || 'right'
      })) setSettingsValue('layout', name, value);
      setSettingsChecked('versioning', 'versioning.enabled', cfg.versioning.enabled);
      setSettingsValue('versioning', 'versioning.current', cfg.versioning.current);
      setSettingsValue('build', 'versioning.artifactsDir', cfg.versioning.artifactsDir);
      for (const [name, value] of Object.entries({
        'translation.provider': cfg.translation.provider, 'translation.model': cfg.translation.model, 'translation.reasoningEffort': cfg.translation.reasoningEffort || '', 'translation.command': cfg.translation.command || '',
        'translation.baseURL': cfg.translation.baseURL, 'translation.apiKeyEnv': cfg.translation.apiKeyEnv,
        'translation.sourceLanguage': cfg.translation.sourceLanguage, 'translation.glossary': cfg.translation.glossary,
        'translation.styleGuide': cfg.translation.styleGuide, 'translation.stateDir': cfg.translation.stateDir,
		'translation.inputPricePerMillion': cfg.translation.inputPricePerMillion || 0,
		'translation.outputPricePerMillion': cfg.translation.outputPricePerMillion || 0,
        'translation.dataCollection': cfg.translation.dataCollection || ''
      })) setSettingsValue('translation', name, value);
      setSettingsChecked('translation', 'translation.requireParameters', cfg.translation.requireParameters);
      setSettingsValue('deploy', 'deploy.default', cfg.deploy.default);
      installStyledSelects(q('#config-form'));
      installStyledSelects(q('#config-complete-form'));
      installColorControls(q('#config-form'));
      installColorControls(q('#config-complete-form'));

      const syncSearchFields = () => {
        const enabled = q('[data-settings-page="theme"] [name="search.enabled"]')?.checked !== false;
        for (const name of ['search.rememberRecent', 'search.placeholder', 'search.maxResults', 'search.shortcut']) {
          const control = q(`[data-settings-page="theme"] [name="${name}"]`);
          if (!control) continue;
          control.disabled = !enabled;
          control.mpressSelectSync?.();
          const field = control.closest('.mpress-form-field, .mpress-form-check');
          field?.classList.toggle('settings-field-disabled', !enabled);
          field?.querySelector('[data-shortcut-change]')?.toggleAttribute('disabled', !enabled);
        }
      };
      const searchEnabled = q('[data-settings-page="theme"] [name="search.enabled"]');
      searchEnabled?.addEventListener('change', syncSearchFields);
      syncSearchFields();

      // A floating hero is deliberately image-only. Do not present controls
      // that have no effect in that mode, or ask an author to infer which
      // settings apply to the selected layout.
      const syncBlogImageFields = () => {
        const form = q('[data-settings-page="blog"]');
        const mode = form?.querySelector('[name="blog.imageMode"]');
        if (!form || !mode) return;
        const floating = mode.value === 'floating';
        const setVisible = (name, visible) => {
          const field = form.querySelector(`[name="${name}"]`)?.closest('.mpress-form-field');
          if (field) field.hidden = !visible;
        };
        setVisible('blog.imageFit', !floating);
        setVisible('blog.imageBackground', !floating);
        setVisible('blog.imageWidth', floating);
      };
      const blogImageMode = q('[data-settings-page="blog"] [name="blog.imageMode"]');
      blogImageMode?.addEventListener('change', syncBlogImageFields);
      syncBlogImageFields();

      // Sidebar navigation swaps the visible configuration section in place.
      // Keep a stable field snapshot so an author cannot lose an edit by
      // opening another tool before they save.
      let initialSettingsState = '';
      const settingsState = () => JSON.stringify(Array.from(q('#config-complete-form')?.querySelectorAll('input, textarea, select') || [])
        .filter(control => control.type !== 'file' && !control.closest('#blog-post-editor'))
        .map((control, index) => [index, control.type, control.type === 'checkbox' || control.type === 'radio' ? control.checked : control.value]));
      const settingsHaveChanged = () => Boolean(initialSettingsState) && Boolean(q('#config-complete-form')?.isConnected) && settingsState() !== initialSettingsState;

      const updateWorkspacePageHeader = page => {
        if (!workspaceMode || !configPages[page]) return;
        const header = q('.workspace-page-header');
        header.querySelector('h1').textContent = configPages[page].title;
        let description = header.querySelector('p');
        if (!description) {
          description = document.createElement('p');
          header.append(description);
        }
        description.textContent = configPages[page].description;
      };
      const completeSettingsActions = q('#config-complete-form > .settings-actions');
      let activeBlogPanel = 'posts';
      q('[aria-label="Blog settings"]')?.remove();
      const blogTabs = Array.from(root.querySelectorAll('[data-blog-settings-tab]'));
      const showBlogPanel = selected => {
        activeBlogPanel = selected;
        blogTabs.forEach(tab => {
          const active = tab.dataset.blogSettingsTab === selected;
          tab.classList.toggle('active', active);
          tab.setAttribute('aria-selected', String(active));
          tab.tabIndex = active ? 0 : -1;
        });
        root.querySelectorAll('[data-blog-settings-panel]').forEach(panel => panel.hidden = panel.dataset.blogSettingsPanel !== selected);
        if (completeSettingsActions && !q('#settings-blog').hidden) completeSettingsActions.hidden = selected === 'posts';
        if (workspaceMode) {
          const details = selected === 'preferences'
            ? {title: 'Blog preferences', description: 'Set the defaults for the blog archive and article hero.'}
            : {title: 'Blog posts', description: 'Create and edit Markdown articles.'};
          q('.workspace-page-header h1').textContent = details.title;
          q('.workspace-page-header p').textContent = details.description;
          setWorkspaceActive(selected === 'preferences' ? 'blog-preferences' : 'blog-posts');
        }
      };
      blogTabs.forEach(tab => tab.addEventListener('click', () => showBlogPanel(tab.dataset.blogSettingsTab)));

      let blogPosts = [];
      const blogPostList = q('#blog-post-list');
      const blogPostEditor = q('#blog-post-editor');
      const blogPostToolbar = q('#blog-new-post')?.closest('.settings-section-toolbar');
      let blogEditorOpen = false;
      let blogEditorInitialState = '';
      const renderBlogPostList = () => {
        if (blogEditorOpen) return;
        blogPostEditor.hidden = true;
        blogPostToolbar.hidden = false;
        blogPostList.hidden = false;
        const today = new Date().toISOString().slice(0, 10);
        blogPostList.innerHTML = blogPosts.length ? blogPosts.map(post => {
          const status = post.draft ? 'Draft' : post.date > today ? 'Future-dated' : 'Published';
          const detail = post.date ? `${post.date} · ${status}` : status;
          return `<article class="content-list-row"><div><strong>${escapeHTML(post.title)}</strong><span>${escapeHTML(detail)}</span></div><div class="content-list-actions"><button class="button" type="button" data-open-blog-post="${escapeHTML(post.path)}" aria-label="Open ${escapeHTML(post.title)}" title="Open ${escapeHTML(post.title)}">Open</button></div></article>`;
        }).join('') : '<p class="empty-repeat">No blog posts yet. Create the first post when you are ready.</p>';
      };
      const loadBlogPosts = async () => {
        try {
          const result = await api('blog-posts');
          blogPosts = result.posts || [];
          renderBlogPostList();
        } catch (error) {
          blogPostList.innerHTML = `<p class="empty-repeat error-copy">${escapeHTML(error.message)}</p>`;
        }
      };
      const blogEditorValue = name => blogPostEditor.querySelector(`[data-blog-field="${name}"]`)?.value || '';
      const blogEditorState = () => JSON.stringify({
        title: blogEditorValue('title'), slug: blogEditorValue('slug'), status: blogEditorValue('status'),
        date: blogEditorValue('date'), description: blogEditorValue('description'), author: blogEditorValue('author'),
        tags: blogEditorValue('tags'), image: blogEditorValue('image'), body: blogEditorValue('body'),
        pendingTag: blogPostEditor.querySelector('[data-blog-tag-input]')?.value || ''
      });
      const closeBlogEditor = (discard = false) => {
        if (!discard && blogEditorOpen && blogEditorInitialState && blogEditorState() !== blogEditorInitialState && !window.confirm('Discard unsaved changes to this post?')) return false;
        blogEditorOpen = false;
        blogEditorInitialState = '';
        showBlogPanel('posts');
        renderBlogPostList();
        q('#drawer-body').scrollTop = 0;
        return true;
      };
      const openBlogEditor = async path => {
        const today = new Date().toISOString().slice(0, 10);
        const authorKey = `mpress-blog-last-author:${state.project?.repository?.path || state.project?.name || 'project'}`;
        let post = {path: '', title: '', description: '', slug: '', date: '', author: localStorage.getItem(authorKey) || '', tags: [], image: '', draft: true, body: ''};
        const installTagEditor = () => {
          const existingTagField = blogPostEditor.querySelector('[data-blog-field="tags"]')?.closest('.field');
        if (existingTagField) {
          const selectedTags = [];
          const knownTags = Array.from(new Set(blogPosts.flatMap(item => item.tags || []).map(tag => String(tag).trim()).filter(Boolean))).sort((left, right) => left.localeCompare(right));
          const tagField = document.createElement('div');
          tagField.className = 'field blog-tag-field';
          tagField.innerHTML = `<span>Tags</span><div class="blog-tag-editor" data-blog-tag-editor><div class="blog-tag-values" data-blog-tag-values></div><input data-blog-tag-input type="text" autocomplete="off" aria-label="Add a tag" placeholder="Add a tag"><div class="blog-tag-palette" data-blog-tag-palette role="listbox" aria-label="Tag suggestions" hidden></div></div><input data-blog-field="tags" type="hidden"><small>Press Enter or comma to add a tag.</small>`;
          existingTagField.replaceWith(tagField);
          const tagValueField = tagField.querySelector('[data-blog-field="tags"]');
          const tagInput = tagField.querySelector('[data-blog-tag-input]');
          const tagValues = tagField.querySelector('[data-blog-tag-values]');
          const tagPalette = tagField.querySelector('[data-blog-tag-palette]');
          const normaliseTag = value => String(value || '').trim().replace(/\s+/g, ' ');
          const hasTag = value => selectedTags.some(tag => tag.toLocaleLowerCase() === value.toLocaleLowerCase());
          const filteredTags = () => {
            const query = normaliseTag(tagInput.value).toLocaleLowerCase();
            return knownTags.filter(tag => !hasTag(tag) && (!query || tag.toLocaleLowerCase().includes(query))).slice(0, 8);
          };
          const renderTags = () => {
            tagValueField.value = selectedTags.join(', ');
            tagValues.innerHTML = selectedTags.map(tag => `<span class="blog-tag-chip"><span>${escapeHTML(tag)}</span><button type="button" data-remove-blog-tag="${escapeHTML(tag)}" aria-label="Remove ${escapeHTML(tag)}">${iconHTML('x')}</button></span>`).join('');
            const suggestions = filteredTags();
            tagPalette.innerHTML = suggestions.map(tag => `<button type="button" role="option" data-add-blog-tag="${escapeHTML(tag)}">${escapeHTML(tag)}</button>`).join('');
            tagPalette.hidden = suggestions.length === 0 || !tagInput.matches(':focus');
          };
          const addTag = value => {
            const tag = normaliseTag(value);
            if (!tag || hasTag(tag)) { tagInput.value = ''; renderTags(); return; }
            selectedTags.push(tag);
            tagInput.value = '';
            renderTags();
          };
          (post.tags || []).forEach(addTag);
          tagInput.addEventListener('focus', renderTags);
          tagInput.addEventListener('input', renderTags);
          tagInput.addEventListener('keydown', event => {
            if (event.key === 'Enter' || event.key === ',') {
              event.preventDefault();
              const firstSuggestion = filteredTags()[0];
              addTag(normaliseTag(tagInput.value) || firstSuggestion || '');
            } else if (event.key === 'Backspace' && !tagInput.value && selectedTags.length) {
              selectedTags.pop();
              renderTags();
            } else if (event.key === 'ArrowDown') {
              const firstOption = tagPalette.querySelector('[data-add-blog-tag]');
              if (firstOption) { event.preventDefault(); firstOption.focus(); }
            } else if (event.key === 'Escape') {
              tagPalette.hidden = true;
            }
          });
          tagField.addEventListener('click', event => {
            const remove = event.target.closest('[data-remove-blog-tag]');
            if (remove) {
              const index = selectedTags.findIndex(tag => tag === remove.dataset.removeBlogTag);
              if (index >= 0) selectedTags.splice(index, 1);
              renderTags();
              tagInput.focus();
              return;
            }
            const add = event.target.closest('[data-add-blog-tag]');
            if (add) { addTag(add.dataset.addBlogTag); tagInput.focus(); }
          });
          tagField.addEventListener('focusout', () => setTimeout(() => {
            if (!tagField.contains(root.activeElement)) tagPalette.hidden = true;
          }, 0));
          renderTags();
          }
        };
        if (path) {
          try { post = await api('blog-posts?path=' + encodeURIComponent(path)); }
          catch (error) { toast(error.message, true); return; }
        }
        blogEditorOpen = true;
        blogPostToolbar.hidden = true;
        blogPostList.hidden = true;
        blogPostEditor.hidden = false;
        updateWorkspacePageHeader('blog');
        if (workspaceMode) {
          q('.workspace-page-header h1').textContent = path ? post.title : 'New blog post';
          q('.workspace-page-header p').textContent = path ? 'Write visually or edit the Markdown source.' : 'Create a Markdown article and choose when to publish it.';
          setWorkspaceActive(path ? 'blog-posts' : 'blog-new');
        }
        const postStatus = post.draft ? 'draft' : 'published';
        blogPostEditor.innerHTML = `<div class="blog-editor-header"><button class="text-button" type="button" data-blog-editor-close>${iconHTML('arrow-left')}<span>Back to posts</span></button></div><div class="blog-editor-grid"><label class="field settings-wide"><span>Title</span><input data-blog-field="title" value="${escapeHTML(post.title)}" autocomplete="off"></label><label class="field"><span>Post URL</span><input data-blog-field="slug" value="${escapeHTML(post.slug)}" readonly placeholder="generated-from-title"><small>Generated from the title. Existing URLs stay unchanged.</small></label><label class="field"><span>Status</span><select data-blog-field="status"><option value="draft"${selectedAttr(postStatus, 'draft')}>Draft</option><option value="published"${selectedAttr(postStatus, 'published')}>Published</option></select><small data-blog-status-help></small></label><label class="field" data-blog-date-field><span>Publication date</span><input data-blog-field="date" type="date" value="${escapeHTML(post.date)}"></label><label class="field settings-wide"><span>Description</span><textarea data-blog-field="description" rows="3" placeholder="Summarise the post in one or two sentences.">${escapeHTML(post.description)}</textarea></label><label class="field"><span>Author</span><input data-blog-field="author" value="${escapeHTML(post.author)}" autocomplete="name"><small>M-Press remembers the last author on this device.</small></label><label class="field"><span>Tags</span><input data-blog-field="tags" value="${escapeHTML((post.tags || []).join(', '))}" placeholder="release, desktop"></label><div class="field settings-wide"><span>Cover image</span><div class="image-field-row"><input data-blog-field="image" value="${escapeHTML(post.image)}" readonly placeholder="No cover image"><button class="button" type="button" data-blog-cover-edit>Choose or edit image</button><input type="file" data-blog-cover-file accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif" hidden></div><small>Crop, position, brighten, darken, or add text before you save.</small></div></div><div class="markdown-editor"><div class="settings-subnav" role="tablist" aria-label="Post body"><button class="active" type="button" role="tab" aria-selected="true" data-blog-editor-tab="write">Write</button><button type="button" role="tab" aria-selected="false" data-blog-editor-tab="preview">Preview</button></div><div data-blog-editor-panel="write"><label class="field"><span class="visually-hidden">Markdown body</span><textarea class="markdown-source" data-blog-field="body" spellcheck="true" placeholder="Write the post in Markdown.">${escapeHTML(post.body)}</textarea></label></div><div class="markdown-preview" data-blog-editor-panel="preview" hidden><p class="empty-repeat">Choose Preview to render the Markdown.</p></div></div><div class="settings-actions blog-editor-actions"><div><button class="button" type="button" data-blog-editor-close>Cancel</button><button class="button primary" type="button" id="blog-post-save">${path ? 'Save post' : 'Create post'}</button></div></div>`;
        installTagEditor();
        const metadataGrid = blogPostEditor.querySelector('.blog-editor-grid');
        const markdownEditor = blogPostEditor.querySelector('.markdown-editor');
        const originalMarkdownTabs = markdownEditor.querySelector('.settings-subnav');
        const markdownPanel = markdownEditor.querySelector('[data-blog-editor-panel="write"]');
        const writePanel = markdownEditor.querySelector('[data-blog-editor-panel="preview"]');
        const markdownField = markdownPanel.querySelector('.field');
        const bodyField = markdownField.querySelector('[data-blog-field="body"]');
        const editorTabs = document.createElement('nav');
        editorTabs.className = 'settings-subnav blog-editor-tabs';
        editorTabs.setAttribute('role', 'tablist');
        editorTabs.setAttribute('aria-label', 'Edit blog post');
        editorTabs.innerHTML = `<button id="blog-editor-tab-markdown" class="active" type="button" role="tab" aria-selected="true" aria-controls="blog-editor-markdown" data-blog-editor-tab="markdown">${iconHTML('code-xml')}<span>Markdown</span></button><button id="blog-editor-tab-write" type="button" role="tab" aria-selected="false" aria-controls="blog-editor-write" data-blog-editor-tab="write">${iconHTML('file-text')}<span>Write</span></button><button id="blog-editor-tab-details" type="button" role="tab" aria-selected="false" aria-controls="blog-editor-details" data-blog-editor-tab="details">${iconHTML('settings')}<span>Details</span></button>`;
        originalMarkdownTabs.remove();
        markdownPanel.id = 'blog-editor-markdown';
        markdownPanel.dataset.blogEditorPanel = 'markdown';
        markdownPanel.classList.add('blog-editor-pane', 'blog-editor-writing');
        markdownPanel.setAttribute('role', 'tabpanel');
        markdownPanel.setAttribute('aria-labelledby', 'blog-editor-tab-markdown');
        writePanel.id = 'blog-editor-write';
        writePanel.dataset.blogEditorPanel = 'write';
        writePanel.classList.remove('markdown-preview');
        writePanel.classList.add('blog-editor-pane', 'blog-editor-write');
        writePanel.setAttribute('role', 'tabpanel');
        writePanel.setAttribute('aria-labelledby', 'blog-editor-tab-write');
        const markdownToolbar = document.createElement('div');
        markdownToolbar.className = 'markdown-toolbar';
        markdownToolbar.dataset.markdownToolbar = '';
        markdownToolbar.setAttribute('aria-label', 'Markdown formatting');
        markdownToolbar.innerHTML = `<div><strong>Markdown</strong><span>Write the article source. Drop or paste images anywhere.</span></div><div class="markdown-toolbar-actions"><button type="button" data-markdown-format="bold" aria-label="Bold" title="Bold">${iconHTML('bold')}</button><button type="button" data-markdown-format="italic" aria-label="Italic" title="Italic">${iconHTML('italic')}</button><button type="button" data-markdown-format="link" aria-label="Insert link" title="Insert link">${iconHTML('link')}</button><button type="button" data-markdown-format="code" aria-label="Insert code block" title="Insert code block">${iconHTML('code')}</button><button type="button" data-markdown-format="image" aria-label="Add an image or a light and dark pair" title="Add image">${iconHTML('image')}</button></div>`;
        const markdownFooter = document.createElement('div');
        markdownFooter.className = 'markdown-writing-footer';
        markdownFooter.innerHTML = `<span data-markdown-count></span><span>Use Tab to indent. Press ${/(Mac|iPhone|iPad|iPod)/i.test(navigator.userAgentData?.platform || navigator.platform || '') ? 'Command' : 'Ctrl'}+S to save.</span>`;
        markdownField.classList.add('markdown-writing-field');
        markdownField.insertBefore(markdownToolbar, bodyField);
        markdownField.append(markdownFooter);
        const richTextEditor = document.createElement('div');
        richTextEditor.className = 'rich-text-editor';
        const richTextToolbar = document.createElement('div');
        richTextToolbar.className = 'markdown-toolbar rich-text-toolbar';
        richTextToolbar.setAttribute('role', 'toolbar');
        richTextToolbar.setAttribute('aria-label', 'Rich text formatting');
        richTextToolbar.innerHTML = `<label class="rich-text-style"><span class="visually-hidden">Text style</span><select data-rich-block-format aria-label="Text style"><option value="p">Paragraph</option><option value="h2">Heading 2</option><option value="h3">Heading 3</option><option value="blockquote">Quote</option></select></label><div class="markdown-toolbar-actions"><button type="button" data-rich-format="bold" aria-label="Bold" title="Bold">${iconHTML('bold')}</button><button type="button" data-rich-format="italic" aria-label="Italic" title="Italic">${iconHTML('italic')}</button><button type="button" data-rich-format="link" aria-label="Insert link" title="Insert link">${iconHTML('link')}</button><button type="button" data-rich-format="unordered-list" aria-label="Bulleted list" title="Bulleted list">${iconHTML('list')}</button><button type="button" data-rich-format="ordered-list" aria-label="Numbered list" title="Numbered list">${iconHTML('list-ordered')}</button><button type="button" data-rich-format="quote" aria-label="Quote" title="Quote">${iconHTML('quote')}</button><button type="button" data-rich-format="code" aria-label="Code block" title="Code block">${iconHTML('code')}</button><button type="button" data-rich-format="image" aria-label="Add an image or a light and dark pair" title="Add image">${iconHTML('image')}</button></div>`;
        const richTextSurface = document.createElement('article');
        richTextSurface.className = 'markdown-preview rich-text-surface';
        richTextSurface.dataset.richTextSurface = '';
        richTextSurface.contentEditable = 'true';
        richTextSurface.spellcheck = true;
        richTextSurface.setAttribute('role', 'textbox');
        richTextSurface.setAttribute('aria-multiline', 'true');
        richTextSurface.setAttribute('aria-label', 'Blog post body');
        const richTextFooter = document.createElement('div');
        richTextFooter.className = 'markdown-writing-footer rich-text-footer';
        richTextFooter.innerHTML = '<span>Formatting is saved as Markdown. Drop or paste images anywhere.</span><span>Use the Markdown tab for components and advanced syntax.</span>';
        richTextEditor.append(richTextToolbar, richTextSurface, richTextFooter);
        writePanel.replaceChildren(richTextEditor);
        const detailsPanel = document.createElement('section');
        detailsPanel.id = 'blog-editor-details';
        detailsPanel.className = 'blog-editor-pane blog-editor-details';
        detailsPanel.dataset.blogEditorPanel = 'details';
        detailsPanel.setAttribute('role', 'tabpanel');
        detailsPanel.setAttribute('aria-labelledby', 'blog-editor-tab-details');
        detailsPanel.hidden = true;
        detailsPanel.innerHTML = `<header class="blog-editor-pane-heading"><span>Post details</span><h2>Publishing and presentation</h2><p>Set the title, publication status, audience details, and cover image.</p></header>`;
        detailsPanel.append(metadataGrid);
        markdownEditor.replaceWith(editorTabs, markdownPanel, writePanel, detailsPanel);
        if (path) {
          const editorHeader = blogPostEditor.querySelector('.blog-editor-header');
          const editPanel = document.createElement('div');
          editPanel.dataset.blogPostModePanel = 'edit';
          Array.from(blogPostEditor.children).filter(child => child !== editorHeader).forEach(child => editPanel.append(child));
          const modeTabs = document.createElement('div');
          modeTabs.className = 'settings-subnav blog-post-mode-tabs';
          modeTabs.setAttribute('role', 'tablist');
          modeTabs.setAttribute('aria-label', 'Blog post view');
          modeTabs.innerHTML = `<button type="button" role="tab" data-blog-post-mode="view">View</button><button type="button" role="tab" data-blog-post-mode="edit">Edit</button>`;
          const viewPanel = document.createElement('div');
          viewPanel.className = 'blog-post-preview';
          viewPanel.dataset.blogPostModePanel = 'view';
          viewPanel.innerHTML = `<iframe src="${escapeHTML(post.route)}" title="Preview of ${escapeHTML(post.title)}"></iframe>`;
          editorHeader.after(modeTabs, viewPanel, editPanel);
          const previewFrame = viewPanel.querySelector('iframe');
          previewFrame.addEventListener('load', () => {
            try {
              const previewDocument = previewFrame.contentDocument;
              const article = previewDocument?.querySelector('.blog-article-main');
              if (!previewDocument?.body || !article) return;
              previewDocument.body.replaceChildren(article);
              previewDocument.body.className = 'blog-article-page blog-post-preview-document';
              const previewStyle = previewDocument.createElement('style');
              previewStyle.textContent = '.blog-post-preview-document{min-height:100%;margin:0!important;padding:0!important;overflow-x:hidden}.blog-post-preview-document .blog-article-main{width:100%!important;max-width:none!important;margin:0!important;padding:clamp(2rem,5vw,4.5rem)!important}.blog-post-preview-document .blog-article{max-width:760px!important;margin:0 auto!important}';
              previewDocument.head.append(previewStyle);
              previewFrame.dataset.ready = 'true';
            } catch {}
          });
          const showPostMode = selected => {
            modeTabs.querySelectorAll('[data-blog-post-mode]').forEach(tab => {
              const active = tab.dataset.blogPostMode === selected;
              tab.classList.toggle('active', active);
              tab.setAttribute('aria-selected', String(active));
              tab.tabIndex = active ? 0 : -1;
            });
            blogPostEditor.querySelectorAll('[data-blog-post-mode-panel]').forEach(panel => panel.hidden = panel.dataset.blogPostModePanel !== selected);
          };
          modeTabs.addEventListener('click', event => {
            const selected = event.target.closest('[data-blog-post-mode]')?.dataset.blogPostMode;
            if (selected) showPostMode(selected);
          });
          showPostMode('view');
        }
        q('#drawer-body').scrollTop = 0;
        const titleField = blogPostEditor.querySelector('[data-blog-field="title"]');
        const slugField = blogPostEditor.querySelector('[data-blog-field="slug"]');
        titleField.addEventListener('input', () => {
          if (!path) slugField.value = titleField.value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
        });
        const statusField = blogPostEditor.querySelector('[data-blog-field="status"]');
        const dateField = blogPostEditor.querySelector('[data-blog-field="date"]');
        const dateContainer = blogPostEditor.querySelector('[data-blog-date-field]');
        const statusHelp = blogPostEditor.querySelector('[data-blog-status-help]');
        const syncPostStatus = () => {
          const status = statusField.value;
          dateContainer.hidden = status === 'draft';
          if (status === 'published') {
            if (!dateField.value || dateField.value > today) dateField.value = today;
            dateField.max = today;
            dateField.removeAttribute('min');
            statusHelp.textContent = 'The post is included in the next production build.';
          } else {
            dateField.removeAttribute('min');
            dateField.removeAttribute('max');
            statusHelp.textContent = 'Drafts appear only in development builds.';
          }
        };
        statusField.addEventListener('change', syncPostStatus);
        installStyledSelects(blogPostEditor);
        syncPostStatus();
        const markdownCount = blogPostEditor.querySelector('[data-markdown-count]');
        const updateMarkdownCount = () => {
          const text = bodyField.value.trim();
          const words = text ? text.split(/\s+/).length : 0;
          markdownCount.textContent = `${words} ${words === 1 ? 'word' : 'words'} · ${bodyField.value.length} characters`;
        };
        const escapeMarkdownText = value => String(value || '').replaceAll('\u00a0', ' ').replace(/([\\`*_[\]])/g, '\\$1');
        const richInlineMarkdown = node => {
          if (node.nodeType === 3) return escapeMarkdownText(node.nodeValue);
          if (node.nodeType !== 1) return '';
          const tag = node.tagName.toLowerCase();
          if (tag === 'br') return '  \n';
          if (node.classList.contains('mpress-theme-image')) {
            const light = node.querySelector('.mpress-theme-image-light');
            const dark = node.querySelector('.mpress-theme-image-dark');
            const alt = light?.getAttribute('alt') || dark?.getAttribute('alt') || '';
            if (light && dark) return `@image{light="${contentImageAttribute(light.getAttribute('src') || '')}" dark="${contentImageAttribute(dark.getAttribute('src') || '')}" alt="${contentImageAttribute(alt)}"}`;
          }
          if (tag === 'img') return `![${escapeMarkdownText(node.getAttribute('alt') || '')}](${node.getAttribute('src') || ''})`;
          if (tag === 'code') {
            const value = node.textContent || '';
            const marker = value.includes('`') ? '``' : '`';
            return `${marker}${value}${marker}`;
          }
          const content = Array.from(node.childNodes).map(richInlineMarkdown).join('');
          if (tag === 'strong' || tag === 'b') return `**${content}**`;
          if (tag === 'em' || tag === 'i') return `*${content}*`;
          if (tag === 's' || tag === 'del') return `~~${content}~~`;
          if (tag === 'a') {
            const href = node.getAttribute('href') || '';
            const title = node.getAttribute('title');
            return `[${content || escapeMarkdownText(href)}](${href}${title ? ` "${title.replaceAll('"', '\\"')}"` : ''})`;
          }
          return content;
        };
        const richCodeMarkdown = node => {
          const pre = node.matches('pre') ? node : node.querySelector('pre');
          if (!pre) return '';
          const codeNode = pre.querySelector('code');
          const languageClass = Array.from(codeNode?.classList || []).find(name => name.startsWith('language-')) || '';
          const language = node.classList.contains('mpress-terminal') ? 'sh' : languageClass.slice('language-'.length);
          const title = node.querySelector('.mpress-codeframe-title')?.textContent?.trim() || '';
          const info = `${language}${title ? ` title="${title.replaceAll('"', '\\"')}"` : ''}`.trim();
          return `\`\`\`${info}\n${(pre.textContent || '').replace(/\n+$/, '')}\n\`\`\`\n\n`;
        };
        const richListMarkdown = (list, depth = 0) => {
          const ordered = list.tagName.toLowerCase() === 'ol';
          const start = Number.parseInt(list.getAttribute('start') || '1', 10) || 1;
          return Array.from(list.children).filter(item => item.tagName?.toLowerCase() === 'li').map((item, index) => {
            const nested = Array.from(item.children).filter(child => ['ul', 'ol'].includes(child.tagName.toLowerCase()));
            const content = Array.from(item.childNodes).filter(child => !(child.nodeType === 1 && ['ul', 'ol'].includes(child.tagName.toLowerCase()))).map(richInlineMarkdown).join('').trim();
            const indent = '  '.repeat(depth);
            const prefix = ordered ? `${start + index}. ` : '- ';
            const continuation = `${indent}  `;
            const line = `${indent}${prefix}${content.replaceAll('\n', `\n${continuation}`)}`;
            const children = nested.map(child => richListMarkdown(child, depth + 1).trimEnd()).filter(Boolean).join('\n');
            return children ? `${line}\n${children}` : line;
          }).join('\n') + '\n\n';
        };
        const richTableMarkdown = table => {
          const rows = Array.from(table.querySelectorAll('tr')).map(row => Array.from(row.querySelectorAll(':scope > th, :scope > td')).map(cell => richInlineMarkdown(cell).trim().replaceAll('|', '\\|'))).filter(row => row.length);
          if (!rows.length) return '';
          const width = Math.max(...rows.map(row => row.length));
          rows.forEach(row => { while (row.length < width) row.push(''); });
          return `| ${rows[0].join(' | ')} |\n| ${rows[0].map(() => '---').join(' | ')} |\n${rows.slice(1).map(row => `| ${row.join(' | ')} |`).join('\n')}\n\n`;
        };
        const richBlockMarkdown = node => {
          if (node.nodeType === 3) return escapeMarkdownText(node.nodeValue);
          if (node.nodeType !== 1) return '';
          const tag = node.tagName.toLowerCase();
          if (node.classList.contains('mpress-terminal') || node.classList.contains('mpress-codeframe')) return richCodeMarkdown(node);
          if (/^h[1-6]$/.test(tag)) return `${'#'.repeat(Number(tag[1]))} ${Array.from(node.childNodes).map(richInlineMarkdown).join('').trim()}\n\n`;
          if (tag === 'p') return `${Array.from(node.childNodes).map(richInlineMarkdown).join('').trim()}\n\n`;
          if (tag === 'ul' || tag === 'ol') return richListMarkdown(node);
          if (tag === 'blockquote') {
            const content = Array.from(node.childNodes).map(richBlockMarkdown).join('').trim();
            return content ? `${content.split('\n').map(line => `> ${line}`.trimEnd()).join('\n')}\n\n` : '';
          }
          if (tag === 'pre') return richCodeMarkdown(node);
          if (tag === 'hr') return '---\n\n';
          if (tag === 'table') return richTableMarkdown(node);
          if (['div', 'section', 'article', 'main', 'aside'].includes(tag)) return Array.from(node.childNodes).map(richBlockMarkdown).join('');
          return `${richInlineMarkdown(node)}\n\n`;
        };
        const richTextToMarkdown = () => Array.from(richTextSurface.childNodes).map(richBlockMarkdown).join('').replace(/\n{3,}/g, '\n\n').trim();
        const syncRichTextToMarkdown = () => {
          bodyField.value = richTextToMarkdown();
          bodyField.dispatchEvent(new Event('input', {bubbles: true}));
        };
        let richTextSelection = null;
        const rememberRichTextSelection = () => {
          const selection = document.getSelection();
          if (selection?.rangeCount && richTextSurface.contains(selection.anchorNode)) richTextSelection = selection.getRangeAt(0).cloneRange();
        };
        const restoreRichTextSelection = () => {
          if (!richTextSelection) return;
          const selection = document.getSelection();
          selection.removeAllRanges();
          selection.addRange(richTextSelection);
        };
        const contentImageTypes = 'image/svg+xml,image/png,image/jpeg,image/webp,image/avif';
        const contentImageFiles = list => Array.from(list || []).filter(file => file?.type?.startsWith('image/'));
        const contentImagePath = path => `/${String(path || '').replace(/^\/+/, '')}`;
        const contentImageAttribute = value => String(value || '').replaceAll('\\', '\\\\').replaceAll('"', '\\"');
        const classifyThemeImages = files => {
          const selected = contentImageFiles(files).slice(0, 2);
          const detectedLight = selected.find(file => /(?:^|[-_.\s])(light|day)(?:[-_.\s]|$)/i.test(file.name));
          const detectedDark = selected.find(file => /(?:^|[-_.\s])(dark|night)(?:[-_.\s]|$)/i.test(file.name));
          const light = detectedLight || selected.find(file => file !== detectedDark) || null;
          const dark = detectedDark || selected.find(file => file !== light) || null;
          return {
            light,
            dark
          };
        };
        const insertContentImage = ({target, selection, description, source = '', light = '', dark = ''}) => {
          const alt = String(description || '').trim();
          const markdown = light && dark
            ? `@image{light="${contentImageAttribute(contentImagePath(light))}" dark="${contentImageAttribute(contentImagePath(dark))}" alt="${contentImageAttribute(alt)}"}`
            : `![${escapeMarkdownText(alt)}](${contentImagePath(source)})`;
          if (target === 'rich') {
            restoreRichTextSelection();
            richTextSurface.focus();
            const html = light && dark
              ? `<p><span class="mpress-theme-image"><img class="mpress-theme-image-light" src="${escapeHTML(contentImagePath(light))}" alt="${escapeHTML(alt)}" draggable="false"><img class="mpress-theme-image-dark" src="${escapeHTML(contentImagePath(dark))}" alt="${escapeHTML(alt)}" draggable="false"></span></p><p><br></p>`
              : `<p><img src="${escapeHTML(contentImagePath(source))}" alt="${escapeHTML(alt)}" draggable="false"></p><p><br></p>`;
            document.execCommand('insertHTML', false, html);
            syncRichTextToMarkdown();
            rememberRichTextSelection();
            return;
          }
          const start = selection?.start ?? bodyField.selectionStart;
          const end = selection?.end ?? bodyField.selectionEnd;
          const before = bodyField.value.slice(0, start);
          const after = bodyField.value.slice(end);
          const prefix = before && !before.endsWith('\n\n') ? (before.endsWith('\n') ? '\n' : '\n\n') : '';
          const suffix = after && !after.startsWith('\n\n') ? (after.startsWith('\n') ? '\n' : '\n\n') : '';
          bodyField.setRangeText(`${prefix}${markdown}${suffix}`, start, end, 'end');
          bodyField.dispatchEvent(new Event('input', {bubbles: true}));
          bodyField.focus();
        };
        const editContentImages = ({mode, single, light, dark, description, target, selection}) => {
          const edit = (file, title, onSave) => {
            void openImageEditor({
              file,
              category: 'content',
              suggestedName: imageAssetName(file.name),
              title,
              onSave
            }).catch(error => toast(error.message, true));
          };
          if (mode === 'single') {
            edit(single, 'Edit article image', savedPath => insertContentImage({target, selection, description, source: savedPath}));
            return;
          }
          edit(light, 'Edit light-mode image', lightPath => {
            edit(dark, 'Edit dark-mode image', darkPath => insertContentImage({target, selection, description, light: lightPath, dark: darkPath}));
          });
        };
        const openContentImageDialog = ({files = [], target = 'markdown'} = {}) => {
          const initial = contentImageFiles(files).slice(0, 2);
          if (contentImageFiles(files).length > 2) {
            toast('Choose one image, or one light and one dark image', true);
            return;
          }
          const pair = classifyThemeImages(initial);
          const slots = {single: initial[0] || null, light: pair.light, dark: initial.length > 1 ? pair.dark : null};
          let mode = initial.length > 1 ? 'theme' : 'single';
          const insertionSelection = target === 'markdown' ? {start: bodyField.selectionStart, end: bodyField.selectionEnd} : null;
          const selectedText = target === 'markdown' ? bodyField.value.slice(insertionSelection.start, insertionSelection.end).trim() : document.getSelection()?.toString().trim() || '';
          const overlay = document.createElement('div');
          overlay.className = 'image-editor-overlay content-image-overlay';
          overlay.innerHTML = `<div class="content-image-dialog" role="dialog" aria-modal="true" aria-labelledby="content-image-title"><div class="image-editor-top"><div><span class="drawer-eyebrow">Article image</span><h2 id="content-image-title">Add an image</h2></div><button class="icon-button" type="button" data-content-image-close aria-label="Close image chooser">${iconHTML('x')}</button></div><div class="content-image-body"><p class="content-image-intro">Drop one image for every colour scheme, or add a light and dark pair.</p><div class="settings-subnav content-image-tabs" role="tablist" aria-label="Image colour schemes"><button type="button" role="tab" data-content-image-mode="single">One image</button><button type="button" role="tab" data-content-image-mode="theme">Light and dark</button></div><div class="content-image-panel" data-content-image-panel="single"><button class="content-image-dropzone" type="button" data-content-image-slot="single"><span class="content-image-drop-icon">${iconHTML('image')}</span><span><strong>Drop an image here</strong><small>or choose PNG, JPEG, WebP, AVIF, or SVG</small></span><img alt="" hidden></button><input type="file" data-content-image-input="single" accept="${contentImageTypes}" hidden></div><div class="content-image-panel content-image-pair" data-content-image-panel="theme"><div><span>Light mode</span><button class="content-image-dropzone compact" type="button" data-content-image-slot="light"><span class="content-image-drop-icon">${iconHTML('sun')}</span><span><strong>Choose the light image</strong><small>No file selected</small></span><img alt="" hidden></button><input type="file" data-content-image-input="light" accept="${contentImageTypes}" hidden></div><div><span>Dark mode</span><button class="content-image-dropzone compact" type="button" data-content-image-slot="dark"><span class="content-image-drop-icon">${iconHTML('moon')}</span><span><strong>Choose the dark image</strong><small>No file selected</small></span><img alt="" hidden></button><input type="file" data-content-image-input="dark" accept="${contentImageTypes}" hidden></div></div><label class="field content-image-description"><span>Image description</span><input type="text" data-content-image-description value="${escapeHTML(selectedText)}" placeholder="Describe what the image shows"><small>Write a useful description for people who cannot see the image.</small></label><label class="content-image-decorative"><input type="checkbox" data-content-image-decorative><span>This image is decorative</span></label><p class="content-image-error" data-content-image-error role="alert" hidden></p></div><div class="image-editor-actions"><button class="button" type="button" data-content-image-close>Cancel</button><button class="button primary" type="button" data-content-image-continue>Continue to edit</button></div></div>`;
          root.append(overlay);
          lockPageScroll();
          const previewURLs = new Map();
          const handleKeydown = event => { if (event.key === 'Escape') close(); };
          const close = () => {
            previewURLs.forEach(url => URL.revokeObjectURL(url));
            overlay.removeEventListener('keydown', handleKeydown);
            overlay.remove();
            unlockPageScroll();
          };
          overlay.addEventListener('keydown', handleKeydown);
          const renderSlot = name => {
            const button = overlay.querySelector(`[data-content-image-slot="${name}"]`);
            const file = slots[name];
            const image = button.querySelector('img');
            const strong = button.querySelector('strong');
            const small = button.querySelector('small');
            if (previewURLs.has(name)) URL.revokeObjectURL(previewURLs.get(name));
            if (!file) {
              image.hidden = true;
              image.removeAttribute('src');
              strong.textContent = name === 'single' ? 'Drop an image here' : `Choose the ${name} image`;
              small.textContent = name === 'single' ? 'or choose PNG, JPEG, WebP, AVIF, or SVG' : 'No file selected';
              button.classList.remove('has-file');
              return;
            }
            const url = URL.createObjectURL(file);
            previewURLs.set(name, url);
            image.src = url;
            image.hidden = false;
            strong.textContent = file.name;
            small.textContent = `${Math.max(1, Math.round(file.size / 1024))} KB`;
            button.classList.add('has-file');
          };
          const setMode = next => {
            mode = next;
            overlay.querySelectorAll('[data-content-image-mode]').forEach(tab => {
              const active = tab.dataset.contentImageMode === mode;
              tab.classList.toggle('active', active);
              tab.setAttribute('aria-selected', String(active));
              tab.tabIndex = active ? 0 : -1;
            });
            overlay.querySelectorAll('[data-content-image-panel]').forEach(panel => panel.hidden = panel.dataset.contentImagePanel !== mode);
          };
          const setSlot = (name, file) => {
            if (!file?.type?.startsWith('image/')) { toast('Choose an image file', true); return; }
            slots[name] = file;
            renderSlot(name);
          };
          ['single', 'light', 'dark'].forEach(name => {
            const button = overlay.querySelector(`[data-content-image-slot="${name}"]`);
            const input = overlay.querySelector(`[data-content-image-input="${name}"]`);
            button.addEventListener('click', () => input.click());
            input.addEventListener('change', () => { if (input.files?.[0]) setSlot(name, input.files[0]); input.value = ''; });
            ['dragenter', 'dragover'].forEach(type => button.addEventListener(type, event => {
              if (!Array.from(event.dataTransfer?.types || []).includes('Files')) return;
              event.preventDefault();
              button.classList.add('is-dragging');
            }));
            button.addEventListener('dragleave', () => button.classList.remove('is-dragging'));
            button.addEventListener('drop', event => {
              event.preventDefault();
              button.classList.remove('is-dragging');
              const dropped = contentImageFiles(event.dataTransfer?.files);
              if (dropped.length > 2) { toast('Choose one image, or one light and one dark image', true); return; }
              if (name === 'single' && dropped.length > 1) {
                const droppedPair = classifyThemeImages(dropped);
                slots.light = droppedPair.light;
                slots.dark = droppedPair.dark;
                renderSlot('light');
                renderSlot('dark');
                setMode('theme');
                return;
              }
              if (dropped[0]) setSlot(name, dropped[0]);
            });
            renderSlot(name);
          });
          overlay.querySelectorAll('[data-content-image-mode]').forEach(tab => tab.addEventListener('click', () => setMode(tab.dataset.contentImageMode)));
          overlay.querySelectorAll('[data-content-image-close]').forEach(button => button.addEventListener('click', close));
          overlay.addEventListener('click', event => { if (event.target === overlay) close(); });
          const description = overlay.querySelector('[data-content-image-description]');
          const decorative = overlay.querySelector('[data-content-image-decorative]');
          const error = overlay.querySelector('[data-content-image-error]');
          decorative.addEventListener('change', () => { description.disabled = decorative.checked; if (decorative.checked) error.hidden = true; });
          overlay.querySelector('[data-content-image-continue]').addEventListener('click', () => {
            const missingFile = mode === 'single' ? !slots.single : !slots.light || !slots.dark;
            if (missingFile) {
              error.textContent = mode === 'single' ? 'Choose an image.' : 'Choose both a light and a dark image.';
              error.hidden = false;
              return;
            }
            if (!decorative.checked && !description.value.trim()) {
              error.textContent = 'Add an image description, or mark the image as decorative.';
              error.hidden = false;
              description.focus();
              return;
            }
            const values = {mode, single: slots.single, light: slots.light, dark: slots.dark, description: decorative.checked ? '' : description.value.trim(), target, selection: insertionSelection};
            close();
            editContentImages(values);
          });
          setMode(mode);
          requestAnimationFrame(() => (initial.length ? overlay.querySelector('[data-content-image-description]') : overlay.querySelector(`[data-content-image-slot="${mode === 'single' ? 'single' : 'light'}"]`))?.focus());
        };
        const installContentImageDrop = (element, target) => {
          let dragDepth = 0;
          const hasFiles = event => Array.from(event.dataTransfer?.types || []).includes('Files');
          element.addEventListener('dragenter', event => {
            if (!hasFiles(event)) return;
            event.preventDefault();
            dragDepth += 1;
            element.classList.add('content-image-drop-active');
          });
          element.addEventListener('dragover', event => { if (hasFiles(event)) event.preventDefault(); });
          element.addEventListener('dragleave', () => {
            dragDepth = Math.max(0, dragDepth - 1);
            if (!dragDepth) element.classList.remove('content-image-drop-active');
          });
          element.addEventListener('drop', event => {
            if (!hasFiles(event)) return;
            event.preventDefault();
            dragDepth = 0;
            element.classList.remove('content-image-drop-active');
            const files = contentImageFiles(event.dataTransfer?.files);
            if (files.length) openContentImageDialog({files, target});
          });
          element.addEventListener('paste', event => {
            const files = contentImageFiles(event.clipboardData?.files);
            if (!files.length) return;
            event.preventDefault();
            openContentImageDialog({files, target});
          });
        };
        const copyRenderedCode = async button => {
          const container = button.closest('.mpress-terminal, .mpress-codeframe');
          const code = container?.querySelector('pre')?.dataset.commands || container?.querySelector('code')?.textContent || '';
          if (!code.trim()) return;
          try {
            await navigator.clipboard.writeText(code.trimEnd());
            const original = button.textContent;
            button.textContent = 'Copied';
            setTimeout(() => { button.textContent = original; }, 1200);
          } catch { toast('Could not copy the code', true); }
        };
        const renderRichTextEditor = async () => {
          richTextSurface.setAttribute('aria-busy', 'true');
          richTextSurface.innerHTML = '<p class="empty-repeat">Preparing the editor.</p>';
          try {
            const result = await api('markdown-preview', {method: 'POST', body: JSON.stringify({markdown: bodyField.value})});
            richTextSurface.innerHTML = result.html || '<p><br></p>';
            richTextSurface.querySelectorAll('button').forEach(button => button.contentEditable = 'false');
            richTextSurface.querySelectorAll('img').forEach(image => image.draggable = false);
          } catch (error) {
            richTextSurface.innerHTML = '<p class="empty-repeat error-copy" contenteditable="false">The visual editor is temporarily unavailable. Use the Markdown tab to continue.</p>';
          } finally {
            richTextSurface.removeAttribute('aria-busy');
          }
        };
        richTextSurface.addEventListener('input', () => {
          rememberRichTextSelection();
          syncRichTextToMarkdown();
        });
        richTextSurface.addEventListener('keyup', rememberRichTextSelection);
        richTextSurface.addEventListener('mouseup', rememberRichTextSelection);
        richTextSurface.addEventListener('click', event => {
          const link = event.target.closest('a');
          if (link) event.preventDefault();
          const copy = event.target.closest('.mpress-copy');
          if (copy) void copyRenderedCode(copy);
        });
        richTextToolbar.addEventListener('mousedown', event => {
          if (event.target.closest('[data-rich-format]')) event.preventDefault();
        });
        richTextToolbar.addEventListener('click', event => {
          const button = event.target.closest('[data-rich-format]');
          if (!button) return;
          restoreRichTextSelection();
          richTextSurface.focus();
          const format = button.dataset.richFormat;
          if (format === 'link') {
            const href = window.prompt('Link URL', 'https://');
            if (!href) return;
            const selection = document.getSelection();
            if (selection?.toString()) document.execCommand('createLink', false, href);
            else document.execCommand('insertHTML', false, `<a href="${escapeHTML(href)}">${escapeHTML(href)}</a>`);
          } else if (format === 'image') {
            openContentImageDialog({target: 'rich'});
            return;
          } else {
            const commands = {bold: 'bold', italic: 'italic', 'unordered-list': 'insertUnorderedList', 'ordered-list': 'insertOrderedList', quote: 'formatBlock', code: 'formatBlock'};
            document.execCommand(commands[format], false, format === 'quote' ? 'blockquote' : format === 'code' ? 'pre' : null);
          }
          syncRichTextToMarkdown();
          rememberRichTextSelection();
        });
        richTextToolbar.querySelector('[data-rich-block-format]').addEventListener('change', event => {
          restoreRichTextSelection();
          richTextSurface.focus();
          document.execCommand('formatBlock', false, event.currentTarget.value);
          syncRichTextToMarkdown();
          rememberRichTextSelection();
        });
        const insertMarkdown = format => {
          const start = bodyField.selectionStart;
          const end = bodyField.selectionEnd;
          const selected = bodyField.value.slice(start, end);
          const replacement = {
            bold: `**${selected || 'bold text'}**`,
            italic: `*${selected || 'italic text'}*`,
            link: `[${selected || 'link text'}](https://)`,
            code: `\`\`\`\n${selected || 'code'}\n\`\`\``
          }[format];
          if (!replacement) return;
          bodyField.setRangeText(replacement, start, end, 'end');
          bodyField.dispatchEvent(new Event('input', {bubbles: true}));
          bodyField.focus();
        };
        markdownToolbar.addEventListener('click', event => {
          const button = event.target.closest('[data-markdown-format]');
          if (!button) return;
          if (button.dataset.markdownFormat === 'image') openContentImageDialog({target: 'markdown'});
          else insertMarkdown(button.dataset.markdownFormat);
        });
        installContentImageDrop(markdownField, 'markdown');
        installContentImageDrop(richTextEditor, 'rich');
        bodyField.addEventListener('input', updateMarkdownCount);
        bodyField.addEventListener('keydown', event => {
          if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
            event.preventDefault();
            q('#blog-post-save')?.click();
            return;
          }
          if (event.key !== 'Tab') return;
          event.preventDefault();
          if (bodyField.selectionStart === bodyField.selectionEnd) {
            bodyField.setRangeText('  ', bodyField.selectionStart, bodyField.selectionEnd, 'end');
          } else {
            const lineStart = bodyField.value.lastIndexOf('\n', bodyField.selectionStart - 1) + 1;
            const lineEnd = bodyField.value.indexOf('\n', bodyField.selectionEnd);
            const end = lineEnd < 0 ? bodyField.value.length : lineEnd;
            const selected = bodyField.value.slice(lineStart, end);
            const replacement = event.shiftKey ? selected.replace(/^ {1,2}/gm, '') : selected.replace(/^/gm, '  ');
            bodyField.setRangeText(replacement, lineStart, end, 'preserve');
          }
          bodyField.dispatchEvent(new Event('input', {bubbles: true}));
        });
        updateMarkdownCount();
        const coverField = blogPostEditor.querySelector('[data-blog-field="image"]');
        const coverChooser = blogPostEditor.querySelector('[data-blog-cover-file]');
        const editCover = async file => {
          const source = file ? '' : brandPreviewURL(coverField.value);
          if (!file && !source) { coverChooser.click(); return; }
          try {
            await openImageEditor({file, source, category: 'blog', suggestedName: imageAssetName(slugField.value || titleField.value || file?.name || 'blog-cover'), title: 'Edit cover image', onSave: savedPath => {
              coverField.value = `/${savedPath}`;
              coverField.dispatchEvent(new Event('input', {bubbles: true}));
            }});
          } catch (error) { toast(error.message, true); }
        };
        blogPostEditor.querySelector('[data-blog-cover-edit]').addEventListener('click', () => void editCover());
        coverChooser.addEventListener('change', () => { const file = coverChooser.files?.[0]; if (file) void editCover(file); coverChooser.value = ''; });
        blogPostEditor.querySelectorAll('[data-blog-editor-close]').forEach(button => button.addEventListener('click', () => closeBlogEditor()));
        const showEditorPanel = async selected => {
          blogPostEditor.querySelectorAll('[data-blog-editor-tab]').forEach(tab => {
            const active = tab.dataset.blogEditorTab === selected;
            tab.classList.toggle('active', active);
            tab.setAttribute('aria-selected', String(active));
            tab.tabIndex = active ? 0 : -1;
          });
          blogPostEditor.querySelectorAll('[data-blog-editor-panel]').forEach(panel => panel.hidden = panel.dataset.blogEditorPanel !== selected);
          if (selected === 'preview') {
            const preview = blogPostEditor.querySelector('[data-blog-editor-panel="preview"]');
            preview.innerHTML = '<p class="empty-repeat">Rendering preview.</p>';
            try {
              const request = () => api('markdown-preview', {method: 'POST', body: JSON.stringify({markdown: blogEditorValue('body')})});
              let result;
              try {
                result = await request();
              } catch (firstError) {
                if (!(firstError instanceof TypeError)) throw firstError;
                await new Promise(resolve => setTimeout(resolve, 180));
                result = await request();
              }
              preview.innerHTML = result.html || '<p class="empty-repeat">The post body is empty.</p>';
              preview.querySelectorAll('.mpress-copy').forEach(button => button.addEventListener('click', async () => {
                const container = button.closest('.mpress-terminal, .mpress-codeframe');
                const code = container?.querySelector('pre')?.dataset.commands || container?.querySelector('code')?.textContent || '';
                if (!code.trim()) return;
                try {
                  await navigator.clipboard.writeText(code.trimEnd());
                  const defaultLabel = button.dataset.copyLabel || button.getAttribute('aria-label') || 'Copy';
                  const copiedLabel = button.dataset.copiedLabel || 'Copied';
                  const status = button.querySelector('[data-copy-status]');
                  clearTimeout(button._mpressCopyTimer);
                  button.classList.add('is-copied');
                  button.setAttribute('aria-label', copiedLabel);
                  button.setAttribute('title', copiedLabel);
                  if (status) status.textContent = copiedLabel;
                  button._mpressCopyTimer = setTimeout(() => {
                    button.classList.remove('is-copied');
                    button.setAttribute('aria-label', defaultLabel);
                    button.setAttribute('title', defaultLabel);
                    if (status) status.textContent = defaultLabel;
                  }, 3000);
                } catch { toast('Could not copy the code', true); }
              }));
            } catch (error) { preview.innerHTML = `<p class="empty-repeat error-copy">Preview is temporarily unavailable. Choose Preview to try again.</p>`; }
          }
          if (selected === 'write') await renderRichTextEditor();
        };
        const editorTabItems = Array.from(blogPostEditor.querySelectorAll('[data-blog-editor-tab]'));
        editorTabItems.forEach((tab, index) => {
          tab.addEventListener('click', () => void showEditorPanel(tab.dataset.blogEditorTab));
          tab.addEventListener('keydown', event => {
            const direction = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
            if (!direction && event.key !== 'Home' && event.key !== 'End') return;
            event.preventDefault();
            const destination = event.key === 'Home' ? 0 : event.key === 'End' ? editorTabItems.length - 1 : (index + direction + editorTabItems.length) % editorTabItems.length;
            const next = editorTabItems[destination];
            void showEditorPanel(next.dataset.blogEditorTab);
            next.focus();
          });
        });
        q('#blog-post-save').addEventListener('click', async event => {
          if (!writePanel.hidden) syncRichTextToMarkdown();
          const title = blogEditorValue('title').trim();
          const slug = blogEditorValue('slug').trim();
          if (!title) { titleField.focus(); toast('Enter a post title', true); return; }
          if (!path && !/^[a-z0-9][a-z0-9-]*$/.test(slug)) { slugField.focus(); toast('Use lowercase letters, numbers and hyphens in the post URL', true); return; }
          const button = event.currentTarget;
          button.disabled = true;
          button.textContent = path ? 'Saving' : 'Creating';
          const status = blogEditorValue('status');
          const publicationDate = status === 'draft' ? '' : blogEditorValue('date');
          const author = blogEditorValue('author').trim();
          const payload = {path: post.path, revision: post.revision, title, slug, description: blogEditorValue('description').trim(), date: publicationDate, author, tags: blogEditorValue('tags').split(',').map(tag => tag.trim()).filter(Boolean), image: blogEditorValue('image').trim(), draft: status === 'draft', body: blogEditorValue('body')};
          try {
            const result = await api('blog-posts', {method: path ? 'PUT' : 'POST', body: JSON.stringify(payload)});
            if (author) localStorage.setItem(authorKey, author);
            updateBuild(result.state);
            toast(path ? 'Blog post saved' : 'Blog post created');
            await loadBlogPosts();
            closeBlogEditor(true);
          } catch (error) { button.disabled = false; button.textContent = path ? 'Save post' : 'Create post'; toast(error.message, true); }
        });
        blogEditorInitialState = blogEditorState();
        if (!path) bodyField.focus();
      };
      q('#blog-new-post')?.addEventListener('click', () => void openBlogEditor(''));
      blogPostList?.addEventListener('click', event => {
        const open = event.target.closest('[data-open-blog-post]');
        if (open) void openBlogEditor(open.dataset.openBlogPost);
      });
      void loadBlogPosts();

      const versionList = q('#version-list');
      const versionStatus = q('#version-status');
      const versionCreate = q('#version-create-form');
      const renderVersions = result => {
        setSettingsChecked('versioning', 'versioning.enabled', result.enabled);
        const versions = result.versions || [];
        const hasCurrent = Boolean(result.current && versions.some(version => version.label === result.current));
        versionStatus.className = hasCurrent || !result.current ? 'feature-summary' : 'settings-notice';
        versionStatus.innerHTML = hasCurrent
          ? `<strong>Current release: ${escapeHTML(result.current)}</strong><span>This captured build is the default version shown to readers.</span>`
          : result.current
            ? `<strong>${escapeHTML(result.current)} has not been captured</strong><span>Create this version before you publish, or select another captured build as current.</span>`
            : '<strong>No current release</strong><span>Create a version. M-Press will make the first successful capture current.</span>';
        versionList.innerHTML = versions.length ? versions.map(version => `<article class="content-list-row"><div><strong>${escapeHTML(version.label)}${version.label === result.current ? '<span class="status-pill">Current</span>' : ''}</strong><span>${version.createdAt ? `Captured ${escapeHTML(new Date(version.createdAt).toLocaleString())}` : 'Captured version'}</span></div><div class="content-list-actions">${version.label === result.current ? '' : `<button class="button" type="button" data-version-current="${escapeHTML(version.label)}">Set current</button>`}<button class="button" type="button" data-version-verify="${escapeHTML(version.label)}">Verify</button><button class="button danger-quiet" type="button" data-version-remove="${escapeHTML(version.label)}">Remove</button></div></article>`).join('') : '<p class="empty-repeat">No versions have been captured.</p>';
      };
      const loadVersions = async () => {
        try { renderVersions(await api('versions')); }
        catch (error) { versionList.innerHTML = `<p class="empty-repeat error-copy">${escapeHTML(error.message)}</p>`; }
      };
      q('#version-new')?.addEventListener('click', () => {
        versionCreate.hidden = false;
        q('#version-new').hidden = true;
        q('#version-create-error').hidden = true;
        versionCreate.querySelector('[data-version-label]').focus();
      });
      q('#version-create-cancel')?.addEventListener('click', () => {
        versionCreate.hidden = true;
        q('#version-new').hidden = false;
      });
      q('#version-create-submit')?.addEventListener('click', async event => {
        const input = versionCreate.querySelector('[data-version-label]');
        const label = input.value.trim();
        if (!/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(label)) { input.focus(); toast('Enter a version label such as 3.0', true); return; }
        const button = event.currentTarget;
        const error = q('#version-create-error');
        error.hidden = true;
        button.disabled = true;
        button.textContent = 'Capturing';
        try {
          const result = await api('versions', {method: 'POST', body: JSON.stringify({action: 'capture', label})});
          renderVersions(result);
          versionCreate.hidden = true;
          q('#version-new').hidden = false;
          input.value = '';
          toast(`Version ${label} created`);
        } catch (captureError) {
          error.textContent = captureError.message;
          error.hidden = false;
          toast(captureError.message, true);
        }
        button.disabled = false;
        button.textContent = 'Capture build';
      });
      versionList?.addEventListener('click', async event => {
        const verify = event.target.closest('[data-version-verify]');
        const current = event.target.closest('[data-version-current]');
        const remove = event.target.closest('[data-version-remove]');
        if (!verify && !current && !remove) return;
        const button = verify || current || remove;
        const label = verify?.dataset.versionVerify || current?.dataset.versionCurrent || remove?.dataset.versionRemove;
        if (remove && remove.dataset.confirm !== 'true') {
          remove.dataset.confirm = 'true';
          remove.textContent = 'Confirm remove';
          setTimeout(() => { if (remove.isConnected) { remove.dataset.confirm = ''; remove.textContent = 'Remove'; } }, 4000);
          return;
        }
        button.disabled = true;
        try {
          const action = verify ? 'verify' : current ? 'current' : 'remove';
          const result = await api('versions', {method: 'POST', body: JSON.stringify({action, label})});
          renderVersions(result);
          toast(verify ? `Version ${label} verified` : current ? `Version ${label} is current` : `Version ${label} removed`);
        } catch (error) { button.disabled = false; toast(error.message, true); }
      });
      void loadVersions();

      const showConfigPanel = selected => {
        root.querySelectorAll('[data-config-tab]').forEach(item => {
          const active = item.dataset.configTab === selected;
          item.classList.toggle('active', active);
          item.setAttribute('aria-selected', String(active));
        });
        root.querySelectorAll('[data-config-panel]').forEach(panel => panel.hidden = panel.dataset.configPanel !== selected);
        q(`[data-config-panel="${selected}"] input, [data-config-panel="${selected}"] textarea`)?.focus();
      };
      root.querySelectorAll('[data-config-tab]').forEach(tab => tab.addEventListener('click', () => showConfigPanel(tab.dataset.configTab)));
      const settingsPages = [
        {id: 'site', label: 'Site details'},
        {id: 'languages', label: 'Languages'},
        {id: 'brand', label: 'Brand'},
        {id: 'social', label: 'Navigation and links'},
        {id: 'build', label: 'Project files'},
        {id: 'blog', label: 'Blog'},
        {id: 'theme', label: 'Appearance'},
        {id: 'accessibility', label: 'Accessibility'},
        {id: 'layout', label: 'Layout'},
        {id: 'versioning', label: 'Versions'},
        {id: 'translation', label: 'Translation provider'},
        {id: 'deploy', label: 'Deployment targets'}
      ];
      let settingsPageIndex = 0;
      const showSettingsPage = (id, focus = false) => {
        const nextIndex = settingsPages.findIndex(page => page.id === id);
        if (nextIndex < 0) return;
        settingsPageIndex = nextIndex;
        const current = settingsPages[settingsPageIndex];
        updateWorkspacePageHeader(current.id);
        if (completeSettingsActions) completeSettingsActions.hidden = current.id === 'blog' && activeBlogPanel === 'posts';
        let activeTab;
        root.querySelectorAll('[data-settings-tab]').forEach(tab => {
          const active = tab.dataset.settingsTab === current.id;
          tab.classList.toggle('active', active);
          tab.setAttribute('aria-selected', String(active));
          tab.tabIndex = active ? 0 : -1;
          if (active) activeTab = tab;
        });
        root.querySelectorAll('[data-settings-page]').forEach(page => page.hidden = page.dataset.settingsPage !== current.id);
        q('#settings-step-label').textContent = `${current.label} · ${settingsPageIndex + 1} of ${settingsPages.length}`;
        q('#settings-previous').disabled = settingsPageIndex === 0;
        q('#settings-next').hidden = settingsPageIndex === settingsPages.length - 1;
        setWorkspaceActive(`config-${id}`);
        activeTab?.scrollIntoView({block: 'nearest', inline: 'nearest', behavior: 'smooth'});
        if (focus) q(`[data-settings-page="${current.id}"] input, [data-settings-page="${current.id}"] textarea, [data-settings-page="${current.id}"] select`)?.focus();
      };
      const currentSettingsPageIsValid = () => {
        const page = q(`[data-settings-page="${settingsPages[settingsPageIndex].id}"]`);
        const invalid = Array.from(page.querySelectorAll('input, textarea, select')).find(control => !control.checkValidity());
        if (!invalid) return true;
        invalid.reportValidity();
        invalid.focus();
        return false;
      };
      root.querySelectorAll('[data-settings-tab]').forEach(tab => {
        tab.addEventListener('click', () => showSettingsPage(tab.dataset.settingsTab, true));
        tab.addEventListener('keydown', event => {
          const movement = event.key === 'ArrowRight' || event.key === 'ArrowDown' ? 1 : event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -1 : 0;
          const destination = event.key === 'Home' ? 0 : event.key === 'End' ? settingsPages.length - 1 : Math.min(settingsPages.length - 1, Math.max(0, settingsPageIndex + movement));
          if (!movement && event.key !== 'Home' && event.key !== 'End') return;
          event.preventDefault();
          showSettingsPage(settingsPages[destination].id);
          q(`[data-settings-tab="${settingsPages[destination].id}"]`).focus();
        });
      });
      q('#settings-previous').addEventListener('click', () => showSettingsPage(settingsPages[settingsPageIndex - 1].id, true));
      q('#settings-next').addEventListener('click', () => {
        if (currentSettingsPageIsValid()) showSettingsPage(settingsPages[settingsPageIndex + 1].id, true);
      });
      if (initialPage) {
        showConfigPanel('complete');
        showSettingsPage(initialPage, true);
        if (initialPage === 'blog') {
          showBlogPanel(initialBlogView === 'preferences' ? 'preferences' : 'posts');
          if (initialBlogView === 'new') void openBlogEditor('');
        }
      } else setWorkspaceActive('quick');
      q('#config-complete-form').addEventListener('invalid', event => {
        const page = event.target.closest('[data-settings-page]');
        if (page?.hidden) showSettingsPage(page.dataset.settingsPage);
      }, true);
      const appendRow = (listSelector, html) => {
        const list = q(listSelector);
        list.querySelector('.empty-repeat')?.remove();
        list.insertAdjacentHTML('beforeend', html);
        installStyledSelects(list);
        return list.lastElementChild;
      };
      q('#add-language').addEventListener('click', () => {
        const existing = Array.from(q('#language-list').querySelectorAll('[data-language-code]')).map(input => input.value);
        const suggestion = suggestLanguage(existing);
        appendRow('#language-list', languageRow(suggestion.code, suggestion.label))?.querySelector('[data-language-code]')?.focus();
      });
      q('#add-header-link').addEventListener('click', () => appendRow('#header-link-list', headerLinkRow())?.querySelector('[data-header-label]')?.focus());
      q('#add-deploy-target').addEventListener('click', () => appendRow('#deploy-target-list', deployTargetRow()));
      q('#open-translation-workspace')?.addEventListener('click', showTranslations);
      const layoutPresets = {
        starlight: {contentWidth: '45rem', wideContentWidth: '64rem', sidebarWidth: '18.75rem', tocWidth: '16rem', contentTocGap: '2.5rem', alignment: 'cluster', toc: 'right'},
        wide: {contentWidth: '60rem', wideContentWidth: '72rem', sidebarWidth: '18.75rem', tocWidth: '16rem', contentTocGap: '2.5rem', alignment: 'cluster', toc: 'right'},
        reading: {contentWidth: '65ch', wideContentWidth: '52rem', sidebarWidth: '17rem', tocWidth: '15rem', contentTocGap: '2rem', alignment: 'cluster', toc: 'right'}
      };
      const layoutPreset = q('[name="theme.layout.preset"]');
      const syncLayoutPreset = applyValues => {
        const custom = layoutPreset.value === 'custom';
        const values = layoutPresets[layoutPreset.value];
        for (const name of ['contentWidth', 'wideContentWidth', 'sidebarWidth', 'tocWidth', 'contentTocGap', 'alignment', 'toc']) {
          const control = q(`[name="theme.layout.${name}"]`);
          control.disabled = !custom;
          control.closest('.mpress-form-field').hidden = !custom;
          if (applyValues && values) control.value = values[name];
          control.mpressSelectSync?.();
        }
      };
      layoutPreset.addEventListener('change', () => syncLayoutPreset(true));
      syncLayoutPreset(false);
      q('#config-complete-form').addEventListener('click', event => {
        const remove = event.target.closest('[data-remove-row]');
        if (!remove) return;
        const list = remove.closest('.repeat-list');
        remove.closest('.repeat-row')?.remove();
        if (list && !list.querySelector('.repeat-row')) {
          const empty = list.id === 'language-list' ? 'Add at least one language.' : list.id === 'header-link-list' ? 'No header links yet.' : 'No deployment targets yet.';
          list.innerHTML = `<p class="empty-repeat">${empty}</p>`;
        }
      });
      const previewQuickConfig = () => {
        const form = q('#config-form');
        applyConfigPreview({
          title: form.elements.title.value,
          description: form.elements.description.value,
          colorScheme: form.elements.colorScheme.value,
          accentColor: form.elements.accentColor.value,
          hoverColorLight: form.elements.hoverColorLight.value,
          hoverColorDark: form.elements.hoverColorDark.value
        });
      };
      const previewCompleteConfig = () => {
        const form = q('#config-complete-form');
        applyConfigPreview({
          title: form.elements.namedItem('site.title')?.value,
          description: form.elements.namedItem('site.description')?.value,
          colorScheme: form.elements.namedItem('theme.colorScheme')?.value,
          accentColor: form.elements.namedItem('theme.accentColor')?.value,
          hoverColorLight: form.elements.namedItem('theme.hoverColorLight')?.value,
          hoverColorDark: form.elements.namedItem('theme.hoverColorDark')?.value,
          searchEnabled: form.elements.namedItem('search.enabled')?.checked,
          searchPlaceholder: form.elements.namedItem('search.placeholder')?.value
        });
      };
      q('#config-form').addEventListener('input', previewQuickConfig);
      q('#config-form').addEventListener('change', previewQuickConfig);
      q('#config-complete-form').addEventListener('input', previewCompleteConfig);
      q('#config-complete-form').addEventListener('change', previewCompleteConfig);
      const configResetGuided = q('#config-reset-guided, #config-form button[type="reset"]');
      configResetGuided?.addEventListener('click', event => {
        // The Markdown form uses a native reset button. Keep the browser from
        // clearing hydrated values before the saved configuration is restored.
        event.preventDefault();
        restoreConfigPreview();
        void showConfig(fromWizard);
        toast('Unsaved changes reverted');
      });
      q('#config-reset-complete').addEventListener('click', () => {
        const currentPage = settingsPages[settingsPageIndex].id;
        restoreConfigPreview();
        void showConfig(fromWizard, currentPage);
        toast('Unsaved changes reverted');
      });
      q('#config-form').addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Saving';
        const values = Object.fromEntries(new FormData(event.currentTarget));
        try {
          if (fromWizard) sessionStorage.setItem('mpress-wizard-step', '3');
          const result = await api('config', {method: 'PUT', body: JSON.stringify({...values, revision: data.revision})});
          updateBuild(result.state);
          if (state.configPreview) state.configPreview.committed = true;
          restoreConfigPreview();
          data.revision = result.revision;
          initialSettingsState = settingsState();
          button.disabled = false;
          button.textContent = 'Save and rebuild';
          toast('Quick setup saved');
          if (fromWizard) wizard(3); else if (!workspaceMode) closeDrawer();
        } catch (error) {
          button.disabled = false;
          button.textContent = 'Save and rebuild';
          toast(error.message, true);
        }
      });
      q('#config-complete-form').addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Validating';
        try {
          const form = event.currentTarget;
          const value = name => form.elements.namedItem(name)?.value.trim() || '';
          const enabled = name => Boolean(form.elements.namedItem(name)?.checked);
          const complete = JSON.parse(JSON.stringify(cfg));
          complete.site.title = value('site.title');
          complete.site.description = value('site.description');
          complete.site.baseURL = value('site.baseURL');
          complete.site.defaultLanguage = value('site.defaultLanguage');
          complete.site.defaultLanguageAtRoot = enabled('site.defaultLanguageAtRoot');
          complete.site.missingTranslation = value('site.missingTranslation');
          complete.site.logoLight = value('site.logoLight');
          complete.site.logoDark = value('site.logoDark');
          delete complete.site.logoWidth;
          complete.site.favicon = value('site.favicon');
          complete.site.socialImage = value('site.socialImage');
          complete.site.languages = [];
          complete.site.languageLabels = {};
          form.querySelectorAll('.language-row').forEach(row => {
            const code = row.querySelector('[data-language-code]').value.trim();
            const label = row.querySelector('[data-language-label]').value.trim();
            if (!code) return;
            complete.site.languages.push(code);
            if (label) complete.site.languageLabels[code] = label;
          });
          complete.site.headerLinks = Array.from(form.querySelectorAll('.header-link-row')).map(row => {
            const type = row.querySelector('[data-header-type]')?.value || 'link';
            const variant = row.querySelector('[data-header-variant]')?.value || 'primary';
            const link = {label: row.querySelector('[data-header-label]').value.trim(), url: row.querySelector('[data-header-url]').value.trim(), type};
            if (type === 'button') link.variant = variant;
            if (type === 'button' && variant === 'custom') link.color = row.querySelector('[data-header-color]')?.value || '#5375f6';
            return link;
          });
          complete.social.github = value('social.github');
          complete.social.discord = value('social.discord');
          complete.social.reddit = value('social.reddit');
          complete.social.x = value('social.x');
          complete.social.rss = value('social.rss');
          complete.social.sponsor = value('social.sponsor');
          complete.social.editURL = value('social.editURL');
          complete.contribution ||= {};
          complete.contribution.enabled = enabled('contribution.enabled');
          complete.contribution.repository = value('contribution.repository');
          complete.contribution.branch = value('contribution.branch');
          complete.contribution.guide = value('contribution.guide');
          complete.contribution.quickEdit = enabled('contribution.quickEdit');
          complete.build.contentDir = value('build.contentDir');
          complete.build.staticDir = value('build.staticDir');
          complete.build.outputDir = value('build.outputDir');
          complete.build.navFile = value('build.navFile');
          complete.build.customCSS = value('build.customCSS');
          complete.knowledge ||= {};
          complete.knowledge.enabled = enabled('knowledge.enabled');
          complete.blog ||= {};
          complete.blog.landingStyle = value('blog.landingStyle');
          complete.blog.tagline = value('blog.tagline');
          complete.blog.showTags = enabled('blog.showTags');
          complete.blog.headingSize = value('blog.headingSize');
          complete.blog.imageMode = value('blog.imageMode');
          complete.blog.imageFit = value('blog.imageFit');
          complete.blog.imageWidth = Number.parseInt(value('blog.imageWidth') || '100', 10);
          // Background colour belongs to a framed panel. A floating image
          // must not retain an invisible panel background setting.
          complete.blog.imageBackground = complete.blog.imageMode === 'floating' ? '' : value('blog.imageBackground');
          complete.theme.colorScheme = value('theme.colorScheme');
          complete.theme.accentColor = value('theme.accentColor');
          complete.theme.hoverColorLight = value('theme.hoverColorLight');
          complete.theme.hoverColorDark = value('theme.hoverColorDark');
          complete.theme.layout ||= {};
          complete.theme.layout.preset = value('theme.layout.preset');
          complete.theme.layout.contentWidth = value('theme.layout.contentWidth');
          complete.theme.layout.wideContentWidth = value('theme.layout.wideContentWidth');
          complete.theme.layout.sidebarWidth = value('theme.layout.sidebarWidth');
          complete.theme.layout.tocWidth = value('theme.layout.tocWidth');
          complete.theme.layout.contentTocGap = value('theme.layout.contentTocGap');
          complete.theme.layout.alignment = value('theme.layout.alignment');
          complete.theme.layout.toc = value('theme.layout.toc');
          complete.search.enabled = enabled('search.enabled');
          complete.search.shortcut = value('search.shortcut');
          complete.search.placeholder = value('search.placeholder');
          complete.search.maxResults = Number.parseInt(value('search.maxResults') || '12', 10);
          complete.search.rememberRecent = enabled('search.rememberRecent');
          complete.accessibility ||= {};
          complete.accessibility.enabled = enabled('accessibility.enabled');
          complete.accessibility.shortcut = value('accessibility.shortcut');
          complete.versioning.enabled = enabled('versioning.enabled');
          complete.versioning.current = value('versioning.current');
          complete.versioning.artifactsDir = value('versioning.artifactsDir');
          complete.translation.provider = value('translation.provider');
          complete.translation.model = value('translation.model');
          complete.translation.reasoningEffort = value('translation.reasoningEffort');
          complete.translation.command = value('translation.command');
          complete.translation.baseURL = value('translation.baseURL');
          complete.translation.apiKeyEnv = value('translation.apiKeyEnv');
          complete.translation.sourceLanguage = value('translation.sourceLanguage');
          complete.translation.glossary = value('translation.glossary');
          complete.translation.styleGuide = value('translation.styleGuide');
          complete.translation.stateDir = value('translation.stateDir');
		  complete.translation.inputPricePerMillion = Number(value('translation.inputPricePerMillion') || 0);
		  complete.translation.outputPricePerMillion = Number(value('translation.outputPricePerMillion') || 0);
          complete.translation.dataCollection = value('translation.dataCollection');
          complete.translation.requireParameters = enabled('translation.requireParameters');
          complete.deploy.default = value('deploy.default');
          complete.deploy.targets = {};
          for (const row of form.querySelectorAll('.deploy-target')) {
            const name = row.querySelector('[data-deploy-name]').value.trim();
            if (complete.deploy.targets[name]) throw new Error(`Deployment target ${name} is duplicated`);
            complete.deploy.targets[name] = {
              provider: row.querySelector('[data-deploy-provider]').value,
              accountID: row.querySelector('[data-deploy-account]').value.trim(),
              project: row.querySelector('[data-deploy-project]').value.trim(),
              productionBranch: row.querySelector('[data-deploy-branch]').value.trim(),
              domain: row.querySelector('[data-deploy-domain]').value.trim()
            };
          }
          if (fromWizard) sessionStorage.setItem('mpress-wizard-step', '3');
          const result = await api('config', {method: 'PUT', body: JSON.stringify({config: complete, complete: true, revision: data.revision})});
          updateBuild(result.state);
          if (state.configPreview) state.configPreview.committed = true;
          restoreConfigPreview();
          data.revision = result.revision;
          initialSettingsState = settingsState();
          button.disabled = false;
          button.textContent = 'Save and rebuild';
          toast('Settings saved');
          if (fromWizard) wizard(3); else if (!workspaceMode) closeDrawer();
        } catch (error) {
          button.disabled = false;
          button.textContent = 'Save and rebuild';
          toast(error.message, true);
        }
      });
      initialSettingsState = settingsState();
      workspaceHasUnsavedChanges = () => settingsHaveChanged() || (blogEditorOpen && blogEditorInitialState && blogEditorState() !== blogEditorInitialState);
      canLeaveWorkspace = () => {
        if (!workspaceHasUnsavedChanges()) return true;
        return window.confirm('Discard unsaved changes?');
      };
    } catch (error) { restoreConfigPreview(); toast(error.message, true); }
  };
  const showTranslations = async () => {
    let route = null;
    try { route = await api('route?url=' + encodeURIComponent(siteRoute)); } catch (_) {}
    let data;
    const loadTranslationData = async (language = '', file = '') => {
      const query = new URLSearchParams();
      if (language) query.set('lang', language);
      if (file) query.set('file', file);
      data = await api('translations' + (query.size ? '?' + query : ''));
      return data;
    };
    let activeLanguage = '';
    try {
      await loadTranslationData();
      activeLanguage = (data.languages || []).find(language => language !== data.defaultLanguage) || '';
      if (activeLanguage) await loadTranslationData(activeLanguage);
    } catch (error) { toast(error.message, true); return; }
    const targets = () => (data.languages || []).filter(language => language !== data.defaultLanguage);
    const languageLabel = language => data.languageLabels?.[language] || language;
    const languageOptions = selected => targets().map(language => `<option value="${escapeHTML(language)}"${selectedAttr(language, selected)}>${escapeHTML(languageLabel(language))}</option>`).join('');
    const formats = () => data.documentFormats || {markdown: 0, mpd: 0, nativeMPD: false};
    const automatic = () => data.automatic || {};
    const selectedStage = stage => automatic()?.[stage]?.selection || {};
    const bindBack = handler => q('[data-translation-back]')?.addEventListener('click', handler);
    const totalPending = () => (data.report?.files || []).reduce((sum, file) => sum + (file.needsTranslation || 0), 0);
    const totalSegments = () => (data.report?.files || []).reduce((sum, file) => {
	  const states = file.states || {};
	  const prose = Object.entries(states).reduce((count, [state, value]) => state === 'protected' ? count : count + (Number(value) || 0), 0);
	  return sum + (prose || file.segments || 0);
	}, 0);
    const providerName = selection => selection.provider === 'codex' ? 'Codex subscription' : selection.provider === 'claude' ? 'Claude Code subscription' : selection.provider === 'openrouter' ? 'OpenRouter' : selection.provider === 'openai' ? 'OpenAI API' : 'Not available';
    const modelName = selection => selection.model || 'No model selected';
    const pipelineHTML = () => {
      const draft = selectedStage('draft');
      const refinement = selectedStage('refinement');
      const audit = selectedStage('audit');
      return `<ol class="translation-pipeline" aria-label="Automatic translation stages">
        <li><span>1</span><div><strong>Initial translation</strong><small>${escapeHTML(modelName(draft))} through ${escapeHTML(providerName(draft))}</small></div>${iconHTML('check')}</li>
        <li><span>2</span><div><strong>Quality refinement</strong><small>${escapeHTML(modelName(refinement))} is ready for flagged passages</small></div>${automatic().refinement?.ready ? iconHTML('check') : iconHTML('minus')}</li>
        <li><span>3</span><div><strong>Independent checks</strong><small>${escapeHTML(modelName(audit))} plus M-Press structure and terminology checks</small></div>${automatic().audit?.ready ? iconHTML('check') : iconHTML('minus')}</li>
        <li><span>4</span><div><strong>Human approval</strong><small>Review only the passages that still need attention</small></div>${iconHTML('check-circle')}</li>
      </ol>`;
    };
    const setupHTML = () => {
      const draft = selectedStage('draft');
      if (!automatic().ready) return `<div class="translation-readiness blocked">${iconHTML('triangle-alert')}<div><strong>No translation service found</strong><span>Install Codex or Claude Code, or connect OpenRouter or OpenAI.</span></div><button class="button" id="translation-advanced" type="button">Set up a provider</button></div>`;
      const compare = data.provider === 'openrouter' && targets().length ? '<button class="button quiet" id="translation-compare-models" type="button">Compare models</button>' : '';
      return `<div class="translation-readiness ready">${iconHTML('star')}<div><strong>Automatic setup is ready</strong><span>M-Press selected ${escapeHTML(modelName(draft))} through ${escapeHTML(providerName(draft))}. You can inspect or change this under Advanced.</span></div><div class="translation-readiness-actions">${compare}<button class="button quiet" id="translation-advanced" type="button">Advanced</button></div></div>`;
    };
    const bindAdvanced = () => {
      q('#translation-advanced')?.addEventListener('click', () => ensureRepositoryReady(() => showConfig(false, 'translation')));
      q('#translation-compare-models')?.addEventListener('click', () => renderModelChooser());
    };
    const returnOverview = async () => { await loadTranslationData(activeLanguage); renderOverview(); };
    let modelCatalog = null;
    const comparisonSegments = segments => (segments || []).map(segment => `<div class="translation-sample-segment">${segment.section ? `<small>${escapeHTML(segment.section)}</small>` : ''}<p>${escapeHTML(segment.text)}</p></div>`).join('');
    const renderModelChooser = async () => {
      if (!modelCatalog) {
        setDrawerBody('Compare translation models', '<div class="result-box"><strong>Loading current models</strong>Reading OpenRouter pricing and compatible text models. No translation request is being made.</div>', 'Choose using evidence from this documentation rather than a generic leaderboard.');
        try { modelCatalog = await api('translation-models'); }
        catch (error) {
          setDrawerBody('Compare translation models', `<div class="result-box"><strong>Model catalog unavailable</strong>${escapeHTML(error.message)}</div><div class="actions"><button class="button" type="button" data-translation-back>Back</button></div>`, 'OpenRouter model comparison needs a configured OpenRouter provider.');
          bindBack(() => void returnOverview());
          return;
        }
      }
      const models = modelCatalog.models || [];
      const language = activeLanguage || targets()[0] || '';
      const recommendations = modelCatalog.recommendations?.[language]?.modelIds || [];
      const configured = modelCatalog.configuredLanguageModels?.[language]?.model || modelCatalog.configuredModel || '';
      const first = configured || recommendations[0] || models[0]?.id || '';
      const second = recommendations.find(id => id !== first) || models.find(model => model.id !== first)?.id || '';
      const options = selected => models.map(model => `<option value="${escapeHTML(model.id)}"${selectedAttr(model.id, selected)}>${escapeHTML(model.name || model.id)} · about $${Number(model.sampleCost || 0).toFixed(3)} for this sample</option>`).join('');
      setDrawerBody('Compare translation models', `<form id="translation-model-comparison" class="form-grid"><div class="translation-comparison-intro settings-wide"><h3>Run a blind comparison</h3><p>M-Press sends the same protected sample to two models. It does not write either result to the site.</p></div><label class="field settings-wide"><span>Target language</span><select name="language">${languageOptions(language)}</select></label><label class="field"><span>Option A model</span><select name="modelA">${options(first)}</select></label><label class="field"><span>Option B model</span><select name="modelB">${options(second)}</select></label><div class="actions"><button class="button" type="button" data-translation-back>Back</button><button class="button primary" type="submit"${modelCatalog.hasAPIKey ? '' : ' disabled'}>Run blind comparison</button></div></form>`, modelCatalog.hasAPIKey ? 'The model names are hidden while you judge the output.' : `Set ${modelCatalog.apiKeyEnv || 'OPENROUTER_API_KEY'} before running a paid comparison.`);
      bindBack(() => void returnOverview());
      const form = q('#translation-model-comparison');
      installStyledSelects(form);
      const sync = () => { form.querySelector('[type="submit"]').disabled = !modelCatalog.hasAPIKey || form.elements.modelA.value === form.elements.modelB.value; };
      form.elements.modelA.addEventListener('change', sync);
      form.elements.modelB.addEventListener('change', sync);
      sync();
      form.addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        const candidates = [form.elements.modelA.value, form.elements.modelB.value];
        button.disabled = true;
        button.textContent = 'Comparing';
        try {
          const comparison = await api('translation-models', {method: 'POST', body: JSON.stringify({action: 'compare', models: candidates, language: form.elements.language.value, file: route?.path || ''})});
          renderModelComparison(comparison);
        } catch (error) { button.disabled = false; button.textContent = 'Run blind comparison'; toast(error.message, true); }
      });
    };
    const renderModelComparison = comparison => {
      setDrawerBody('Choose the better translation', `<section class="translation-source-sample"><h3>Original text</h3>${comparisonSegments(comparison.source)}</section><div class="translation-comparison-grid">${(comparison.outputs || []).map(option => `<article class="translation-comparison-option"><header><strong>Option ${escapeHTML(option.label)}</strong></header>${comparisonSegments(option.texts)}<button class="button primary" type="button" data-comparison-winner="${escapeHTML(option.label)}">Choose option ${escapeHTML(option.label)}</button></article>`).join('')}</div><div class="actions"><button class="button" type="button" data-translation-back>Back</button></div>`, 'Judge technical accuracy, terminology, clarity, and natural language. Model names remain hidden until you choose.');
      bindBack(() => void renderModelChooser());
      root.querySelectorAll('[data-comparison-winner]').forEach(button => button.addEventListener('click', async () => {
        root.querySelectorAll('[data-comparison-winner]').forEach(item => { item.disabled = true; });
        try {
          const result = await api('translation-models', {method: 'POST', body: JSON.stringify({action: 'judge', comparisonId: comparison.comparisonId, winner: button.dataset.comparisonWinner})});
          renderModelWinner(result);
        } catch (error) { root.querySelectorAll('[data-comparison-winner]').forEach(item => { item.disabled = false; }); toast(error.message, true); }
      }));
    };
    const renderModelWinner = result => {
      const winner = (modelCatalog?.models || []).find(model => model.id === result.winnerModel);
      const loser = (modelCatalog?.models || []).find(model => model.id === result.loserModel);
      setDrawerBody('Comparison result', `<div class="translation-winner"><span class="wizard-kicker">Preferred for ${escapeHTML(languageLabel(result.language))}</span><h3>${escapeHTML(winner?.name || result.winnerModel)}</h3><p>It beat ${escapeHTML(loser?.name || result.loserModel)} on this sample. Select it for this language or challenge it with another model.</p></div><div class="actions"><button class="button" id="translation-challenge-model" type="button">Compare again</button><button class="button primary" id="translation-use-model" type="button">Use this model</button></div>`, 'This choice applies only to the selected language.');
      q('#translation-challenge-model').addEventListener('click', () => void renderModelChooser());
      q('#translation-use-model').addEventListener('click', async event => {
        event.currentTarget.disabled = true;
        try {
          const saved = await api('translation-models', {method: 'POST', body: JSON.stringify({action: 'set-model', model: result.winnerModel, language: result.language})});
          data.languageModels = saved.languageModels || data.languageModels || {};
          modelCatalog.configuredLanguageModels = data.languageModels;
          toast(`${winner?.name || result.winnerModel} selected for ${languageLabel(result.language)}`);
          await returnOverview();
        } catch (error) { event.currentTarget.disabled = false; toast(error.message, true); }
      });
    };
    const renderOverview = () => {
      if (!targets().length) { renderAddLanguage(true); return; }
      const selected = activeLanguage || targets()[0];
      const pending = totalPending();
      const segments = totalSegments();
      const current = Math.max(0, segments - pending);
      const percent = segments ? Math.round(current / segments * 100) : 0;
      openDrawer('Translations', 'Translations', `
        <section class="translation-hero">
          <div>${targets().length > 1 ? `<label class="translation-overview-language"><span>Language</span><select id="translation-overview-language">${languageOptions(selected)}</select></label>` : `<span class="wizard-kicker">${escapeHTML(languageLabel(selected))}</span>`}<h2>${pending ? `Continue ${escapeHTML(languageLabel(selected))}` : `${escapeHTML(languageLabel(selected))} is current`}</h2><p>${pending ? `${pending} passage${pending === 1 ? '' : 's'} need translation or updating.` : 'Every source passage has a current translation.'}</p></div>
          <div class="translation-score" aria-label="${percent} percent current"><strong>${percent}%</strong><span>current</span></div>
        </section>
        <div class="translation-stats"><div><strong>${(data.report?.files || []).length}</strong><span>pages</span></div><div><strong>${current}</strong><span>current passages</span></div><div><strong>${pending}</strong><span>need attention</span></div></div>
        ${setupHTML()}
        <h3 class="translation-section-heading">What M-Press will do</h3>
        ${pipelineHTML()}
        <div class="actions translation-primary-actions"><button class="button" id="translation-add-language" type="button">Add another language</button>${route?.path ? '<button class="button" id="translation-this-page" type="button">This page only</button>' : ''}<button class="button primary" id="translation-continue" type="button">${pending ? `Translate ${pending} passage${pending === 1 ? '' : 's'}` : 'Run quality checks'}</button></div>`, 'tool', 'A guided workflow protects structure, preserves human edits, and uses the best translation services available on this computer.');
      bindAdvanced();
	  q('#translation-overview-language')?.addEventListener('change', async event => { activeLanguage = event.currentTarget.value; await loadTranslationData(activeLanguage); renderOverview(); });
      q('#translation-add-language').addEventListener('click', () => renderAddLanguage(false));
      q('#translation-this-page')?.addEventListener('click', () => renderTranslationPlan('page', selected));
      q('#translation-continue').addEventListener('click', () => renderTranslationPlan('site', selected));
    };
    const localeChoices = [
      ['fr','Français'],['de','Deutsch'],['es','Español'],['it','Italiano'],['pt-BR','Português (Brasil)'],['pt-PT','Português (Portugal)'],
      ['nl','Nederlands'],['pl','Polski'],['uk','Українська'],['ru','Русский'],['zh-CN','简体中文'],['zh-TW','繁體中文'],['ja','日本語'],['ko','한국어'],
      ['ar','العربية'],['he','עברית'],['hi','हिन्दी'],['id','Bahasa Indonesia'],['tr','Türkçe'],['sv','Svenska']
    ];
    const renderAddLanguage = first => {
      setDrawerBody('Add a language', `<form id="translation-language-form" class="form-grid translation-language-form"><label class="field settings-wide"><span>Language</span><select name="locale" required><option value="">Choose a language</option>${localeChoices.filter(([code]) => !data.languages?.includes(code)).map(([code,label]) => `<option value="${escapeHTML(code)}">${escapeHTML(label)} · ${escapeHTML(code)}</option>`).join('')}<option value="custom">Another language or locale</option></select><small>M-Press configures its label, route, script, and translation guidance automatically.</small></label><div class="translation-custom-locale" hidden><label class="field"><span>Language code</span><input name="language" pattern="[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*" maxlength="24" placeholder="cy"></label><label class="field"><span>Display name</span><input name="label" placeholder="Cymraeg"></label></div><div class="actions">${first ? '<button class="button" type="button" data-translation-close>Not now</button>' : '<button class="button" type="button" data-translation-back>Back</button>'}<button class="button primary" type="submit">Add language and continue</button></div></form>`, 'Choose the language. M-Press will prepare the source, select the best available models, and show the complete plan before spending anything.');
      if (first) q('[data-translation-close]').addEventListener('click', closeDrawer); else bindBack(() => void returnOverview());
      const form = q('#translation-language-form');
      form.elements.locale.addEventListener('change', () => {
        const custom = form.elements.locale.value === 'custom';
        q('.translation-custom-locale').hidden = !custom;
        form.elements.language.required = custom;
        form.elements.label.required = custom;
        if (custom) form.elements.language.focus();
      });
      form.addEventListener('submit', async event => {
        event.preventDefault();
        const locale = event.currentTarget.elements.locale.value;
        const known = localeChoices.find(([code]) => code === locale);
        const language = known?.[0] || event.currentTarget.elements.language.value.trim();
        const label = known?.[1] || event.currentTarget.elements.label.value.trim();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Preparing language';
        try {
          const result = await api('translations', {method: 'POST', body: JSON.stringify({action: 'add-language', language, label})});
          updateBuild(result.state);
          await loadTranslationData(language);
		  activeLanguage = language;
          toast(`${label} added`);
          renderTranslationPlan('site', language);
        } catch (error) { button.disabled = false; button.textContent = 'Add language and continue'; toast(error.message, true); }
      });
    };
    const renderTranslationPlan = (goal, selectedLanguage = '') => {
      const isPage = goal === 'page';
      const selected = selectedLanguage || targets()[0];
      setDrawerBody(isPage ? 'Translate this page' : `Translate ${languageLabel(selected)}`, `<form id="translation-plan-form" class="form-grid"><label class="field"><span>Target language</span><select name="language">${languageOptions(selected)}</select></label><label class="field"><span>Update</span><select name="scope"><option value="stale">Everything that needs attention</option><option value="missing">Only missing translations</option><option value="all">Replace previous machine translations</option></select></label>${isPage ? `<div class="result-box settings-wide"><strong>Page</strong>${escapeHTML(route?.path || '')}</div>` : ''}<label class="check-field settings-wide"><input name="force" type="checkbox"><span><strong>Replace passages changed by a person</strong><small>Leave this off to preserve every manual edit.</small></span></label>${formats().nativeMPD ? '<div class="translation-preparation ready">' + iconHTML('check') + '<span><strong>Source is ready</strong>All documents already use native MPD.</span></div>' : '<div class="translation-preparation">' + iconHTML('zap') + `<span><strong>Source preparation included</strong>M-Press will convert ${formats().markdown} Markdown document${formats().markdown === 1 ? '' : 's'} to MPD and validate the result before translation.</span></div>`}${setupHTML()}<div class="actions"><button class="button" type="button" data-translation-back>Back</button><button class="button primary" type="submit"${automatic().ready ? '' : ' disabled'}>Review the plan</button></div></form>`, 'Choose the outcome. M-Press preserves code, links, metadata, components, and human edits.');
      bindBack(() => void returnOverview());
      bindAdvanced();
      q('#translation-plan-form').addEventListener('submit', async event => {
        event.preventDefault();
        const form = event.currentTarget;
        const plan = {goal, language: form.elements.language.value, scope: form.elements.scope.value, force: form.elements.force.checked, file: isPage ? route?.path || '' : '', workers: 4};
        try {
          const planData = await loadTranslationData(plan.language, plan.file);
          plan.report = planData.report;
          renderTranslationConfirmation(plan);
        } catch (error) { toast(error.message, true); }
      });
    };
    const renderTranslationConfirmation = plan => {
      const label = languageLabel(plan.language);
      const estimate = plan.report?.estimate || {};
      const pending = plan.report?.pending || 0;
      const files = plan.report?.files?.length || 0;
      const draft = selectedStage('draft');
      const refinement = selectedStage('refinement');
      const subscriptionDraft = draft.provider === 'codex' || draft.provider === 'claude';
      const subscriptionRefinement = refinement.provider === 'codex' || refinement.provider === 'claude';
      let cost = 'Provider usage appears in your provider account';
      if (estimate.pricingConfigured) cost = `About $${Number(estimate.totalCostUSD || 0).toFixed(2)} for the initial pass; refinement depends on findings`;
      else if (subscriptionDraft && subscriptionRefinement) cost = 'Initial translation and refinement use your subscriptions';
      else if (subscriptionDraft) cost = 'Initial translation uses your subscription; only flagged text may use a paid API';
      setDrawerBody('Review translation plan', `<div class="translation-plan-summary"><div><span>Language</span><strong>${escapeHTML(label)}</strong></div><div><span>Scope</span><strong>${files} page${files === 1 ? '' : 's'} · ${pending} passage${pending === 1 ? '' : 's'}</strong></div><div><span>Cost</span><strong>${escapeHTML(cost)}</strong></div><div><span>Human work</span><strong>${plan.force ? 'Replace conflicting edits' : 'Preserve all manual edits'}</strong></div></div>${pipelineHTML()}<div class="translation-assurance">${iconHTML('check-circle')}<span><strong>Nothing is published automatically</strong>M-Press writes only structurally valid pages to this private branch. You will review the result before submitting it.</span></div><div class="actions"><button class="button" type="button" data-translation-back>Back</button><button class="button primary" id="translation-run" type="button">Start translation</button></div>`, `M-Press has prepared a safe, resumable plan for ${label}.`);
      bindBack(() => renderTranslationPlan(plan.goal, plan.language));
      q('#translation-run').addEventListener('click', () => runTranslation(plan));
    };
    const renderProgress = (label, activeStage, detail) => {
	  const stages = ['Prepare source','Translate prose','Review and refine quality','Prepare human review'];
      setDrawerBody(`Translating ${label}`, `<div class="translation-progress" role="status" aria-live="polite"><div class="translation-progress-heading"><span class="translation-spinner" aria-hidden="true"></span><div><strong>${escapeHTML(stages[activeStage])}</strong><span>${escapeHTML(detail)}</span></div></div><ol>${stages.map((stage,index) => `<li class="${index < activeStage ? 'done' : index === activeStage ? 'active' : ''}">${index < activeStage ? iconHTML('check') : `<span>${index + 1}</span>`}<strong>${stage}</strong></li>`).join('')}</ol><p>You can leave this page open. Completed pages are saved safely and the workflow can be resumed.</p></div>`, 'M-Press protects document structure and saves only validated output.');
    };
    const runTranslation = async plan => {
      const label = languageLabel(plan.language);
      const previousReloadSuppression = state.suppressReloadUntil;
      state.suppressReloadUntil = Number.MAX_SAFE_INTEGER;
      try {
        if (!formats().nativeMPD) {
          renderProgress(label, 0, `Converting and validating ${formats().markdown} source documents.`);
          const conversion = await api('translations', {method: 'POST', body: JSON.stringify({action: 'convert-mpd'})});
          updateBuild(conversion.state);
          await loadTranslationData(plan.language, plan.file);
        }
        renderProgress(label, 1, `${plan.report?.pending || 0} passages using ${modelName(selectedStage('draft'))}.`);
        const result = await api('translations', {method: 'POST', body: JSON.stringify(plan)});
        updateBuild(result.state);
		renderProgress(label, 2, 'An independent reviewer is checking meaning, terminology, and structure. Flagged passages are repaired and checked again.');
		const checked = await api('translations', {method: 'POST', body: JSON.stringify({action: 'audit', language: plan.language, file: plan.file, refine: true})});
        renderProgress(label, 3, 'Preparing the result and any passages that still need attention.');
		renderTranslationResult(plan, result, checked);
      } catch (error) {
        setDrawerBody('Translation paused', `<div class="translation-failure">${iconHTML('triangle-alert')}<div><strong>Your completed work is safe</strong><span>${escapeHTML(error.message)}</span></div></div><p class="intro">Fix the problem and continue the same plan. M-Press will reuse completed translations rather than starting over.</p><div class="actions"><button class="button" id="translation-error-advanced" type="button">Advanced settings</button><button class="button primary" id="translation-error-back" type="button">Return to plan</button></div>`, 'The workflow stopped safely. No invalid page was written.');
        q('#translation-error-advanced').addEventListener('click', () => ensureRepositoryReady(() => showConfig(false, 'translation')));
        q('#translation-error-back').addEventListener('click', () => renderTranslationConfirmation(plan));
        toast(error.message, true);
      } finally {
        state.suppressReloadUntil = Math.max(previousReloadSuppression, Date.now() + 1500);
      }
    };
    const renderTranslationResult = (plan, result, audit) => {
	  const translated = (result.report?.files || []).reduce((sum, file) => sum + (file.translated || 0), 0);
	  const refined = Number(audit?.refinement?.segments || 0);
	  if (audit?.audit) audit = audit.audit;
      const clean = !audit.errors && !audit.warnings;
      const findings = audit.findings || [];
	  setDrawerBody(clean ? 'Translation ready for review' : 'Translation needs review', `<div class="translation-result ${clean ? 'clean' : 'attention'}">${iconHTML(clean ? 'check-circle' : 'search')}<div><strong>${clean ? 'Automated checks passed' : `${audit.errors} error${audit.errors === 1 ? '' : 's'} and ${audit.warnings} warning${audit.warnings === 1 ? '' : 's'} remain`}</strong><span>${translated} passage${translated === 1 ? '' : 's'} translated${refined ? ` and ${refined} refined after review` : ''} across ${result.report?.written || 0} written page${result.report?.written === 1 ? '' : 's'}.</span></div></div>${findings.length ? `<div class="translation-findings">${findings.slice(0,20).map(finding => `<article><span class="${finding.severity}">${escapeHTML(finding.severity)}</span><div><strong>${escapeHTML(finding.file)}${finding.segment ? ` · ${escapeHTML(finding.segment)}` : ''}</strong><p>${escapeHTML(finding.message)}</p></div></article>`).join('')}</div>` : '<p class="translation-clean-copy">Code, links, metadata, components, glossary terms, and translated structure are intact.</p>'}<div class="actions"><button class="button" id="translation-overview" type="button">Translation overview</button>${plan.file && clean ? '<button class="button primary" id="translation-review" type="button">Review and approve this page</button>' : '<button class="button primary" id="translation-done" type="button">Done for now</button>'}</div>`, clean ? 'The independent review and automated checks found no remaining structural, terminology, or meaning problems. Read the result before approving it.' : 'Your translation and automatic repairs are saved. Review the remaining passages before approval.');
      q('#translation-overview').addEventListener('click', async () => { await loadTranslationData(plan.language); renderOverview(); });
      q('#translation-review')?.addEventListener('click', () => renderReview(plan.language, plan.file, audit));
      q('#translation-done')?.addEventListener('click', closeDrawer);
    };
    const renderReview = (language, file, audit = null) => {
      const label = languageLabel(language);
      const sourceParts = returnLocation.pathname.split('/').filter(Boolean);
      const firstPartIsLanguage = (data.languages || []).includes(sourceParts[0]);
      if (firstPartIsLanguage) sourceParts.shift();
      const translatedPage = `/${encodeURIComponent(language)}/${sourceParts.map(encodeURIComponent).join('/')}${sourceParts.length ? '/' : ''}`;
      setDrawerBody('Review and approve', `<form id="translation-review-form" class="form-grid"><div class="translation-review-intro settings-wide">${iconHTML('eye')}<div><strong>Read the translated page in the site</strong><span>Check meaning, tone, links, and instructions. M-Press will record the exact source and target versions you approved.</span></div></div><div class="translation-plan-summary settings-wide"><div><span>Language</span><strong>${escapeHTML(label)}</strong></div><div><span>Page</span><strong>${escapeHTML(file)}</strong></div><div><span>Automated checks</span><strong>${audit && !audit.errors ? 'Passed' : 'Review findings first'}</strong></div></div><div class="actions settings-wide translation-review-preview"><a class="button" href="${escapeHTML(translatedPage)}" target="_blank" rel="noopener">${iconHTML('eye')} Open translated page</a></div><label class="check-field settings-wide"><input name="confirmed" type="checkbox" required><span><strong>I have read this translation</strong><small>I confirm that it preserves the source meaning and is ready for contribution.</small></span></label><div class="actions"><button class="button" type="button" data-translation-back>Back</button><button class="button primary" type="submit">Approve this translation</button></div></form>`, 'Human approval is the final step. M-Press keeps it current only while the source and translation remain unchanged.');
      bindBack(() => void returnOverview());
      q('#translation-review-form').addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Recording approval';
        try {
          await api('translations', {method: 'POST', body: JSON.stringify({action: 'mark', language, file, status: 'reviewed'})});
          toast('Translation approved');
          await loadTranslationData(language);
          renderOverview();
        } catch (error) { button.disabled = false; button.textContent = 'Approve this translation'; toast(error.message, true); }
      });
    };
    renderOverview();
  };
  const formatCheckDuration = value => {
    const milliseconds = Math.max(0, Number(value) || 0);
    if (milliseconds < 1) return '<1 ms';
    if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
    if (milliseconds < 10000) return `${(milliseconds / 1000).toFixed(2)} s`;
    return `${(milliseconds / 1000).toFixed(1)} s`;
  };
  const renderCheckPerformance = (performance, collapsed = false) => {
    const steps = Array.isArray(performance?.steps) ? performance.steps : [];
    if (!steps.length) return '';
    const latestEnd = steps.reduce((maximum, step) => Math.max(maximum, (Number(step.offsetMs) || 0) + (Number(step.durationMs) || 0)), 0);
    const total = Math.max(Number(performance.durationMs) || 0, latestEnd, 0.01);
    const started = performance.startedAt ? new Date(performance.startedAt) : null;
    const startedLabel = started && !Number.isNaN(started.valueOf()) ? started.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'}) : 'this run';
    const rows = steps.map(step => {
      const offset = Math.max(0, Number(step.offsetMs) || 0);
      const duration = Math.max(0, Number(step.durationMs) || 0);
      const startPercent = Math.min(100, offset / total * 100);
      const durationPercent = Math.max(0, Math.min(100 - startPercent, duration / total * 100));
      const status = step.status === 'failed' ? 'failed' : 'passed';
      const detail = step.detail ? `${escapeHTML(step.detail)} · ` : '';
      return `<div class="check-timeline-row ${status}"><div class="check-timeline-label"><strong>${escapeHTML(step.label || step.name)}</strong><small>${detail}starts +${formatCheckDuration(offset)}</small></div><div class="check-timeline-track" aria-label="${escapeHTML(step.label || step.name)} took ${formatCheckDuration(duration)}"><span class="check-timeline-bar" style="--check-start:${startPercent.toFixed(3)}%;--check-duration:${durationPercent.toFixed(3)}%"></span></div><span class="check-timeline-duration">${formatCheckDuration(duration)}</span></div>`;
    }).join('');
    const content = `<section class="check-performance" aria-labelledby="check-performance-title"><div class="check-performance-header"><div><h3 id="check-performance-title">Validation performance</h3><p>Started at ${escapeHTML(startedLabel)}. Bars show execution order and elapsed time.</p></div><strong class="check-performance-total">${formatCheckDuration(total)}</strong></div><div class="check-timeline">${rows}</div><div class="check-timeline-axis" aria-hidden="true"><span>Start</span><span>${formatCheckDuration(total)}</span></div></section>`;
    return collapsed ? `<details class="check-performance-details"><summary>View build details</summary>${content}</details>` : content;
  };
  const runChecks = async (showPanel = true, onDone = null, doneLabel = '') => {
    if (showPanel) openDrawer('Project health', 'Run checks', '<div class="result-box"><strong>Checks are running</strong>Please wait for the generated output.</div>', 'tool', 'M-Press rebuilds the site and checks every internal link and asset.');
    q('#quick-check').disabled = true;
    try {
      const result = await api('check', {method: 'POST'});
      updateBuild(result.state);
      q('#check-label').textContent = result.ok ? 'All checks passed' : `${result.broken?.length || 0} issue${result.broken?.length === 1 ? '' : 's'}`;
      if (showPanel) {
        const broken = (result.broken || []).map(item => `<div class="check-item fail">${iconHTML('x')}<div><strong>${escapeHTML(item.File || item.file)}</strong><span>${escapeHTML(item.Href || item.href)} · ${escapeHTML(item.Reason || item.reason)}</span></div></div>`).join('');
        setDrawerBody('Run checks', `<div class="check-summary"><div class="check-item ${result.ok ? 'pass' : 'fail'}">${iconHTML(result.ok ? 'check' : 'x')}<div><strong>${result.ok ? 'Build completed' : 'Checks need attention'}</strong><span>${result.state.pages || 0} pages · ${result.state.files || 0} generated files</span></div></div>${broken}</div>${renderCheckPerformance(result.performance, Boolean(onDone))}<div class="actions"><button class="button primary" id="checks-done" type="button">${escapeHTML(doneLabel || (onDone ? 'Back to guide' : 'Done'))}</button></div>`, result.ok ? 'The generated site is ready to publish.' : 'M-Press found issues that you must fix before deployment.');
        q('#checks-done').addEventListener('click', onDone || closeDrawer);
      }
      toast(result.ok ? 'All checks passed' : 'Checks found issues', !result.ok);
      return result;
    } catch (error) {
      toast(error.message, true);
      if (showPanel) setDrawerBody('Run checks', `<div class="result-box"><strong>Checks could not finish</strong>${escapeHTML(error.message)}</div>`, 'M-Press could not validate the generated site.');
      return null;
    } finally { q('#quick-check').disabled = false; }
  };
  const lighthouseRating = score => score >= 90 ? 'good' : score >= 50 ? 'average' : 'poor';
  const renderLighthouseResult = result => {
    const scores = (result.categories || []).map(category => {
      const score = Math.max(0, Math.min(100, Number(category.score) || 0));
      return `<div class="lighthouse-score ${lighthouseRating(score)}"><div class="lighthouse-ring" style="--score:${score}" aria-label="${escapeHTML(category.title)} score ${score} out of 100"><strong>${score}</strong></div><span>${escapeHTML(category.title)}</span></div>`;
    }).join('');
    const metrics = (result.metrics || []).map(metric => `<div class="lighthouse-metric"><span>${escapeHTML(metric.title)}</span><strong>${escapeHTML(metric.displayValue || 'Not available')}</strong></div>`).join('');
    const diagnostics = (result.diagnostics || []).map(item => `<div class="check-item ${item.score >= 50 ? 'warn' : 'fail'}"><div class="lighthouse-diagnostic-score">${Number(item.score) || 0}</div><div><strong>${escapeHTML(item.title)}</strong>${item.displayValue ? `<span>${escapeHTML(item.displayValue)}</span>` : ''}</div></div>`).join('');
    const fetched = result.fetchedAt ? new Date(result.fetchedAt) : null;
    const fetchedLabel = fetched && !Number.isNaN(fetched.valueOf()) ? fetched.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'}) : 'just now';
    setDrawerBody('Lighthouse results', `<section class="lighthouse-scores" aria-label="Lighthouse scores">${scores}</section>${metrics ? `<section class="lighthouse-section"><h2>Loading metrics</h2><div class="lighthouse-metrics">${metrics}</div></section>` : ''}${diagnostics ? `<section class="lighthouse-section"><h2>Top opportunities</h2><div class="check-summary">${diagnostics}</div></section>` : '<div class="result-box"><strong>No major opportunities found</strong>Lighthouse did not return any low-scoring diagnostics for this page.</div>'}<div class="lighthouse-meta">Lighthouse ${escapeHTML(result.version || '')} · completed in ${formatCheckDuration(result.durationMs)}</div><div class="actions"><button class="button" id="lighthouse-again" type="button">Test again</button><button class="button primary" id="lighthouse-done" type="button">Done</button></div>`, `Audited ${result.url || siteRoute} with the ${result.formFactor} profile at ${fetchedLabel}. Scores can vary slightly between runs.`);
    q('#lighthouse-again').addEventListener('click', showLighthouse);
    q('#lighthouse-done').addEventListener('click', closeDrawer);
  };
  const runLighthouse = async form => {
    const button = form.querySelector('[type="submit"]');
    const formFactor = new FormData(form).get('formFactor') || 'mobile';
    button.disabled = true;
    button.textContent = 'Running Lighthouse';
    q('#lighthouse-progress').innerHTML = '<strong>Testing this page</strong>Lighthouse opens a clean browser session, loads the page and runs its audits. This usually takes 20 to 60 seconds.';
    try {
      const result = await api('lighthouse', {method: 'POST', body: JSON.stringify({path: siteRoute, formFactor})});
      renderLighthouseResult(result);
      toast('Lighthouse audit completed');
    } catch (error) {
      q('#lighthouse-progress').innerHTML = `<strong>Lighthouse could not run</strong>${escapeHTML(error.message)}`;
      button.disabled = false;
      button.textContent = 'Run Lighthouse';
      toast(error.message, true);
    }
  };
  const showLighthouse = () => {
    openDrawer('Site quality', 'Lighthouse audit', `<form id="lighthouse-form" class="form-grid"><fieldset class="lighthouse-profile"><legend>Test profile</legend><label><input type="radio" name="formFactor" value="mobile" checked><span><strong>Mobile</strong><small>Simulated mobile device and connection</small></span></label><label><input type="radio" name="formFactor" value="desktop"><span><strong>Desktop</strong><small>Desktop viewport and scoring profile</small></span></label></fieldset><div class="result-box lighthouse-progress" id="lighthouse-progress"><strong>Current page</strong>${escapeHTML(siteRoute)}</div><div class="actions"><button class="button" id="lighthouse-cancel" type="button">Cancel</button><button class="button primary" type="submit">Run Lighthouse</button></div></form>`, 'tool', 'Test the current page with Google Lighthouse. This optional check does not change the generated site.');
    q('#lighthouse-cancel').addEventListener('click', closeDrawer);
    q('#lighthouse-form').addEventListener('submit', event => { event.preventDefault(); runLighthouse(event.currentTarget); });
  };
  const showExport = () => {
    openDrawer('Export', 'Download this site', `
      <p class="intro">M-Press creates a clean production build and packages every generated page and asset into one portable ZIP file.</p>
      <form id="export-form" class="form-grid">
        <label class="check-field"><input name="strict" type="checkbox" checked><span><strong>Require a strict build</strong><small>Stop if M-Press finds unsupported content.</small></span></label>
        <label class="check-field"><input name="drafts" type="checkbox"><span><strong>Include draft pages</strong><small>Leave this off for a production archive.</small></span></label>
        <div class="actions"><button class="button" id="export-cancel" type="button">Cancel</button><button class="button primary" type="submit">Build and download ZIP</button></div>
      </form>`);
    q('#export-cancel').addEventListener('click', closeDrawer);
    q('#export-form').addEventListener('submit', async event => {
      event.preventDefault();
      const form = event.currentTarget;
      const button = form.querySelector('[type="submit"]');
      button.disabled = true;
      button.textContent = 'Building archive';
      try {
        const headers = new Headers({'Content-Type': 'application/json'});
        if (state.token) headers.set('X-MPress-Token', state.token);
        const response = await fetch('/__mpress/api/export', {method: 'POST', headers, body: JSON.stringify({strict: form.elements.strict.checked, drafts: form.elements.drafts.checked})});
        if (!response.ok) {
          const payload = await response.json().catch(() => ({}));
          throw new Error(payload.error || 'M-Press could not export this site');
        }
        const blob = await response.blob();
        const filename = response.headers.get('X-MPress-Filename') || 'site.zip';
        const downloadURL = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = downloadURL;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        link.remove();
        setTimeout(() => URL.revokeObjectURL(downloadURL), 1000);
        const pages = response.headers.get('X-MPress-Pages') || '0';
        const files = response.headers.get('X-MPress-Files') || '0';
        setDrawerBody('Download ready', `<div class="result-box"><strong>${escapeHTML(filename)}</strong>${escapeHTML(pages)} pages · ${escapeHTML(files)} files · ${escapeHTML(formatBytes(blob.size))}</div><div class="actions"><button class="button primary" id="export-done" type="button">Done</button></div>`, 'The production archive is ready.');
        q('#export-done').addEventListener('click', closeDrawer);
        toast('Production ZIP downloaded');
      } catch (error) {
        button.disabled = false;
        button.textContent = 'Build and download ZIP';
        toast(error.message, true);
      }
    });
  };
  const showDeploy = async (preferredProvider = '') => {
    let data;
    try { data = await api('deployment'); } catch (error) { toast(error.message, true); return; }
    const target = data.target || {};
    const provider = preferredProvider || data.provider || target.provider || 'cloudflare-pages';
    openDrawer('Deployment', 'Publish this site', `
      <div class="deploy-provider-banner" data-deploy-provider-banner>
        <span class="provider-logo provider-logo-cloudflare" data-deploy-provider-logo>${iconHTML('cloudflare')}</span>
        <div><strong data-deploy-provider-name>Cloudflare Pages</strong><small data-deploy-provider-description>Preview deployments do not replace production.</small></div>
        <span class="provider-credential" data-deploy-credential></span>
      </div>
      <p class="intro">Publish the checked static build without a Node.js build step. M-Press saves the target details, never the credential.</p>
      <form id="deploy-form" class="form-grid">
        <label class="field"><span>Provider</span><select name="provider"><option value="cloudflare-pages"${selectedAttr(provider, 'cloudflare-pages')}>Cloudflare Pages</option><option value="netlify"${selectedAttr(provider, 'netlify')}>Netlify</option></select></label>
        <label class="field"><span>Target name</span><input name="target" required value="${escapeHTML(data.name || (provider === 'netlify' ? 'netlify' : 'cloudflare'))}"></label>
        <label class="field"><span data-deploy-account-label>Account or team</span><input name="accountID" value="${escapeHTML(target.accountID)}"><small data-deploy-account-help>Cloudflare account ID or Netlify team slug.</small></label>
        <label class="field"><span data-deploy-project-label>Project or site</span><input name="project" required placeholder="my-documentation" value="${escapeHTML(target.project)}"></label>
        <label class="field"><span>Production branch</span><input name="productionBranch" value="${escapeHTML(target.productionBranch || 'main')}"></label>
        <label class="field"><span>Custom domain</span><input name="domain" placeholder="docs.example.com" value="${escapeHTML(target.domain)}"></label>
        <label class="field"><span>Environment</span><select name="environment"><option value="preview">Preview</option><option value="production">Production</option></select></label>
        <div class="result-box"><strong>Credential</strong><span data-deploy-credential></span></div>
        <div class="actions"><button class="button" id="save-deploy" type="button">Save setup</button><button class="button primary" type="submit">Create preview</button></div>
      </form>`);
    const form = q('#deploy-form');
    const syncProvider = () => {
      const netlify = form.elements.provider.value === 'netlify';
      const providerLogo = q('[data-deploy-provider-logo]');
      providerLogo.className = `provider-logo provider-logo-${netlify ? 'netlify' : 'cloudflare'}`;
      providerLogo.innerHTML = iconHTML(netlify ? 'netlify' : 'cloudflare');
      q('[data-deploy-provider-name]').textContent = netlify ? 'Netlify' : 'Cloudflare Pages';
      q('[data-deploy-provider-description]').textContent = netlify ? 'Atomic previews stay separate from your published site.' : 'Preview deployments stay isolated until you promote them.';
      q('[data-deploy-account-label]').textContent = netlify ? 'Netlify team slug (optional)' : 'Cloudflare account ID';
      q('[data-deploy-account-help]').textContent = netlify ? 'Leave blank for your personal Netlify team.' : 'Find this on the Cloudflare account overview.';
      q('[data-deploy-project-label]').textContent = netlify ? 'Netlify site ID, domain, or name' : 'Cloudflare Pages project';
      form.elements.project.pattern = netlify ? '' : '[a-z0-9-]+';
      form.elements.project.placeholder = netlify ? 'docs.example.com' : 'my-documentation';
      q('[data-deploy-credential]').textContent = netlify
        ? (data.hasNetlifyToken ? 'Netlify access is ready.' : 'Set NETLIFY_AUTH_TOKEN before starting M-Press.')
        : (data.hasCloudflareToken ? 'Cloudflare access is ready.' : 'Set CLOUDFLARE_API_TOKEN before starting M-Press.');
    };
    syncProvider();
    form.elements.provider.addEventListener('change', syncProvider);
    const saveSetup = async () => {
      const values = Object.fromEntries(new FormData(form));
      const saved = await api('deployment', {method: 'PUT', body: JSON.stringify(values)});
      data = saved;
      toast('Deployment setup saved');
      return values;
    };
    q('#save-deploy').addEventListener('click', async event => {
      event.currentTarget.disabled = true;
      try { await saveSetup(); } catch (error) { toast(error.message, true); }
      event.currentTarget.disabled = false;
    });
    form.addEventListener('change', () => {
      const production = form.elements.environment.value === 'production';
      form.querySelector('[type="submit"]').textContent = production ? 'Review production deploy' : 'Create preview';
    });
    form.addEventListener('submit', async event => {
      event.preventDefault();
      const values = await saveSetup().catch(error => { toast(error.message, true); return null; });
      if (!values) return;
      if (values.environment === 'production' && !confirm('Publish this build to production?')) return;
      const button = event.submitter;
      button.disabled = true;
      button.textContent = values.environment === 'production' ? 'Publishing' : 'Creating preview';
      try {
        const result = await api('deployment', {method: 'POST', body: JSON.stringify({target: values.target, environment: values.environment})});
        setDrawerBody('Deployment complete', `<div class="result-box"><strong>${values.environment === 'production' ? 'Production is live' : 'Preview is ready'}</strong><a href="${escapeHTML(result.url)}" target="_blank" rel="noopener">${escapeHTML(result.url)}</a><br>${result.uploaded || 0} files uploaded · ${result.reused || 0} reused</div><div class="actions"><button class="button" id="deploy-close" type="button">Done</button><a class="button primary" href="${escapeHTML(result.url)}" target="_blank" rel="noopener">Open deployment</a></div>`, `The site passed its checks. ${values.provider === 'netlify' ? 'Netlify' : 'Cloudflare'} accepted the deployment.`);
        q('#deploy-close').addEventListener('click', closeDrawer);
        toast('Deployment complete');
      } catch (error) {
        button.disabled = false;
        button.textContent = values.environment === 'production' ? 'Review production deploy' : 'Create preview';
        toast(error.message, true);
      }
    });
  };
  const showPublish = () => {
    openDrawer('Publish', 'Publish', `
      <div class="publish-providers" aria-label="Publish providers">
        <button type="button" id="publish-cloudflare" class="publish-provider-card">
          <span class="provider-logo provider-logo-cloudflare">${iconHTML('cloudflare')}</span>
          <span><strong>Cloudflare Pages</strong><small>Fast previews at the edge, then one explicit production promotion.</small><em>Use Cloudflare Pages</em></span>
        </button>
        <button type="button" id="publish-netlify" class="publish-provider-card">
          <span class="provider-logo provider-logo-netlify">${iconHTML('netlify')}</span>
          <span><strong>Netlify</strong><small>Atomic draft deploys with no build service and no Node.js runtime.</small><em>Use Netlify</em></span>
        </button>
      </div>
      <div class="publish-or" aria-hidden="true"><span>or</span></div>
      <button type="button" id="publish-zip" class="publish-zip-row">
        ${iconHTML('package')}<span><strong>Download a production ZIP</strong><small>Keep a portable archive for any static host or release system.</small></span>${iconHTML('chevron-right')}
      </button>`, 'tool', 'Publish the checked static output to a host, or download it for another static server.');
    q('#publish-zip').addEventListener('click', showExport);
    q('#publish-cloudflare').addEventListener('click', () => ensureRepositoryReady(showDeploy));
    q('#publish-netlify').addEventListener('click', () => ensureRepositoryReady(() => showDeploy('netlify')));
  };
  const repositoryReadyKey = () => `mpress-repository-ready:${state.project?.repository?.path || state.project?.name || 'project'}`;
  const showRepositorySetup = async next => {
    let repository;
    try { repository = await api('setup/repository'); } catch (error) { toast(error.message, true); return; }
    const defaultBranch = repository.onWorkBranch ? repository.branch : 'docs/mpress';
    const defaultRepository = repository.origin || '';
    const parentCheckout = repository.path ? repository.path.replace(/\/+$/, '') : '';
    const localProject = !repository.isGit;
    openDrawer('Setup', 'Prepare a safe checkout', `
      <div class="wizard-page setup-wizard">
        <span class="wizard-kicker">Before you edit</span>
        <h3 class="wizard-title">${localProject ? 'Choose how to work.' : 'Work in your own branch.'}</h3>
        <p class="intro">${localProject ? 'Continue with this new local project, or bring in documentation from a repository.' : 'Use this checkout, download another repository, or fork it on GitHub. M-Press creates a work branch before opening any editing tools.'}</p>
        <div class="setup-modes" role="tablist" aria-label="Repository setup method">
          <button type="button" role="tab" aria-selected="true" class="active" data-setup-mode="current"><strong>${localProject ? 'Local project' : 'Current checkout'}</strong><span>${localProject ? 'Edit this folder without Git' : escapeHTML(repository.branch || 'Detached checkout')}</span></button>
          <button type="button" role="tab" aria-selected="false" data-setup-mode="clone"><strong>Clone</strong><span>Download a repository you can access</span></button>
          <button type="button" role="tab" aria-selected="false" data-setup-mode="fork"${repository.githubCLI ? '' : ' disabled'}><strong>Fork</strong><span>${repository.githubCLI ? 'Create your GitHub fork and clone it' : 'Install GitHub CLI to enable this'}</span></button>
        </div>
        <form id="setup-current-form" class="form-grid setup-panel" data-setup-panel="current">
          <div class="result-box"><strong>${escapeHTML(repository.path)}</strong>${localProject ? 'Changes are saved directly in this folder. Add Git later when you want version history.' : repository.origin ? escapeHTML(repository.origin) : 'This checkout has no origin remote.'}${repository.dirty && !localProject ? '<br>Uncommitted changes will stay with the new branch.' : ''}</div>
          ${localProject ? '' : `<label class="field"><span>Work branch</span><input name="branch" required value="${escapeHTML(defaultBranch)}"><small>Use a short purpose, for example docs/migrate-to-mpress.</small></label>`}
          <div class="actions"><button class="button" type="button" data-setup-cancel>Cancel</button><button class="button primary" type="submit">${localProject ? 'Continue locally' : repository.onWorkBranch ? 'Continue on this branch' : 'Create branch and continue'}</button></div>
        </form>
        <form id="setup-clone-form" class="form-grid setup-panel" data-setup-panel="clone" hidden>
          <label class="field"><span>Repository URL</span><input name="repository" required placeholder="https://github.com/owner/docs.git" value="${escapeHTML(defaultRepository)}"></label>
          <label class="field"><span>Checkout directory</span><input name="destination" required value="${escapeHTML(parentCheckout ? parentCheckout + '-checkout' : '')}"></label>
          <label class="field"><span>Work branch</span><input name="branch" required value="docs/mpress"></label>
          <div class="actions"><button class="button" type="button" data-setup-cancel>Cancel</button><button class="button primary" type="submit">Clone and create branch</button></div>
        </form>
        <form id="setup-fork-form" class="form-grid setup-panel" data-setup-panel="fork" hidden>
          <label class="field"><span>Repository</span><input name="repository" required placeholder="owner/repository" value="${escapeHTML(defaultRepository)}"></label>
          <label class="field"><span>Checkout directory</span><input name="destination" required value="${escapeHTML(parentCheckout ? parentCheckout + '-fork' : '')}"></label>
          <label class="field"><span>Work branch</span><input name="branch" required value="docs/mpress"></label>
          <div class="result-box"><strong>GitHub authentication</strong>M-Press uses your existing gh login. Credentials are never written to the project.</div>
          <div class="actions"><button class="button" type="button" data-setup-cancel>Cancel</button><button class="button primary" type="submit">Fork, clone and create branch</button></div>
        </form>
      </div>`, 'wizard');
    const showMode = mode => {
      root.querySelectorAll('[data-setup-mode]').forEach(button => {
        const active = button.dataset.setupMode === mode;
        button.classList.toggle('active', active);
        button.setAttribute('aria-selected', String(active));
      });
      root.querySelectorAll('[data-setup-panel]').forEach(panel => panel.hidden = panel.dataset.setupPanel !== mode);
      q(`[data-setup-panel="${mode}"] input`)?.focus();
    };
    root.querySelectorAll('[data-setup-mode]').forEach(button => button.addEventListener('click', () => showMode(button.dataset.setupMode)));
    root.querySelectorAll('[data-setup-cancel]').forEach(button => button.addEventListener('click', closeDrawer));
    root.querySelectorAll('[data-setup-panel]').forEach(form => form.addEventListener('submit', async event => {
      event.preventDefault();
      const button = event.submitter;
      const values = Object.fromEntries(new FormData(form));
      const action = form.dataset.setupPanel;
      if (action === 'current' && localProject) {
        sessionStorage.setItem(repositoryReadyKey(), 'true');
        closeDrawer();
        next?.();
        return;
      }
      button.disabled = true;
      button.textContent = action === 'current' ? 'Preparing branch' : action === 'clone' ? 'Cloning repository' : 'Creating fork';
      try {
        const result = await api('setup/repository', {method: 'POST', body: JSON.stringify({action, ...values})});
        sessionStorage.setItem(repositoryReadyKey(), 'true');
        if (state.project) state.project.repository = result.repository;
        if (result.restartRequired) {
          q('#drawer-title').textContent = 'Checkout ready';
          setDrawerBody('Repository prepared', `<div class="wizard-page"><div class="result-box"><strong>Command</strong><code>${escapeHTML(result.command)}</code></div><div class="actions"><button class="button" id="setup-copy-command" type="button">Copy command</button><button class="button primary" id="setup-done" type="button">Done</button></div></div>`, `The fork or clone is ready on ${result.repository.branch}. Run this command once to continue in the new project.`);
          q('#setup-copy-command').addEventListener('click', async () => {
            try { await navigator.clipboard.writeText(result.command); toast('Command copied'); }
            catch { toast('Could not copy the command', true); }
          });
          q('#setup-done').addEventListener('click', closeDrawer);
        } else {
          toast(`Ready on ${result.repository.branch}`);
          closeDrawer();
          next?.();
        }
      } catch (error) {
        button.disabled = false;
        button.textContent = action === 'current' ? 'Create branch and continue' : action === 'clone' ? 'Clone and create branch' : 'Fork, clone and create branch';
        toast(error.message, true);
      }
    }));
  };
  const ensureRepositoryReady = next => {
    if (!state.project?.contribution) {
	  next();
	  return;
	}
    if (state.project?.repository?.onWorkBranch || sessionStorage.getItem(repositoryReadyKey()) === 'true') {
      sessionStorage.setItem(repositoryReadyKey(), 'true');
      next();
    }
    else showRepositorySetup(next);
  };
  const showContributionWizard = () => {
    const contribution = state.project?.contribution;
    if (!contribution) return;
    const source = contribution.sourcePath || 'Choose a documentation page after the site opens.';
    rememberContributionFlow('choose');
    const guideHTML = contributorGuideHTML(state.project?.contributorGuide);
    openDrawer('Contribute', 'What would you like to improve?', `
      <div class="wizard-page contribution-wizard">
        <span class="wizard-kicker">Local contribution checkout</span>
        <h3 class="wizard-title">Choose one outcome.</h3>
        <p class="intro">You are working privately on <strong>${escapeHTML(contribution.branch)}</strong>. M-Press saves changes only in this checkout until you decide to submit them.</p>
        ${guideHTML}
        <div class="translation-goals contribution-goals">
          <button type="button" data-contribution-goal="page"><strong>Improve this page</strong><span>${escapeHTML(source)}</span></button>
          <button type="button" data-contribution-goal="translate"><strong>Translate documentation</strong><span>Add a language, translate this page, or update existing translations.</span></button>
          <button type="button" data-contribution-goal="config"><strong>Improve the site setup</strong><span>Change navigation, accessibility, appearance, contribution settings, or deployment.</span></button>
          <button type="button" data-contribution-goal="checks"><strong>Check the project</strong><span>Build every page and verify local links and assets before you start.</span></button>
        </div>
        <div class="actions"><button class="button" type="button" data-contribution-later>Explore the site first</button></div>
      </div>`, 'wizard');
    root.querySelectorAll('[data-contribution-goal]').forEach(button => button.addEventListener('click', () => {
      const goal = button.dataset.contributionGoal;
      rememberContributionFlow(goal);
      if (goal === 'page') showContributionPage();
      if (goal === 'translate') showTranslations();
      if (goal === 'config') showConfig(false, 'social');
      if (goal === 'checks') runChecks(true, showContributionReview, 'Review changes');
    }));
    q('[data-contribution-later]').addEventListener('click', () => { rememberContributionFlow('explore'); closeDrawer(); });
  };
  const showContributionPage = () => {
    const contribution = state.project?.contribution;
    if (!contribution) return;
    const source = contribution.sourcePath || 'Open a page and restart the contribution command with its URL.';
    rememberContributionFlow('page');
    const guideHTML = contributorGuideHTML(state.project?.contributorGuide, ' before editing');
    openDrawer('Contribute', 'Improve this page', `
      <div class="wizard-page contribution-wizard">
        <span class="wizard-kicker">Markdown source</span>
        <h3 class="wizard-title">Edit this page locally.</h3>
        <p class="intro">Open this file in your editor. M-Press rebuilds the site and reloads this browser whenever you save a valid change.</p>
        ${guideHTML}
        <div class="result-box"><strong>Source file</strong><code>${escapeHTML(source)}</code></div>
        <div class="result-box"><strong>Checkout</strong><code>${escapeHTML(state.project.repository?.path || '')}</code></div>
        <div class="actions"><button class="button" type="button" data-contribution-back>Back</button><button class="button" type="button" data-contribution-copy>Copy file path</button><button class="button" type="button" data-contribution-open>Open in editor</button><button class="button primary" type="button" data-contribution-check>Run checks when finished</button></div>
      </div>`, 'wizard');
    q('[data-contribution-back]').addEventListener('click', showContributionWizard);
    q('[data-contribution-copy]').addEventListener('click', async () => {
      try { await navigator.clipboard.writeText(source); toast('Source path copied'); }
      catch { toast('Could not copy the source path', true); }
    });
    q('[data-contribution-open]').addEventListener('click', async () => {
      try {
        const result = await api('open-file', {method: 'POST', body: JSON.stringify({path: source})});
        toast(`Opened in ${result.editor}`);
      } catch (error) {
        toast(`${error.message}. Use Copy file path instead.`, true);
      }
    });
    q('[data-contribution-check]').addEventListener('click', () => runChecks(true, showContributionReview, 'Review changes'));
  };
  const contributionCommandsHTML = commands => `<div class="contribution-commands"><strong>Run these commands manually</strong><pre>${escapeHTML((commands || []).join('\n'))}</pre><button class="button" type="button" data-contribution-copy-commands>Copy commands</button></div>`;
  const showContributionReview = async suppliedStatus => {
    rememberContributionFlow('review');
    openDrawer('Contribute', 'Review your contribution', '<div class="result-box"><strong>Reading changed files</strong>Please wait.</div>', 'wizard', 'Review exactly what changed before anything leaves this checkout.');
    try {
      const status = suppliedStatus?.files ? suppliedStatus : await api('contribution');
      const files = (status.files || []).map(file => `<li><span>${escapeHTML(file.status)}</span><code>${escapeHTML(file.path)}</code></li>`).join('');
      const diff = status.diff ? renderContributionDiff(status.diff) : '';
      let next = '';
      if (status.pullRequest) {
        next = `<div class="result-box"><strong>Draft pull request created</strong><a href="${escapeHTML(status.pullRequest)}" target="_blank" rel="noopener">${escapeHTML(status.pullRequest)}</a></div><div class="actions"><button class="button primary" type="button" data-contribution-finish>Return to the site</button></div>`;
      } else if ((status.files || []).length) {
        next = `<form class="form-grid" id="contribution-commit-form"><label class="field settings-wide"><span>Commit message</span><input name="message" required maxlength="160" placeholder="docs: explain the installation step"></label><div class="actions"><button class="button" type="button" data-contribution-back-page>Back</button><button class="button primary" type="submit">Commit changes</button></div></form>`;
      } else if (status.committed && !status.pushed) {
        next = `<div class="result-box"><strong>Commit ${escapeHTML(status.commit)} is ready</strong>Your validated change is committed only in this checkout.</div><div class="actions"><button class="button" type="button" data-contribution-back-page>Back</button><button class="button primary" type="button" data-contribution-push>Push contribution branch</button></div>`;
      } else if (status.pushed && status.canOpenPR) {
        next = `<form class="form-grid" id="contribution-pr-form"><div class="result-box settings-wide"><strong>Branch pushed</strong>The contribution branch is ready for a draft pull request.</div><label class="field settings-wide"><span>Pull request title</span><input name="title" required maxlength="200" placeholder="Improve documentation"></label><label class="field settings-wide"><span>Summary</span><textarea name="body" rows="5" placeholder="Explain what changed and why."></textarea></label><div class="actions"><button class="button" type="button" data-contribution-back-page>Back</button><button class="button primary" type="submit">Open draft pull request</button></div></form>`;
      } else if (status.pushed) {
		next = `<div class="result-box"><strong>Contribution branch pushed</strong>The remote does not support automatic GitHub pull requests. The branch is ready in the repository's normal review workflow.</div><div class="actions"><button class="button primary" type="button" data-contribution-finish>Return to the site</button></div>`;
      } else if (status.clean) {
        next = `<div class="result-box"><strong>No documentation changes yet</strong>Edit and save a source file, then return here to review it.</div><div class="actions"><button class="button primary" type="button" data-contribution-back-page>Back to the page</button></div>`;
      } else {
        next = contributionCommandsHTML(status.fallback);
      }
      setDrawerBody('Review your contribution', `<div class="wizard-page contribution-review"><span class="wizard-kicker">${escapeHTML(status.branch || 'Contribution branch')}</span>${files ? `<ul class="contribution-file-list">${files}</ul>` : ''}${diff}${next}</div>`, status.pullRequest ? 'Your draft pull request is ready for review.' : 'Nothing is pushed until you approve each step.');
      q('[data-contribution-back-page]')?.addEventListener('click', showContributionPage);
      q('[data-contribution-finish]')?.addEventListener('click', closeDrawer);
      q('[data-contribution-copy-commands]')?.addEventListener('click', async () => {
        try { await navigator.clipboard.writeText((status.fallback || []).join('\n')); toast('Commands copied'); }
        catch { toast('Could not copy the commands', true); }
      });
      q('#contribution-commit-form')?.addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Committing';
        try {
          const result = await api('contribution', {method: 'POST', body: JSON.stringify({action: 'commit', message: new FormData(event.currentTarget).get('message')})});
          toast('Changes committed');
          showContributionReview(result);
        } catch (error) { button.disabled = false; button.textContent = 'Commit changes'; toast(error.message, true); }
      });
      q('[data-contribution-push]')?.addEventListener('click', async event => {
        const button = event.currentTarget;
        button.disabled = true;
        button.textContent = 'Pushing branch';
        try {
          const result = await api('contribution', {method: 'POST', body: JSON.stringify({action: 'push'})});
          toast('Contribution branch pushed');
          showContributionReview(result);
        } catch (error) {
          setDrawerBody('Push the contribution', `<div class="result-box"><strong>M-Press could not push automatically</strong>${escapeHTML(error.message)}</div>${contributionCommandsHTML(status.fallback)}<div class="actions"><button class="button" type="button" data-contribution-review-again>Try again</button></div>`, 'Your commit remains safe in this local checkout.');
          q('[data-contribution-review-again]').addEventListener('click', showContributionReview);
          q('[data-contribution-copy-commands]').addEventListener('click', async () => { try { await navigator.clipboard.writeText((status.fallback || []).join('\n')); toast('Commands copied'); } catch { toast('Could not copy the commands', true); } });
        }
      });
      q('#contribution-pr-form')?.addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        const values = Object.fromEntries(new FormData(event.currentTarget));
        button.disabled = true;
        button.textContent = 'Opening pull request';
        try {
          const result = await api('contribution', {method: 'POST', body: JSON.stringify({action: 'pull-request', ...values})});
          toast('Draft pull request created');
          showContributionReview(result);
        } catch (error) { button.disabled = false; button.textContent = 'Open draft pull request'; toast(error.message, true); }
      });
    } catch (error) {
      setDrawerBody('Review your contribution', `<div class="result-box"><strong>Could not read the contribution</strong>${escapeHTML(error.message)}</div><div class="actions"><button class="button primary" type="button" data-contribution-back-page>Back to the page</button></div>`, 'Your files have not been changed by this error.');
      q('[data-contribution-back-page]').addEventListener('click', showContributionPage);
    }
  };
  const showKnowledge = async () => {
    openDrawer('Knowledge base', 'MCP knowledge base', '<div class="feature-summary"><strong>Loading knowledge artifacts</strong><span>M-Press is verifying the current build.</span></div>', 'knowledge', 'Inspect the portable agent index and test its retrieval before publication.');
    const replaceKnowledgeContent = html => {
      const container = q('.workspace-page') || q('#drawer-body');
      container.querySelectorAll(':scope > :not(.workspace-page-header)').forEach(item => item.remove());
      container.insertAdjacentHTML('beforeend', html);
    };
    const render = result => {
      if (!result.enabled) {
        replaceKnowledgeContent('<div class="feature-summary"><strong>Knowledge generation is disabled</strong><span>Enable it under Project files, then save the configuration.</span></div>');
        return;
      }
      if (!result.ready) {
        replaceKnowledgeContent(`<div class="feature-summary"><strong>The knowledge artifact is not ready</strong><span>${escapeHTML(result.error || 'Run a build and try again.')}</span></div>`);
        return;
      }
      const results = (result.results || []).map(item => `<article class="content-list-row"><div><strong>${escapeHTML(item.title)}</strong><span>${escapeHTML(item.pageTitle)} · ${escapeHTML(item.language)} · score ${escapeHTML(item.score)}</span><span>${escapeHTML(item.snippet)}</span></div><div class="content-list-actions"><a class="button" href="${escapeHTML(item.url)}" target="_blank" rel="noreferrer">Open citation</a></div></article>`).join('');
      const html = `<div class="feature-summary"><strong>${escapeHTML(result.pages)} pages · ${escapeHTML(result.sections)} sections</strong><span>${escapeHTML((result.languages || []).join(', '))} · ${escapeHTML(result.versions)} version${result.versions === 1 ? '' : 's'} · digest ${escapeHTML(String(result.digest || '').slice(0, 12))}</span></div><form id="knowledge-search-form" class="inline-create-form"><label class="field"><span>Test a reader question</span><input name="query" value="${escapeHTML(result.query || '')}" placeholder="How do I create an application menu?" required></label><div><button class="button primary" type="submit">Search knowledge</button></div></form><div class="content-list" id="knowledge-search-results">${result.query ? (results || '<p class="empty-repeat">No matching sections.</p>') : '<p class="empty-repeat">Enter a realistic question to inspect ranking and citations.</p>'}</div>`;
      replaceKnowledgeContent(html);
      q('#knowledge-search-form')?.addEventListener('submit', async event => {
        event.preventDefault();
        const query = new FormData(event.currentTarget).get('query');
        try { render(await api('knowledge?q=' + encodeURIComponent(query))); }
        catch (error) { toast(error.message, true); }
      }, {once: true});
    };
    try {
      const result = await api('knowledge');
      render(result);
    } catch (error) {
      replaceKnowledgeContent(`<div class="feature-summary"><strong>Knowledge check failed</strong><span>${escapeHTML(error.message)}</span></div>`);
    }
  };
  const wizard = step => {
    state.wizardStep = step;
    state.wizardActive = true;
    sessionStorage.setItem('mpress-wizard-step', String(step));
    const visibleStep = Math.max(1, Math.min(4, step));
    const progress = `<div class="stepper" aria-hidden="true">${[1,2,3,4].map(index => `<span class="${index < visibleStep ? 'done' : index === visibleStep ? 'active' : ''}"></span>`).join('')}</div>`;
    if (step === 0) {
      openDrawer('Project guide', 'Your documentation is ready', `<div class="wizard-page">${progress}<div class="welcome-mark">M</div><div class="wizard-list"><div><strong>1</strong><span><strong>Choose your starting point</strong><br>Keep the tutorial or begin with a minimal site.</span></div><div><strong>2</strong><span><strong>Configure the site</strong><br>Set the identity and visual direction.</span></div><div><strong>3</strong><span><strong>Edit Markdown</strong><br>Make a change and see it reload.</span></div><div><strong>4</strong><span><strong>Validate the output</strong><br>Check every link and generated asset.</span></div></div><div class="actions"><button class="button" id="wizard-later" type="button">Explore first</button><button class="button primary" id="wizard-next" type="button">Start setup</button></div></div>`, 'wizard', 'Four short steps prepare this site for real documentation work.');
      q('#wizard-later').addEventListener('click', () => {
        sessionStorage.removeItem('mpress-wizard-step');
        localStorage.setItem(`mpress-onboarding-dismissed:${state.project?.repository?.path || state.project?.name || 'project'}`, 'true');
        state.wizardActive = false;
        exitWorkspace();
      });
      q('#wizard-next').addEventListener('click', () => wizard(1));
    } else if (step === 1) {
      openDrawer('Project guide', 'Choose your starting point', `<div class="wizard-page">${progress}<span class="wizard-kicker">Step 1 of 4</span><form id="wizard-starter" class="form-grid"><fieldset class="lighthouse-profile"><legend>Starter content</legend><label><input type="radio" name="starter" value="demo" checked><span><strong>Keep the guided tutorial</strong><small>Learn M-Press with working pages and rich components.</small></span></label><label><input type="radio" name="starter" value="empty"><span><strong>Start with a minimal site</strong><small>Keep one welcome page and remove the tutorial content.</small></span></label><label><input type="radio" name="starter" value="import"><span><strong>Import Markdown files</strong><small>Replace the tutorial with your existing top-level pages.</small></span></label></fieldset><label class="field settings-wide" id="wizard-import-files" hidden><span>Markdown files</span><input type="file" name="files" accept=".md,.markdown,text/markdown" multiple><small>Select at least one file. You can organize folders and navigation after setup.</small></label><div class="result-box" id="wizard-import-guidance" hidden><strong>Moving from Starlight?</strong>Finish this setup, then run the Starlight importer against the original project to preserve its complete structure.</div><div class="actions"><button class="button" id="wizard-back" type="button">Back</button><button class="button primary" type="submit">Continue</button></div></form></div>`, 'wizard', 'Choose how much example content you want before you configure the site.');
      q('#wizard-back').addEventListener('click', () => wizard(0));
      q('#wizard-starter').addEventListener('change', event => {
		if (event.target.name !== 'starter') return;
		q('#wizard-import-files').hidden = event.target.value !== 'import';
		q('#wizard-import-guidance').hidden = event.target.value !== 'import';
	  });
      q('#wizard-starter').addEventListener('submit', async event => {
        event.preventDefault();
        const button = event.submitter;
        button.disabled = true;
        button.textContent = 'Preparing site';
        try {
          const starter = new FormData(event.currentTarget).get('starter') || 'demo';
		  const files = Array.from(event.currentTarget.elements.files?.files || []);
		  if (starter === 'import' && !files.length) throw new Error('Choose at least one Markdown file to import.');
		  if (files.some(file => !/\.(?:md|markdown)$/i.test(file.name))) throw new Error('Only .md and .markdown files can be imported.');
		  state.suppressReloadUntil = Date.now() + 15000;
		  const result = await api('onboarding', {method: 'POST', body: JSON.stringify({action: 'starter', starter})});
		  updateBuild(result.state);
		  for (const file of files) {
			const body = new FormData();
			body.append('file', file, file.name);
			body.append('replaceStarter', 'true');
			const imported = await api('import', {method: 'POST', body});
			updateBuild(imported.state);
		  }
          wizard(2);
        } catch (error) {
          button.disabled = false;
          button.textContent = 'Continue';
          toast(error.message, true);
        }
      });
    } else if (step === 2) {
      openDrawer('Project guide', 'Configure the site', `<div class="wizard-page">${progress}<span class="wizard-kicker">Step 2 of 4</span><div class="wizard-list"><div>${iconHTML('globe')}<span><strong>Site identity</strong><br>Name the project and describe what readers will find.</span></div><div>${iconHTML('sun')}<span><strong>Appearance</strong><br>Choose the colour scheme and accessible theme colours.</span></div></div><div class="actions"><button class="button" id="wizard-back" type="button">Back</button><button class="button primary" id="wizard-config" type="button">Open quick setup</button></div></div>`, 'wizard', 'Quick setup contains only the choices needed for a useful first build.');
      q('#wizard-back').addEventListener('click', () => wizard(1));
      q('#wizard-config').addEventListener('click', () => showConfig(true));
    } else if (step === 3) {
      openDrawer('Project guide', 'Edit a Markdown file', `<div class="wizard-page">${progress}<span class="wizard-kicker">Step 3 of 4</span><div class="result-box"><strong>Source file</strong><code id="wizard-source-path">Finding the file</code></div><p class="intro">Open this file, change one sentence, and save it. The browser reloads when the new page is ready. You can also continue now and edit later.</p><div class="actions"><button class="button" id="wizard-back" type="button">Back</button><button class="button" id="wizard-copy" type="button">Copy file path</button><button class="button primary" id="wizard-edited" type="button">Continue to validation</button></div></div>`, 'wizard', 'Try one real change now, or continue and return to editing later.');
      q('#wizard-back').addEventListener('click', () => wizard(2));
      q('#wizard-edited').addEventListener('click', () => wizard(4));
      q('#wizard-copy').addEventListener('click', async () => {
        const source = q('#wizard-source-path').textContent;
        const checkout = state.project?.repository?.path || '';
        try { await navigator.clipboard.writeText(checkout ? checkout.replace(/\/$/, '') + '/' + source : source); toast('File path copied'); }
        catch { toast('Could not copy the file path', true); }
      });
      api('route?url=' + encodeURIComponent(siteRoute)).then(result => { q('#wizard-source-path').textContent = result.path; }).catch(() => { q('#wizard-source-path').textContent = 'Check the content directory'; });
    } else {
      openDrawer('Project guide', 'Validate the generated output', `<div class="wizard-page">${progress}<span class="wizard-kicker">Step 4 of 4</span><div class="wizard-list"><div>${iconHTML('check-circle')}<span><strong>Build every page</strong><br>Verify internal links and generated assets.</span></div></div><div class="actions"><button class="button" id="wizard-back" type="button">Back</button><button class="button" id="wizard-check" type="button">Run checks</button><button class="button primary" id="wizard-finish" type="button">Finish setup</button></div></div>`, 'wizard', 'A successful check means the static output is ready for any host.');
      q('#wizard-back').addEventListener('click', () => wizard(3));
      q('#wizard-check').addEventListener('click', () => runChecks(true, () => wizard(4)));
      q('#wizard-finish').addEventListener('click', async () => { try { await api('onboarding', {method: 'POST', body: JSON.stringify({action: 'complete'})}); sessionStorage.removeItem('mpress-wizard-step'); localStorage.removeItem(`mpress-onboarding-dismissed:${state.project?.repository?.path || state.project?.name || 'project'}`); state.wizardActive = false; toast('Project setup completed'); exitWorkspace(); } catch (error) { toast(error.message, true); } });
    }
  };
  const command = name => {
    if (name === 'page') state.project?.contribution ? showContributionPage() : showCurrentPage();
    if (name === 'contribute') showContributionWizard();
    if (name === 'review') showContributionReview();
    if (name === 'config' || name === 'quick') ensureRepositoryReady(() => showConfig());
    if (name === 'config-blog' || name === 'blog-posts') ensureRepositoryReady(() => showConfig(false, 'blog', 'posts'));
    if (name === 'blog-new') ensureRepositoryReady(() => showConfig(false, 'blog', 'new'));
    if (name === 'blog-preferences') ensureRepositoryReady(() => showConfig(false, 'blog', 'preferences'));
    if (name.startsWith('config-')) {
      const page = name.slice('config-'.length);
      if (workspaceConfigPages.some(item => item.page === page)) ensureRepositoryReady(() => showConfig(false, page));
    }
    if (name === 'checks') state.project?.contribution ? runChecks(true, showContributionReview, 'Review changes') : runChecks(true);
    if (name === 'lighthouse') showLighthouse();
    if (name === 'translations') ensureRepositoryReady(() => { if (state.project?.contribution) rememberContributionFlow('translate'); showTranslations(); });
    if (name === 'knowledge') showKnowledge();
    if (name === 'publish' || name === 'export' || name === 'deploy') showPublish();
    if (name === 'onboarding') ensureRepositoryReady(() => state.project?.contribution ? showContributionWizard() : wizard(resumeWizard ? state.wizardStep : 0));
  };
  fetch('/__mpress/devbar.html').then(response => response.text()).then(async html => {
    root.innerHTML = html;
    prepareWorkspaceDocument();
    syncWorkspaceTop();
    if (workspaceMode) addEventListener('resize', syncWorkspaceTop, {passive: true});
    installDrawerDrag();
    q('#launcher').addEventListener('click', () => {
      if (workspaceMode) exitWorkspace();
      else openWorkspace(state.project?.onboarding ? 'onboarding' : 'page');
    });
    q('#drawer-close').addEventListener('click', () => workspaceMode ? exitWorkspace() : closeDrawer());
    q('#scrim').addEventListener('click', closeDrawer);
    q('#quick-check').addEventListener('click', () => {
	  if (state.project?.contribution) runChecks(true, showContributionReview, 'Review changes');
	  else if (workspaceMode) command('checks');
	  else openWorkspace('checks');
	});
    q('.preview-sizes').addEventListener('click', event => {
      const button = event.target.closest('[data-preview-size]');
      if (button) setPreviewSize(button.dataset.previewSize);
    });
    q('#command-menu').addEventListener('click', event => {
      const item = event.target.closest('[data-command]');
      if (!item) return;
      if (workspaceMode) {
        const nextURL = new URL(location.href);
        nextURL.searchParams.set('tool', item.dataset.command);
        history.replaceState({}, '', nextURL.pathname + nextURL.search);
      }
      command(item.dataset.command);
    });
    root.addEventListener('keydown', event => {
      if (event.key === 'Escape') {
        if (!q('#drawer').hidden) closeDrawer(); else setMenu(false);
        return;
      }
      if (event.key === 'Tab' && !q('#drawer').hidden && q('#drawer').dataset.mode !== 'configuration') {
        const controls = visibleDrawerControls();
        if (!controls.length) return;
        const first = controls[0];
        const last = controls[controls.length - 1];
        if (event.shiftKey && root.activeElement === first) { event.preventDefault(); last.focus(); }
        if (!event.shiftKey && root.activeElement === last) { event.preventDefault(); first.focus(); }
      }
      if ((event.key === 'ArrowDown' || event.key === 'ArrowUp') && state.menu) {
        const items = Array.from(root.querySelectorAll('#command-menu [role="menuitem"]:not([hidden])'));
        const current = items.indexOf(root.activeElement);
        const direction = event.key === 'ArrowDown' ? 1 : -1;
        event.preventDefault();
        items[(current + direction + items.length) % items.length]?.focus();
      }
    });
    addEventListener('click', event => {
      const onboardingTrigger = event.target.closest?.('a.mpress-button[href="#your-first-three-steps"]');
      if (onboardingTrigger) {
        event.preventDefault();
        if (state.project?.authoring) openWorkspace('onboarding');
        return;
      }
      if (!workspaceMode && state.menu && !event.composedPath().includes(host)) setMenu(false);
    });
    try {
      state.project = await api('project');
      if (state.project.token) {
        state.token = state.project.token;
        sessionStorage.setItem('mpress-token', state.token);
      }
      updateBuild(state.project.state);
      if (state.project.authoring) document.querySelectorAll('[data-mpress-contribute]').forEach(item => {
		item.hidden = true;
		item.style.setProperty('display', 'none', 'important');
		item.closest('.contribution-header')?.style.setProperty('display', 'none', 'important');
	  });
      if (workspaceMode) {
		installWorkspaceSidebar();
        document.title = `${state.project.title || state.project.name} project editor · M-Press`;
        q('#command-menu').setAttribute('role', 'navigation');
        q('#command-menu').setAttribute('aria-label', 'Project editor');
        root.querySelectorAll('#command-menu [role="menuitem"]').forEach(item => item.removeAttribute('role'));
        q('#drawer').setAttribute('role', 'main');
        q('#drawer').removeAttribute('aria-modal');
        q('#workspace-back').href = returnURL;
        q('#drawer-close').setAttribute('aria-label', 'Back to site');
      }
      if (!state.project.authoring) {
        q('#check-label').textContent = 'Read-only mode';
        root.querySelectorAll('[data-write]').forEach(item => item.hidden = true);
      } else {
        enhanceBlogImages();
      }
      const contributionKey = state.project.contribution ? `mpress-contribution-wizard:${state.project.contribution.branch}` : '';
      const savedContributionFlow = contributionFlow();
      const shouldOpenContribution = !workspaceMode && state.project.authoring && state.project.contribution && savedContributionFlow !== 'explore' && (contributionRequested || sessionStorage.getItem(contributionKey) !== 'shown' || Boolean(sessionStorage.getItem(contributionFlowKey())));
      if (workspaceMode) {
        const allowedTools = new Set(['page', 'contribute', 'review', 'config', 'quick', 'config-blog', 'blog-new', 'blog-posts', 'blog-preferences', ...workspaceConfigPages.map(item => item.tool), 'checks', 'lighthouse', 'knowledge', 'publish', 'export', 'translations', 'deploy', 'onboarding']);
        command(allowedTools.has(workspaceTool) ? workspaceTool : 'page');
      } else if (shouldOpenContribution) {
        sessionStorage.setItem(contributionKey, 'shown');
		const flow = contributionFlow();
		if (flow === 'translate') showTranslations();
		else if (flow === 'page') showContributionPage();
		else if (flow === 'config') showConfig(false, 'social');
		else if (flow === 'checks') runChecks(true, showContributionReview, 'Review changes');
		else if (flow === 'review') showContributionReview();
		else if (flow !== 'explore') showContributionWizard();
      } else if (state.project.onboarding && state.project.authoring) {
        const dismissed = localStorage.getItem(`mpress-onboarding-dismissed:${state.project.repository?.path || state.project.name || 'project'}`) === 'true';
        if (resumeWizard || !dismissed) openWorkspace('onboarding');
      }
    } catch (error) { q('#status-label').textContent = 'Disconnected'; q('.status').classList.add('error'); }
    const events = new EventSource('/__mpress/events');
    events.addEventListener('state', event => updateBuild(JSON.parse(event.data)));
    events.addEventListener('reload', event => {
      updateBuild(JSON.parse(event.data));
      if (Date.now() < state.suppressReloadUntil) return;
      location.reload();
    });
    events.onopen = () => q('.status').classList.add('ready');
    events.onerror = () => q('.status').classList.remove('ready');
  });
})();
