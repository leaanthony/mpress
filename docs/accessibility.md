---
title: Offer reader accessibility controls
description: Enable private reading, focus, motion, contrast, link, and colour settings for every reader.
translationKey: accessibility-controls
order: 7
---

M-Press includes the accessibility menu in the default theme. It is
enabled by default. Each reader can adapt the site without changing the source
or affecting another visitor.

## Enable the menu

New projects already contain this configuration:

```yaml
accessibility:
  enabled: true
  shortcut: Mod+A
```

Add the same setting to an existing project. Then run the development server:

```sh
mpress dev
```

Open the person icon in the navigation bar. The panel groups settings into
Reading, Focus, and Vision tabs. Press Command+A on an Apple device or Ctrl+A
on another platform to open the panel without leaving the keyboard. The
shortcut does not replace Select All while an editable field has focus.

## Reading settings

Readers can:

- increase the main text size;
- use a simple, widely spaced font stack;
- add line, word, and letter spacing;
- emphasise the start of longer words with bionic reading.

These changes apply to the main content. They do not make the navigation or
controls too large to operate.

## Focus settings

Readers can:

- dim navigation until they point to it;
- follow the pointer with a horizontal reading guide;
- stop non-essential animation and smooth scrolling.

The generated theme also honours the browser's reduced-motion and
reduced-transparency preferences.

## Vision settings

Readers can:

- increase text and border contrast;
- underline links so colour is not the only visual signal;
- select a red and green distinction profile;
- select a blue and yellow distinction profile;
- reduce colour saturation.

The theme supports forced-colour mode and the browser's increased-contrast
preference. Light and dark modes receive suitable colour values.

## Privacy and persistence

M-Press stores the reader's choices in local browser storage. It restores the
visual settings before the main stylesheet renders. This avoids a flash of the
wrong text size, contrast, or colour profile.

The generated site does not send accessibility choices to M-Press or another
service. The **Reset settings** button removes every saved choice.

## Test the result

Check the site with a keyboard and at a mobile width. Confirm that:

1. the accessibility button has the name **Accessibility settings**;
2. every tab and control can receive keyboard focus;
3. the selected settings remain after a reload;
4. **Reset settings** restores the default theme;
5. the page remains readable in forced-colour and reduced-motion modes.

Use **Lighthouse audit** in the development menu for an additional automated
accessibility check. Automated checks do not replace keyboard and assistive
technology testing.

Set `accessibility.enabled: false` only when the project must supply a different
accessibility interface. This removes the button, panel, rules, and script from
the generated site.
