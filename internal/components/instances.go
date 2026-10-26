package components

import "fmt"

// InstanceIDs assigns document-scoped identities to generated component IDs.
//
// A page render owns one InstanceIDs value and passes it through the component
// pipeline. The first occurrence of a generated base ID keeps that ID; later
// occurrences receive a deterministic "-2", "-3", ... suffix chosen against
// every ID already claimed by the render, including generated child references.
// Allocation follows the render traversal (including nested and form passes),
// which is deterministic for a given source but is not a guarantee about
// literal document order. There is deliberately no package-level counter or
// shared renderer state, so independent renders and concurrent pages cannot
// influence each other.
//
// InstanceIDs is not safe for concurrent use. Callers rendering pages
// concurrently must create one value per render.
type InstanceIDs struct {
	claimed  map[string]struct{}
	warnings []string
}

// NewInstanceIDs returns an empty document identity scope.
func NewInstanceIDs() *InstanceIDs {
	return &InstanceIDs{claimed: make(map[string]struct{})}
}

// reserve claims id when it is still free.
func (ids *InstanceIDs) reserve(id string) bool {
	if id == "" {
		return false
	}
	if ids.claimed == nil {
		ids.claimed = make(map[string]struct{})
	}
	if _, exists := ids.claimed[id]; exists {
		return false
	}
	ids.claimed[id] = struct{}{}
	return true
}

// unique returns base when it is free, otherwise the first free "-N" suffix.
// Suffixes are checked against all claimed IDs, not merely repeat counts for
// the same base, so generated IDs cannot collide with authored roots or with
// an earlier component's generated child references.
func (ids *InstanceIDs) unique(base string) string {
	if base == "" {
		return ""
	}
	if ids.reserve(base) {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if ids.reserve(candidate) {
			return candidate
		}
	}
}

// claimGenerated returns a document-unique ID derived from base. The first
// claim keeps base unchanged so single-instance output and standalone
// component rendering remain byte-compatible with earlier releases.
func (ids *InstanceIDs) claimGenerated(base string) string {
	return ids.unique(base)
}

// claimAuthored preserves an authored ID exactly and returns a unique stem for
// the generated references nested inside that instance. A collision with any
// previously claimed ID is reported instead of being silently rewritten.
func (ids *InstanceIDs) claimAuthored(id string) string {
	if id == "" {
		return ""
	}
	_, collides := ids.claimed[id]
	stem := ids.unique(id)
	if collides {
		ids.warnings = append(ids.warnings, fmt.Sprintf(
			`duplicate component id %q: the ID is already used on this page; the authored root is preserved and generated local references use %q`,
			id, stem))
	}
	return stem
}

// takeWarnings returns collision diagnostics collected since the previous
// call. Keeping them on the instance scope means a nested processing pass can
// report them once without package-global state.
func (ids *InstanceIDs) takeWarnings() []string {
	if len(ids.warnings) == 0 {
		return nil
	}
	warnings := ids.warnings
	ids.warnings = nil
	return warnings
}
