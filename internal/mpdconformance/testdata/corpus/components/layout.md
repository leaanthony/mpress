# Layout components

@section{variant="hero"}
@columns{variant="hero"}
@column{variant="hero-copy"}
@headline
Modern docs.
Plain text.
@end

Build rich documentation without a runtime.

@actions
@button{href="/start/" variant="primary"}
Start
@end
@button{href="/guide/" variant="secondary"}
Read the guide
@end
@end
@end
@column
@docs-preview{eyebrow="Documentation" title="Create a project" description="One command creates the site."}
$ mpress new docs
@end
@end
@end
@end

@container{display="grid" columns="2" gap="1rem"}
@callout{title="Static" icon="zap"}
The complete layout exists in generated HTML.
@end
@callout{title="Responsive" icon="monitor"}
Dense layouts collapse on small screens.
@end
@end

@cards{cols="2"}
[Start a project](/start/)
Install one binary.
---
[Configure the site](/configure/)
Use one YAML file.
@end

@preview-tabs
[Markdown]
Write plain text.

[Output]
Publish static HTML.
@end

@file-tabs
[index.md]
```markdown
# Hello world
```

[mpress.yaml]
```yaml
site:
  title: Hello world
```

[Output]
Static HTML.
@end

@carousel{label="Accessibility options"}
@note{type="info" title="Reading"}
Increase text size and choose a comfortable content width.
@end
---
@note{type="tip" title="Focus"}
Reduce motion and hide distracting interface elements.
@end
@end

@accessibility-demo{title="Reader controls"}
@end

@timeline
1. **Write** Create the source.
2. **Build** Generate the site.
@end

@capabilities
- **Accessible** Use semantic output.
- **Fast** Build in milliseconds.
@end

@resources
[Tutorial](/tutorial/)
### Build a site
Learn by doing.
---
[Reference](/reference/)
### See every option
Look up details.
@end
