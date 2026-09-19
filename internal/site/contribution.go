package site

import "strings"

const contributionRepositoryMarker = "__MPRESS_CONTRIBUTION_REPOSITORY__"
const contributionBranchMarker = "__MPRESS_CONTRIBUTION_BRANCH__"

const contributionInstallShellTemplate = `#!/bin/sh
set -eu

case "${1-}" in
  -h|--help)
    echo 'Usage: mpress-contribute.sh [PAGE-OR-REPOSITORY [DRAFT-FILE [GOAL]]] [contribute options]'
    echo 'Translate: sh mpress-contribute.sh --goal translate'
    echo 'Options: --goal page|translate, --checkout DIRECTORY, --port PORT, --no-open, --draft-file FILE'
    exit 0 ;;
esac

# Keep the positional handoff used by published sites, then forward CLI options.
target=
draft_file=
goal=
case "${1-}" in -*) ;; *) target=${1-}; if [ "$#" -gt 0 ]; then shift; fi ;; esac
case "${1-}" in -*) ;; *) draft_file=${1-}; if [ "$#" -gt 0 ]; then shift; fi ;; esac
case "${1-}" in -*) ;; *) goal=${1-}; if [ "$#" -gt 0 ]; then shift; fi ;; esac
if [ -z "$target" ]; then
  target=` + contributionRepositoryMarker + `
fi

run_contribution() {
  mpress_binary=$1
  shift
  if [ -n "$draft_file" ]; then
    set -- --draft-file "$draft_file" "$@"
  fi
  if [ -n "$goal" ]; then
    set -- --goal "$goal" "$@"
  fi
  "$mpress_binary" contribute --branch ` + contributionBranchMarker + ` "$target" "$@"
}

if ! command -v git >/dev/null 2>&1; then
  echo 'Git is required to prepare a contribution checkout. Install Git and run this command again.' >&2
  exit 1
fi

if command -v mpress >/dev/null 2>&1; then
  run_contribution mpress "$@"
  exit $?
fi

release_base=${MPRESS_RELEASE_BASE_URL:-https://github.com/leaanthony/mpress/releases/latest/download}
case "$release_base" in
  https://*) ;;
  *) echo "MPRESS_RELEASE_BASE_URL must use HTTPS." >&2; exit 1 ;;
esac

download() {
  source_url=$1
  destination=$2
  if command -v curl >/dev/null 2>&1; then
    if curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error --retry 3 --retry-delay 1 --output "$destination" "$source_url"; then
      return 0
    fi
  fi
  if command -v wget >/dev/null 2>&1; then
    if wget --https-only --tries=3 --quiet --output-document="$destination" "$source_url"; then
      return 0
    fi
  fi
  echo "Could not download $source_url. Install curl or wget and try again." >&2
  return 1
}

system=$(uname -s | tr '[:upper:]' '[:lower:]')
machine=$(uname -m)
case "$system" in
  darwin) system=darwin ;;
  linux) system=linux ;;
  *) echo "M-Press does not provide a release for $(uname -s)." >&2; exit 1 ;;
esac
case "$machine" in
  x86_64|amd64) machine=amd64 ;;
  arm64|aarch64) machine=arm64 ;;
  *) echo "M-Press does not provide a release for architecture $machine." >&2; exit 1 ;;
esac

if ! command -v tar >/dev/null 2>&1; then
  echo "The tar command is required to unpack M-Press." >&2
  exit 1
fi

temporary_directory=$(mktemp -d 2>/dev/null || mktemp -d -t mpress)
cleanup() { rm -rf "$temporary_directory"; }
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

asset="mpress-${system}-${machine}.tar.gz"
archive="$temporary_directory/$asset"
checksums="$temporary_directory/checksums.txt"
download "$release_base/$asset" "$archive"
download "$release_base/checksums.txt" "$checksums"

expected=$(awk -v asset="$asset" '$2 == asset || $2 == "*" asset { print $1; exit }' "$checksums")
if [ -z "$expected" ]; then
  echo "The M-Press release does not contain a checksum for $asset." >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$archive" | awk '{ print $1 }')
else
  echo "A SHA-256 tool is required. Install sha256sum or shasum and try again." >&2
  exit 1
fi
if [ "$actual" != "$expected" ]; then
  echo "Checksum verification failed for $asset." >&2
  exit 1
fi

tar -xzf "$archive" -C "$temporary_directory"
binary="$temporary_directory/mpress"
if [ ! -f "$binary" ]; then
  echo "The M-Press release archive does not contain the mpress executable." >&2
  exit 1
fi
chmod +x "$binary"
run_contribution "$binary" "$@"
`

