# Release component design QA

final result: passed

## Evidence

- Source visual truth: `/home/lea/Downloads/gemini-code-1786507128230.html`
- Source capture: `/var/home/lea/projects/mpress-document-spec/design-qa-release-reference.png`
- Source focused crop: `/var/home/lea/projects/mpress-document-spec/design-qa-release-reference-crop.png`
- Implementation URL: `http://127.0.0.1:4218/?release=platform-final#fixture-components-data`
- Implementation capture: `/var/home/lea/projects/mpress-document-spec/design-qa-release-implementation.png`
- Implementation focused crop: `/var/home/lea/projects/mpress-document-spec/design-qa-release-implementation-crop.png`
- Side-by-side comparison: `/var/home/lea/projects/mpress-document-spec/design-qa-release-comparison.png`
- Browser viewport: source 1280 x 720 CSS pixels; implementation 1265 x 712 CSS pixels; device scale factor 1.
- Focused comparison: the 750 x 512 source component was compared with the 565 x 352 implementation component centred on a 750 x 512 canvas. The difference in component width comes from the fixture viewer's two-column preview, not the release component.
- State: dark theme, latest release, named release, two note groups, two downloadable assets.

## Fidelity review

- Fonts and typography: MPress keeps its existing font stack and weights. The reference hierarchy is retained through the compact release label, strong version and title, restrained date, note headings, and body copy.
- Spacing and layout rhythm: the header, notes, and asset areas follow the reference's compact vertical rhythm. The MPress component uses the existing component radius and spacing tokens.
- Colours and visual tokens: all surfaces, borders, text, muted text, hover states, and accents use MPress theme variables. No GitHub palette was copied.
- Image and icon fidelity: the GitHub avatar and brand mark were deliberately excluded because they are repository-specific. The only added icon is the functional Lucide download icon on asset links. No emoji or decorative imagery remains.
- Copy and content: the fixture exercises latest status, version, release title, release date, ordinary notes, fixes, and assets. The corrupt emoji in the source reference were not reproduced.

## Responsive and interaction checks

- Light and dark themes were rendered and inspected.
- At 390 x 844 CSS pixels, the release header changes to a column, the title wraps, and the component reports no horizontal overflow.
- Asset links remain within the component at the mobile width.
- The theme control was exercised successfully.
- No page-specific browser errors were reported. The browser logged only its standard Electron development CSP warning.

## Comparison history

1. The first MPress redesign was too editorial and did not preserve the selected reference's release-panel structure.
2. The component was rebuilt around the reference's compact panel, inline release identity, conventional notes, divider, and assets region.
3. GitHub-specific author metadata, avatar, colours, and emoji were removed. MPress tokens and Lucide functional icons were applied.
4. The fixture initially used stale Markdown interchange data, so the title, latest state, and assets did not appear. The Markdown fixture was corrected, regenerated, and compared again.
5. The final comparison found no actionable P0, P1, or P2 differences. Remaining differences are intentional MPress product choices.

## Follow-up polish

- Optional asset sizes and file-type labels could be added later if the release metadata model gains structured assets.
