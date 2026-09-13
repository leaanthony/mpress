---
title: Markdown forms
description: Build accessible forms from ordinary Markdown inside an M-Press form block.
order: 7
---

M-Press forms are Markdown documents with controls. The form block establishes
the parsing context. Headings, explanations, lists, and controls can be mixed
in the order that makes sense for the person completing the form.

## A complete form

```md
@form[config.save]

# Edit this site

Change the identity and appearance of this documentation site.

## Site identity

Site title*: [text](site.title)

Description: [textarea](site.description){rows=4}

Public URL: [url](site.url)

## Appearance

Colour scheme: [select](theme.mode)
- Use the device setting = system
- Light
- Dark

[ ] Enable accessibility controls (accessibility.enabled)

[Save and rebuild](submit)

@end
```

The label comes first. The control declaration follows the colon. An asterisk
marks a required field. The name in parentheses is the value binding used by
the form action.

## Controls

Use the common HTML input names: `text`, `url`, `email`, `number`, `date`,
`color`, `password`, `search`, `file`, `range`, and `tel`. Use `textarea` for
long text and `select` for a list of choices.

```md
Email address*: [email](contact.email)

Age: [number](profile.age){min=18 max=120}

Biography: [textarea](profile.biography){rows=6}

Search shortcut: [shortcut](search.shortcut){placeholder=Mod+K}
```

In the development settings editor, `shortcut` fields capture the next key
combination instead of accepting free-form text. The editor writes portable
values such as `Mod+K`, supports `None` by pressing Backspace, and reports
conflicts with other M-Press shortcuts before the form is saved.

Options use ordinary Markdown list items. If the visible label and submitted
value differ, separate them with `=`.

```md
Country: [select](profile.country)
- Australia = au
- France = fr
- United Kingdom = gb
```

Checkboxes use familiar task-list notation:

```md
[ ] Enable search (features.search)

[x] Enable accessibility controls (features.accessibility)
```

For a group of choices, use `radio` or `checkboxes` and follow it with a list.

```md
Colour scheme: [radio](theme.mode)
- (x) System = system
- ( ) Light = light
- ( ) Dark = dark
```

## Actions and layout

Use `[Button text](submit)` for the primary form action and
`[Reset](reset)` to restore the browser's initial values. M-Press renders the
form with labels, stable IDs, required state, keyboard focus, and responsive
stacking. Normal Markdown links and task lists outside a form block keep their
usual meaning.

The form can contain normal Markdown and M-Press layout components. This keeps
long settings flows readable instead of turning the source into a separate
schema language.

```md
@form[contact.send]

## Tell us about the project

The more context you provide, the more useful the reply will be.

Project name*: [text](project.name)

What are you building?: [textarea](project.summary){rows=8}

[Send message](submit)

@end
```
