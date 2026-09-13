package components

import (
	"fmt"
	"html"
	"strings"
)

// Conditional renders content only when a URL parameter or localStorage value
// matches. Client-side rendering via a small inline script.
//
// Usage in Markdown:
//
//	:::if{param="framework" value="react"}
//	React-specific content here.
//	:::
//
//	:::if{param="framework" value="vue"}
//	Vue-specific content here.
//	:::
//
// The `param` is read from: 1) URL query string (?framework=react)
// 2) localStorage (mpress-param-framework). The first match wins.
// If no value is set, blocks with `default="true"` are shown.
type Conditional struct {
	Meta    map[string]string
	Content string
}

func (c *Conditional) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *Conditional) Render() (string, error) {
	param := html.EscapeString(c.Meta["param"])
	value := html.EscapeString(c.Meta["value"])
	isDefault := c.Meta["default"] == "true"

	if param == "" || value == "" {
		return c.Content, nil // no param/value = always show
	}

	// Generate a unique ID for this block
	blockID := fmt.Sprintf("mpress-if-%s-%s", param, strings.ReplaceAll(value, " ", "-"))

	defaultAttr := ""
	if isDefault {
		defaultAttr = ` data-default="true"`
	}

	return fmt.Sprintf(`<div class="mpress-conditional" data-param="%s" data-value="%s"%s id="%s">

%s

</div>`, param, value, defaultAttr, blockID, c.Content), nil
}

// ConditionalScript returns the JS that shows/hides conditional blocks
// based on URL params and localStorage. Injected once per page.
func ConditionalScript() string {
	return `<script>
(function(){
  var blocks=document.querySelectorAll('.mpress-conditional[data-param]');
  if(!blocks.length)return;
  var params=new URLSearchParams(location.search);
  var groups={};
  blocks.forEach(function(el){
    var p=el.dataset.param,v=el.dataset.value;
    if(!groups[p])groups[p]=[];
    groups[p].push(el);
    // Save to localStorage if set via URL
    var urlVal=params.get(p);
    if(urlVal)localStorage.setItem('mpress-param-'+p,urlVal);
  });
  // Show matching blocks
  Object.keys(groups).forEach(function(p){
    var val=params.get(p)||localStorage.getItem('mpress-param-'+p)||'';
    var shown=false;
    groups[p].forEach(function(el){
      if(el.dataset.value===val){el.style.display='';shown=true;}
    });
    // If nothing matched, show default blocks
    if(!shown){
      groups[p].forEach(function(el){
        if(el.dataset.default==='true')el.style.display='';
      });
    }
  });
})();
</script>`
}

func init() {
	Registry["if"] = func(m map[string]string) Component { return &Conditional{Meta: m} }
}
