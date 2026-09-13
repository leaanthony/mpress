#!/usr/bin/env bash
set -euo pipefail

destination="${1:?usage: collect-release-licenses.sh DESTINATION}"
mkdir -p "$destination"
index="$destination/INDEX.txt"
printf 'Third-party licenses included with M-Press\n\n' > "$index"

while IFS='|' read -r module version directory; do
  if [[ -z "$module" || "$module" == "github.com/leaanthony/mpress" ]]; then
    continue
  fi
  mapfile -t candidates < <(find "$directory" -maxdepth 2 -type f \( -iname 'LICENSE*' -o -iname 'COPYING*' -o -iname 'NOTICE*' \) | sort)
  if [[ ${#candidates[@]} -eq 0 ]]; then
    printf 'No license file found for %s %s\n' "$module" "$version" >&2
    exit 1
  fi
  for license in "${candidates[@]}"; do
    relative="${license#"$directory"/}"
    filename="$(printf '%s-%s' "$module" "$relative" | sed 's#[^A-Za-z0-9._-]#_#g')"
    cp "$license" "$destination/$filename"
    printf '%s %s (%s) -> %s\n' "$module" "$version" "$relative" "$filename" >> "$index"
  done
done < <(go list -deps -f '{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}' ./cmd/mpress | sort -u)

cp internal/icons/vendor/LICENSE "$destination/astro-starlight-icons-LICENSE"
printf '%s\n' 'Astro Starlight icon paths -> astro-starlight-icons-LICENSE' >> "$index"