const contributionInstallPowerShellTemplate = `$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

if ($args.Count -gt 0 -and $args[0] -in @('-h', '--help')) {
  Write-Output 'Usage: mpress-contribute.ps1 [PAGE-OR-REPOSITORY [DRAFT-FILE [GOAL]]] [contribute options]'
  Write-Output 'Translate: ./mpress-contribute.ps1 --goal translate'
  Write-Output 'Options: --goal page|translate, --checkout DIRECTORY, --port PORT, --no-open, --draft-file FILE'
  exit 0
}

$positional = @('', '', '')
$argumentOffset = 0
while ($argumentOffset -lt $args.Count -and $argumentOffset -lt 3 -and -not ([string]$args[$argumentOffset]).StartsWith('-')) {
  $positional[$argumentOffset] = [string]$args[$argumentOffset]
  $argumentOffset++
}
$target = if ($positional[0]) { $positional[0] } else { ` + contributionRepositoryMarker + ` }
$draftFile = $positional[1]
$goal = $positional[2]
$extraArguments = @($args | Select-Object -Skip $argumentOffset)

function Start-Contribution([string] $Binary) {
  $contributionArguments = @("contribute", "--branch", ` + contributionBranchMarker + `, $target)
  if ($draftFile) { $contributionArguments += @("--draft-file", $draftFile) }
  if ($goal) { $contributionArguments += @("--goal", $goal) }
  $contributionArguments += $extraArguments
  & $Binary @contributionArguments
  exit $LASTEXITCODE
}

if (-not (Get-Command git -CommandType Application -ErrorAction SilentlyContinue)) {
  throw 'Git is required to prepare a contribution checkout. Install Git and run this command again.'
}

$installedMPress = Get-Command mpress -CommandType Application -ErrorAction SilentlyContinue
if ($installedMPress) {
  Start-Contribution $installedMPress.Source
}

if ([Net.ServicePointManager]::SecurityProtocol -band [Net.SecurityProtocolType]::Tls12 -eq 0) {
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}

$releaseBase = if ($env:MPRESS_RELEASE_BASE_URL) { $env:MPRESS_RELEASE_BASE_URL.TrimEnd('/') } else { "https://github.com/leaanthony/mpress/releases/latest/download" }
if (([uri]$releaseBase).Scheme -ne "https") { throw "MPRESS_RELEASE_BASE_URL must use HTTPS." }
$architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
  "X64" { "amd64" }
  "Arm64" { "arm64" }
  default { throw "M-Press does not provide a Windows release for architecture $($_)." }
}

function Download-ReleaseAsset([string] $Source, [string] $Destination) {
  $lastError = $null
  for ($attempt = 1; $attempt -le 3; $attempt++) {
    try {
      Invoke-WebRequest -UseBasicParsing -Uri $Source -OutFile $Destination
      return
    } catch {
      $lastError = $_
      if ($attempt -lt 3) { Start-Sleep -Seconds $attempt }
    }
  }
  throw "Could not download $Source after three attempts. $lastError"
}

$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("mpress-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
try {
  $asset = "mpress-windows-$architecture.zip"
  $archive = Join-Path $temporaryDirectory $asset
  $checksums = Join-Path $temporaryDirectory "checksums.txt"
  Download-ReleaseAsset "$releaseBase/$asset" $archive
  Download-ReleaseAsset "$releaseBase/checksums.txt" $checksums

  $escapedAsset = [regex]::Escape($asset)
  $checksumLine = Get-Content -LiteralPath $checksums | Where-Object { $_ -match "^([A-Fa-f0-9]{64})\s+\*?$escapedAsset$" } | Select-Object -First 1
  if (-not $checksumLine) { throw "The M-Press release does not contain a checksum for $asset." }
  $expected = ($checksumLine -split "\s+")[0].ToLowerInvariant()
  $actual = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actual -ne $expected) { throw "Checksum verification failed for $asset." }

  Expand-Archive -LiteralPath $archive -DestinationPath $temporaryDirectory -Force
  $binary = Join-Path $temporaryDirectory "mpress.exe"
  if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw "The M-Press release archive does not contain mpress.exe."
  }
  Start-Contribution $binary
} finally {
  Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
`

func renderContributionInstallShell(repository, branch string) string {
	script := strings.ReplaceAll(contributionInstallShellTemplate, contributionRepositoryMarker, shellSingleQuote(repository))
	return strings.ReplaceAll(script, contributionBranchMarker, shellSingleQuote(branch))
}

func renderContributionInstallPowerShell(repository, branch string) string {
	script := strings.ReplaceAll(contributionInstallPowerShellTemplate, contributionRepositoryMarker, powerShellSingleQuote(repository))
	return strings.ReplaceAll(script, contributionBranchMarker, powerShellSingleQuote(branch))
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
