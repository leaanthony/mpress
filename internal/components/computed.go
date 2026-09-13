package components

import (
	"fmt"
	"html"
	"strings"
)

// Input renders a labeled input field that feeds reactive computed blocks.
// Values are stored in a shared reactive store and trigger recomputation.
//
// Usage:
//
//	:::input{name="price" type="number" value="10" label="Unit price ($)"}
//	:::
//
//	:::input{name="quantity" type="range" min="1" max="100" value="5" label="Quantity"}
//	:::
//
// Supported types: number, range, text, select.
// For select, list options in the body (one per line).
type Input struct {
	Meta    map[string]string
	Content string
}

func (inp *Input) Parse(content string) error {
	inp.Content = strings.TrimSpace(content)
	return nil
}

func (inp *Input) Render() (string, error) {
	name := inp.Meta["name"]
	if name == "" {
		return "", fmt.Errorf("input component requires a name attribute")
	}

	inputType := inp.Meta["type"]
	if inputType == "" {
		inputType = "number"
	}

	value := inp.Meta["value"]
	label := inp.Meta["label"]
	if label == "" {
		label = name
	}

	escaped := html.EscapeString(name)
	escapedLabel := html.EscapeString(label)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<div class="mpress-reactive-input" data-reactive-name="%s">`, escaped))
	b.WriteString(fmt.Sprintf(`<label class="mpress-reactive-label">%s`, escapedLabel))

	switch inputType {
	case "select":
		b.WriteString(fmt.Sprintf(`<select class="mpress-reactive-control" data-reactive="%s">`, escaped))
		for _, line := range strings.Split(inp.Content, "\n") {
			opt := strings.TrimSpace(line)
			if opt == "" {
				continue
			}
			selected := ""
			if opt == value {
				selected = ` selected`
			}
			b.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, html.EscapeString(opt), selected, html.EscapeString(opt)))
		}
		b.WriteString(`</select>`)
	case "range":
		min := inp.Meta["min"]
		if min == "" {
			min = "0"
		}
		max := inp.Meta["max"]
		if max == "" {
			max = "100"
		}
		if value == "" {
			value = min
		}
		b.WriteString(fmt.Sprintf(`<span class="mpress-reactive-range-wrap"><input type="range" class="mpress-reactive-control" data-reactive="%s" min="%s" max="%s" value="%s"><span class="mpress-reactive-range-value" data-reactive-display="%s">%s</span></span>`,
			escaped, html.EscapeString(min), html.EscapeString(max), html.EscapeString(value), escaped, html.EscapeString(value)))
	default: // number, text
		if value == "" {
			value = "0"
		}
		b.WriteString(fmt.Sprintf(`<input type="%s" class="mpress-reactive-control" data-reactive="%s" value="%s">`,
			html.EscapeString(inputType), escaped, html.EscapeString(value)))
	}

	b.WriteString(`</label></div>`)
	return b.String(), nil
}

// Computed renders a reactive output that evaluates a JS expression
// whenever its dependency inputs change.
//
// Usage:
//
//	:::computed{expr="price * quantity" deps="price,quantity" label="Total"}
//	:::
//
//	:::computed{expr="price * quantity * 1.2" deps="price,quantity" format="$%.2f"}
//	:::
//
// The expr is evaluated as a JS expression with input names as variables.
// The format attribute uses printf-style formatting (applied client-side).
type Computed struct {
	Meta    map[string]string
	Content string
}

func (c *Computed) Parse(content string) error {
	c.Content = strings.TrimSpace(content)
	return nil
}

func (c *Computed) Render() (string, error) {
	expr := c.Meta["expr"]
	if expr == "" {
		return "", fmt.Errorf("computed component requires an expr attribute")
	}

	deps := c.Meta["deps"]
	label := c.Meta["label"]
	format := c.Meta["format"]

	var b strings.Builder
	b.WriteString(`<span class="mpress-reactive-computed"`)
	b.WriteString(fmt.Sprintf(` data-reactive-expr="%s"`, html.EscapeString(expr)))
	if deps != "" {
		b.WriteString(fmt.Sprintf(` data-reactive-deps="%s"`, html.EscapeString(deps)))
	}
	if format != "" {
		b.WriteString(fmt.Sprintf(` data-reactive-format="%s"`, html.EscapeString(format)))
	}
	b.WriteString(`>`)
	if label != "" {
		b.WriteString(fmt.Sprintf(`<span class="mpress-reactive-computed-label">%s: </span>`, html.EscapeString(label)))
	}
	b.WriteString(`<span class="mpress-reactive-computed-value">…</span>`)
	b.WriteString(`</span>`)

	return b.String(), nil
}

// ReactiveScript returns the JS that connects input components to computed blocks.
// Injected once per page when reactive components are detected.
func ReactiveScript() string {
	return `<script>
(function(){
  var inputs=document.querySelectorAll('[data-reactive]');
  var outputs=document.querySelectorAll('.mpress-reactive-computed');
  if(!inputs.length||!outputs.length) return;

  var store={};

  // Initialize store from input values
  inputs.forEach(function(el){
    var name=el.dataset.reactive;
    store[name]=el.type==='range'||el.type==='number'?parseFloat(el.value)||0:el.value;
  });

  function evaluate(){
    outputs.forEach(function(el){
      var expr=el.dataset.reactiveExpr;
      var fmt=el.dataset.reactiveFormat||'';
      var valEl=el.querySelector('.mpress-reactive-computed-value');
      if(!valEl) return;
      try{
        // Build function args from store keys
        var keys=Object.keys(store);
        var vals=keys.map(function(k){return store[k];});
        var fn=new Function(keys.join(','),'return '+expr);
        var result=fn.apply(null,vals);
        if(fmt){
          // Simple printf: %d for int, %.Nf for fixed decimal, $%.2f etc.
          var m=fmt.match(/%\.?(\d*)([df])/);
          if(m){
            var prefix=fmt.substring(0,fmt.indexOf('%'));
            var suffix=fmt.substring(fmt.indexOf(m[0])+m[0].length);
            if(m[2]==='f'&&m[1]){
              result=prefix+parseFloat(result).toFixed(parseInt(m[1]))+suffix;
            }else{
              result=prefix+Math.round(result)+suffix;
            }
          }else{
            result=String(result);
          }
        }else{
          result=typeof result==='number'&&result%1!==0?result.toFixed(2):String(result);
        }
        valEl.textContent=result;
      }catch(e){
        valEl.textContent='Error';
      }
    });
  }

  // Listen for input changes
  inputs.forEach(function(el){
    var name=el.dataset.reactive;
    el.addEventListener('input',function(){
      store[name]=el.type==='range'||el.type==='number'?parseFloat(el.value)||0:el.value;
      // Update range display
      var disp=document.querySelector('[data-reactive-display="'+name+'"]');
      if(disp) disp.textContent=el.value;
      evaluate();
    });
  });

  // Initial evaluation
  evaluate();
})();
</script>`
}
