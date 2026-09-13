
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
