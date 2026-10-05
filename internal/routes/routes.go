// Package routes defines the portable path contract shared by page parsing,
// configuration and output preflight. Routes are logical, unescaped URL paths.
package routes

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrUnsafe = errors.New("unsafe route")

// Normalize removes empty slash components and decodes single URL escapes.
// Validate before cleaning: cleaning a dot segment would hide traversal.
func Normalize(route string) (string, error) {
	parts := strings.Split(route, "/")
	clean := parts[:0]
	for _, part := range parts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			return "", fmt.Errorf("%w %q: invalid URL escape", ErrUnsafe, route)
		}
		if err := Component(decoded); err != nil {
			return "", err
		}
		clean = append(clean, decoded)
	}
	return strings.Join(clean, "/"), nil
}

// Component accepts one unescaped filesystem and URL path component on every
// supported platform. Percent signs are ambiguous after URL decoding; reject
// them rather than allowing repeated decoding to conceal path separators.
func Component(part string) error {
	if part == "" || part == "." || part == ".." || !utf8.ValidString(part) || strings.ContainsAny(part, `/\<>:"|?*#%`) || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
		return fmt.Errorf("%w component %q", ErrUnsafe, part)
	}
	for _, r := range part {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w component %q: control character", ErrUnsafe, part)
		}
	}
	base, _, _ := strings.Cut(strings.ToUpper(part), ".")
	base = strings.TrimRight(base, " ")
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return fmt.Errorf("%w component %q: Windows device name", ErrUnsafe, part)
	}
	if strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT") {
		suffix := []rune(base[3:])
		if len(suffix) == 1 && strings.ContainsRune("123456789¹²³", suffix[0]) {
			return fmt.Errorf("%w component %q: Windows device name", ErrUnsafe, part)
		}
	}
	return nil
}

// Output validates a canonical, relative output filename without allowing the
// normalization used for user-facing slugs to disguise an unsafe write target.
func Output(filename string) error {
	normalized, err := Normalize(filename)
	if err != nil {
		return err
	}
	if filename == "" || normalized != filename {
		return fmt.Errorf("%w output path %q: must be canonical and relative", ErrUnsafe, filename)
	}
	return nil
}
