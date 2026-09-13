package content

import "unsafe"

// bytesToString creates an immutable string view over data. The caller must
// never mutate data after this conversion. The returned string keeps the
// backing allocation reachable for as long as the string is used.
func bytesToString(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

// stringToBytes creates a read-only byte view over value. The caller must not
// mutate the returned slice. It is used only with APIs that synchronously read
// their input, such as Goldmark and the YAML decoder.
func stringToBytes(value string) []byte {
	if len(value) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(value), len(value))
}
