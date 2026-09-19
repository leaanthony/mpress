package translate

import (
	"strings"
	"testing"
)

func TestMigrationPreservesVerifiedParagraphsAcrossInsertion(t *testing.T) {
	for _, file := range []string{"guide.mpd", "guide.md"} {
		t.Run(file, func(t *testing.T) {
			source := []byte("# Guide\n\nFirst sentence.\n\nSecond sentence.\n\nLast sentence.\n")
			target := []byte("# Manuel\n\nPremière phrase.\n\nDeuxième phrase.\n\nDernière phrase.\n")
			extractor := "mpd-v1"
			if file == "guide.md" {
				extractor = "markdown-text-v1"
			}
			oldSource, err := extractHistorical(file, "_nav.yaml", source, extractor)
			if err != nil {
				t.Fatal(err)
			}
			oldTarget, err := extractHistorical(file, "_nav.yaml", target, extractor)
			if err != nil {
				t.Fatal(err)
			}
			state := newFileState(file, "en", "fr", "")
			state.Extractor = extractor
			previous := map[string]SegmentState{}
			statuses := []string{"reviewed", "final", "manual", "machine-translated"}
			for i, segment := range oldSource.Segments {
				text := oldTarget.Segments[i].Original
				entry := SegmentState{SourceHash: segment.SourceHash, TargetHash: Hash(text), MachineHash: Hash(text), MachineText: text, Status: statuses[i], SourcePreview: preview(segment.Original), Provider: "historical-provider", Model: "historical-model", PromptVersion: "historical-prompt", UpdatedAt: "2025-01-01T00:00:00Z"}
				state.Segments[segment.ID] = entry
				previous[segment.Original] = entry
			}
			convertedSource, err := convertMigrationDocument(file, "guide.md", source, nil)
			if err != nil {
				t.Fatal(err)
			}
			convertedTarget, err := convertMigrationDocument(file, "guide.md", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			current := []byte(strings.Replace(string(convertedSource), "Second sentence.", "Inserted sentence.\n\nSecond sentence.", 1))
			translated := []byte(strings.Replace(string(convertedTarget), "Deuxième phrase.", "Phrase ajoutée.\n\nDeuxième phrase.", 1))
			result, review, err := migrateDocuments(migrationPair{file, "_nav.yaml", source, target}, migrationPair{"guide.md", "_nav.yaml", current, translated}, state, "en", nil)
			if err != nil {
				t.Fatal(err)
			}
			if review != 1 {
				t.Fatalf("expected only insertion to need review: %d, %#v", review, result.Migration.ReviewReasons)
			}
			doc, err := ExtractMarkdown(current)
			if err != nil {
				t.Fatal(err)
			}
			for _, segment := range doc.Segments {
				entry := result.Segments[segment.ID]
				if segment.Original == "Inserted sentence." {
					if entry.Status != "migration-review" || entry.Provider != "" || entry.UpdatedAt != "" || result.Migration.ReviewReasons[segment.ID] != "inserted-content" {
						t.Fatalf("new paragraph acquired historical approval: %#v", entry)
					}
					continue
				}
				if entry != previous[segment.Original] {
					t.Fatalf("changed historical state for %s: got %#v, want %#v", segment.Original, entry, previous[segment.Original])
				}
			}
		})
	}
}

func TestParagraphInsertionAlignmentRequiresUnambiguousPairedEvidence(t *testing.T) {
	for _, test := range []struct {
		name, source, target, current, translated string
		accepted                                  bool
	}{
		{"start", "One.\n\nTwo.\n", "Un.\n\nDeux.\n", "New.\n\nOne.\n\nTwo.\n", "Nouveau.\n\nUn.\n\nDeux.\n", true},
		{"end", "One.\n\nTwo.\n", "Un.\n\nDeux.\n", "One.\n\nTwo.\n\nNew.\n", "Un.\n\nDeux.\n\nNouveau.\n", true},
		{"target disambiguates repeated source", "Same.\n", "Avant.\n", "Same.\n\nSame.\n", "Avant.\n\nAprès.\n", true},
		{"duplicate pair", "Same.\n", "Même.\n", "Same.\n\nSame.\n", "Même.\n\nMême.\n", false},
		{"changed target", "One.\n", "Un.\n", "One.\n\nNew.\n", "Modifié.\n\nNouveau.\n", false},
		{"changed source", "One.\n", "Un.\n", "Changed.\n\nNew.\n", "Un.\n\nNouveau.\n", false},
		{"different insertion positions", "One.\n", "Un.\n", "One.\n\nNew.\n", "Nouveau.\n\nUn.\n", false},
		{"added heading", "One.\n", "Un.\n", "One.\n\n# New\n", "Un.\n\n# Nouveau\n", false},
		{"added list", "One.\n", "Un.\n", "One.\n\n- New.\n", "Un.\n\n- Nouveau.\n", false},
		{"deleted prose", "One.\n\nTwo.\n", "Un.\n\nDeux.\n", "One.\n", "Un.\n", false},
		{"multiple insertions", "One.\n", "Un.\n", "New.\n\nOne.\n\nOther.\n", "Nouveau.\n\nUn.\n\nAutre.\n", false},
		{"changed code", "One.\n\n```go\noriginal()\n```\n", "Un.\n\n```go\noriginal()\n```\n", "One.\n\nNew.\n\n```go\nchanged()\n```\n", "Un.\n\nNouveau.\n\n```go\nchanged()\n```\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			parse := func(text string) *Document {
				t.Helper()
				doc, err := ExtractMarkdown([]byte(text))
				if err != nil {
					t.Fatal(err)
				}
				return doc
			}
			aligned := insertedParagraphAlignment(parse(test.source), parse(test.target), parse(test.current), parse(test.translated))
			if (aligned != nil) != test.accepted {
				t.Fatalf("unexpected alignment: %#v", aligned)
			}
		})
	}
}
