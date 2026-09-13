# Interactive components

@audience{role="developer"}
Extend the pipeline in Go.
@end
@audience{role="writer"}
Write plain text documentation.
@end

@if{param="framework" value="plain" default="true"}
Framework-neutral guidance.
@end
@if{param="framework" value="go"}
Guidance for Go users.
@end

@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}
@input{name="seats" type="number" value="3" label="Editors"}
@computed{expr="projects * seats" deps="projects,seats" label="Project seats" format="%d"}

@api-playground{method="GET" path="/search-index.json" baseUrl="http://127.0.0.1:4174"}
Inspect the generated search index.
@end

@variant{name="react"}
Content selected at build time.
@end
