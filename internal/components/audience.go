package components

import (
	"fmt"
	"html"
	"strings"
)

// Audience renders content blocks that are toggled by an audience role selector.
// All content is rendered at build time (no auth) — visibility is controlled
// client-side via a dropdown. Useful for docs serving multiple audiences
// (admin, developer, user) from a single page.
//
// Usage:
//
//	:::audience{role="admin"}
//	Admin-only content here.
//	:::
//
//	:::audience{role="developer"}
//	Developer content here.
//	:::
//
// An audience selector dropdown is auto-injected at the top of the page
// when audience blocks are detected. The selected role is persisted in
// localStorage so it carries across pages.
type Audience struct {
	Meta    map[string]string
	Content string
}

func (a *Audience) Parse(content string) error {
	a.Content = content
	return nil
}

func (a *Audience) Render() (string, error) {
	role := strings.TrimSpace(a.Meta["role"])
	if role == "" {
		return a.Content, nil // no role = always show
	}

	escaped := html.EscapeString(role)

	return fmt.Sprintf(`<div class="mpress-audience" data-audience="%s">

%s

</div>`, escaped, a.Content), nil
}

// AudienceScript returns the JS that injects the role selector and
// shows/hides audience blocks. Injected once per page when audience
// blocks are detected.
func AudienceScript() string {
	return `<script>
(function(){
  var blocks = document.querySelectorAll('.mpress-audience');
  if (!blocks.length) return;

  // Collect unique roles
  var roleSet = {};
  blocks.forEach(function(b){ roleSet[b.dataset.audience] = true; });
  var roles = Object.keys(roleSet).sort();
  if (!roles.length) return;

  // Read persisted role
  var storageKey = 'mpress-audience-role';
  var saved = '';
  try { saved = localStorage.getItem(storageKey) || ''; } catch(e){}

  // Default to 'all' if no saved role or saved role not in current page's roles
  var current = (saved && (saved === 'all' || roleSet[saved])) ? saved : 'all';

  // Build selector
  var sel = document.createElement('div');
  sel.className = 'mpress-audience-selector';
  sel.setAttribute('role', 'group');
  sel.setAttribute('aria-label', 'Audience filter');

  var label = document.createElement('label');
  label.textContent = 'Audience: ';
  label.className = 'mpress-audience-label';

  var dd = document.createElement('select');
  dd.className = 'mpress-audience-select';
  dd.setAttribute('aria-label', 'Select audience role');

  var optAll = document.createElement('option');
  optAll.value = 'all';
  optAll.textContent = 'All';
  dd.appendChild(optAll);

  roles.forEach(function(r){
    var opt = document.createElement('option');
    opt.value = r;
    opt.textContent = r.charAt(0).toUpperCase() + r.slice(1);
    dd.appendChild(opt);
  });

  dd.value = current;
  label.appendChild(dd);
  sel.appendChild(label);

  // Insert before first audience block
  var article = blocks[0].closest('article') || blocks[0].parentNode;
  article.insertBefore(sel, article.firstChild);

  function apply(role) {
    blocks.forEach(function(b){
      b.style.display = (role === 'all' || b.dataset.audience === role) ? '' : 'none';
    });
    try { localStorage.setItem(storageKey, role); } catch(e){}
  }

  dd.addEventListener('change', function(){ apply(dd.value); });
  apply(current);
})();
</script>`
}
