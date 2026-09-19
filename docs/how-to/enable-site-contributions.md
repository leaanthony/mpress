---
title: Contribute from a published page
description: Move from a published documentation page to a safe local checkout, then submit a reviewed change.
order: 35
---

Site owners can let readers start a contribution from any published page. The
reader does not need to find the source repository or the matching Markdown
file.

## Start from the page

1. Select the edit icon in the documentation navbar.
2. Select **Edit this page locally**, or **Translate documentation** to start
   with a translation.
3. Copy the command selected for your operating system.
4. Run the command in a terminal.

The command includes the current page URL. M-Press uses the generated page
metadata to find the repository, branch, route, and source file.

If M-Press is installed, the command runs it directly. Otherwise, the generated
shell or PowerShell script downloads the matching release and verifies its
checksum before it starts.

@note{type="info" title="The published site does not receive your changes"}
The command creates a local checkout and a private contribution branch. Nothing
is uploaded while you edit or run checks.
@end

## Choose the contribution

The local site opens the editing or translation workflow you selected. Use
**Back** from the editing screen to choose another contribution:

- **Improve this page** opens the exact source file for the published page.
- **Translate documentation** opens the guided translation workflow.
- **Improve the site setup** opens the project configuration.
- **Check the project** validates the current checkout before you edit.

If the project contains a configured contributor guide, M-Press shows it before
you start editing.

## Improve the selected page

M-Press shows the relative source path and the local checkout. Copy the path,
open it in your editor, and save the Markdown file. The development server
rebuilds the site and reloads the browser when the changed page is ready.

If the site has other languages, the editing screen shows **Update this page in
other languages?** when translations need attention. Even a one-line source
change can make a translated passage stale.

1. Select **Review update** beside a language.
2. Keep **Everything that needs attention** to translate only missing or stale
   passages on this page. Leave **Replace passages changed by a person** off.
3. Select **Review the plan** and check the page, passage count, and provider.
4. Select **Start translation**. Read the translated page before approving it.
5. After approval, choose the next language from **Translations of this page**.
   You can also select **Other languages for this page** from the result before
   approval.

Saving a source file never starts translation automatically. Existing human
edits are preserved. If the screen reports missing tracking or migration, resolve
that state first; see [Translation state and review](/translation/#track-freshness-and-human-edits).
You can also leave translations for another contributor and submit the source
change on its own.

Select **Run checks when finished**. M-Press rebuilds every page and validates
internal links and generated assets.

## Submit the contribution

After validation, select **Review changes**. M-Press shows the changed files and
the source diff before it runs any Git command.

1. Write a short commit message and select **Commit changes**.
2. Select **Push contribution branch**.
3. Select **Open draft pull request**.

M-Press first tries the configured origin. If that repository rejects the push
and GitHub CLI is authenticated, M-Press prepares the contributor's fork and
pushes the branch there. If automatic submission is unavailable, the wizard
shows exact commands that the contributor can copy.

## Safety and authentication

The published page contains the repository URL, source branch, page route, and
source path. It does not contain credentials.

Public repositories clone without authentication. Private repositories use the
reader's existing Git credential helper or GitHub CLI login. M-Press refuses to
reuse an unrelated directory. It also refuses a dirty existing checkout unless
that checkout is already on an M-Press contribution branch.

Commits, pushes, and pull requests are separate explicit actions. A validation
run never uploads work.

## Configure contributor instructions

Add contribution settings to `mpress.yaml`:

```yaml
contribution:
  enabled: true
  repository: https://github.com/example/docs.git
  branch: main
  guide: CONTRIBUTING.md
```

The guide path must stay inside the project. If `guide` is not set, M-Press
looks for `CONTRIBUTING.md`, `CONTRIBUTORS.md`, and
`.github/CONTRIBUTING.md`.

You can also configure these values in development mode. Open the M-Press
project editor, select **Navigation and links**, and enable reader
contributions.

## Generated installation scripts

The production build includes `/mpress-contribute.sh` and
`/mpress-contribute.ps1`. Each script contains the configured repository and
branch.

The POSIX script supports Linux and macOS on AMD64 and ARM64. It tries `curl`
and then `wget`, downloads the matching archive from the latest GitHub release,
and verifies it against `checksums.txt`. The Windows script uses PowerShell and
applies the same SHA-256 check.

Both scripts use `mpress` directly when it is already available on the path.

To open the translation workflow directly, pass `--goal translate`. After
saving the generated script locally, run:

```sh
sh mpress-contribute.sh --goal translate
sh mpress-contribute.sh https://docs.example.com/guide/ --goal translate --checkout "../docs translations"
```

On Windows:

```powershell
./mpress-contribute.ps1 --goal translate
```

Options go after the optional page URL. The scripts forward contributor
options such as `--checkout`, `--port`, `--no-open`, and `--draft-file` to
M-Press. The older positional page, draft filename, and goal arguments from
published pages still work. Run the script with `--help` for usage without
downloading anything.

Git must be installed before running the script. Downloaded release files are
temporary and are removed when M-Press exits, including when setup fails.
The contribution checkout remains available for your next session. Provider
configuration and translation review happen in the local translation workflow;
opening it does not start paid translation requests.
