package main

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

// version is set by release builds using -ldflags "-X main.version=...".
var version string

//go:embed version.txt
var sourceVersion string

func cliVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return strings.TrimSpace(sourceVersion)
}
