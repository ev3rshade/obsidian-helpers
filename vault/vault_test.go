package vault

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadTemplateKeys(t *testing.T) {
	cfg := newMockVault(t, nil, map[string]string{
		"daily.md":   "---\ndate: 2026-09-19\nmood:\n---\n",
		"project.md": "---\nstatus: open\nowner: me\n---\n",
		// Skipped: no frontmatter to extract keys from.
		"README.md": "Templates live here.\n",
		// Skipped: loadTemplateKeys does not descend into subdirectories.
		"archive/old.md": "---\nretired: true\n---\n",
	})

	got, err := loadTemplateKeys(cfg.TemplatesPath)
	if err != nil {
		t.Fatalf("loadTemplateKeys() error = %v, want nil", err)
	}

	want := []map[string]bool{
		{"date": true, "mood": true},
		{"status": true, "owner": true},
	}
	if len(got) != len(want) {
		t.Fatalf("loadTemplateKeys() = %v, want %d entries", got, len(want))
	}
	// os.ReadDir sorts its results, so daily.md precedes project.md.
	for i := range want {
		if !maps.Equal(got[i], want[i]) {
			t.Errorf("loadTemplateKeys()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestLoadTemplateKeysErrors(t *testing.T) {
	tests := []struct {
		name string
		dir  func(t *testing.T) string
	}{
		{
			name: "template directory does not exist",
			dir: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "no-such-dir")
			},
		},
		{
			name: "template path is a file rather than a directory",
			dir: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "not-a-dir")
				if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
					t.Fatalf("WriteFile(%q) error = %v", path, err)
				}
				return path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadTemplateKeys(tt.dir(t))
			if err == nil {
				t.Error("loadTemplateKeys() error = nil, want an error")
			}
			if got != nil {
				t.Errorf("loadTemplateKeys() = %v, want nil", got)
			}
		})
	}
}

// ParseVault gives every note its own ID; a regression here would collapse the
// whole index onto note 0 and make every link look resolved.
func TestParseVaultInternsEachNoteOnce(t *testing.T) {
	cfg := newMockVault(t, colourVault, nil)

	v, err := ParseVault(cfg)
	if err != nil {
		t.Fatalf("ParseVault() error = %v", err)
	}

	if len(v.ids) != len(colourVault) {
		t.Errorf("len(ids) = %d, want %d", len(v.ids), len(colourVault))
	}
	if len(v.notes) != len(colourVault) {
		t.Errorf("len(notes) = %d, want %d", len(v.notes), len(colourVault))
	}

	seen := make(map[NoteID]bool)
	for path, id := range v.ids {
		if seen[id] {
			t.Errorf("id %d used for more than one note (%q)", id, path)
		}
		seen[id] = true

		if got := v.notes[id].Title; got != filepath.Base(path) {
			t.Errorf("notes[%d].Title = %q, want %q", id, got, filepath.Base(path))
		}
	}
}

func TestCleanVaultReportsOrphansAndDanglingLinks(t *testing.T) {
	cfg := newMockVault(t, colourVault, nil)
	v := newTestVault(t, cfg, nil)

	if err := CleanVault(cfg, v); err != nil {
		t.Fatalf("CleanVault() error = %v", err)
	}
	report := readReport(t, cfg)

	tests := []struct {
		name    string
		section string
		want    []string
	}{
		{
			// Blue is the only note with a link that resolves. Red and Orange
			// have no links at all; Yellow's single link points at a note that
			// does not exist.
			name:    "orphans",
			section: orphansHeader,
			want:    []string{"[[Orange.md]]", "[[Red.md]]"},
		},
		{
			name:    "dangling",
			section: danglingHeader,
			want:    []string{"[[Yellow.md]]"},
		},
		{
			// No templates were loaded, so every note fails the format check.
			name:    "bad format",
			section: badFormatHeader,
			want:    []string{"[[Blue.md]]", "[[Orange.md]]", "[[Red.md]]", "[[Yellow.md]]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := report[tt.section]; !slices.Equal(got, tt.want) {
				t.Errorf("%s = %v, want %v", tt.section, got, tt.want)
			}
		})
	}
}

func TestCleanVaultReportsBadFormatAgainstTemplates(t *testing.T) {
	cfg := newMockVault(t,
		map[string]string{
			"Conforms.md":      "---\ntitle: T\ntags: [a]\n---\n[[Conforms]]\n",
			"ExtraKeys.md":     "---\ntitle: T\ntags: [a]\nextra: x\n---\n[[ExtraKeys]]\n",
			"MissingKey.md":    "---\ntitle: T\n---\n[[MissingKey]]\n",
			"NoFrontmatter.md": "[[NoFrontmatter]]\n",
		},
		map[string]string{
			"note.md": "---\ntitle:\ntags:\n---\n",
		},
	)

	v, err := ParseVault(cfg)
	if err != nil {
		t.Fatalf("ParseVault() error = %v", err)
	}
	if err := CleanVault(cfg, &v); err != nil {
		t.Fatalf("CleanVault() error = %v", err)
	}

	want := []string{"[[MissingKey.md]]", "[[NoFrontmatter.md]]"}
	if got := readReport(t, cfg)[badFormatHeader]; !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", badFormatHeader, got, want)
	}
}

// Link resolution itself is covered by TestContainsDangling. These cases pin
// what CleanVault does with the answer: which section a note lands in, and
// that it lands there once.
func TestCleanVaultClassifiesNotes(t *testing.T) {
	tests := []struct {
		name       string
		notes      map[string]string
		wantDangle bool
		wantOrphan bool
	}{
		{
			name: "a note is listed once however many links dangle",
			// ContainsDangling returns on the first failure, and CleanVault
			// appends the note once for that single true.
			notes:      map[string]string{"A.md": "[[X]] [[Y]] [[Z]]\n"},
			wantDangle: true,
		},
		{
			name:       "a note with no links is an orphan, not dangling",
			notes:      map[string]string{"A.md": "no links here\n"},
			wantOrphan: true,
		},
		{
			name: "a note whose links are all skipped is in neither section",
			// "+" and "!" links are never resolved, so nothing dangles -- but
			// the note still has links, so the len(links) == 0 orphan check
			// does not fire either.
			notes: map[string]string{"A.md": "[[+tag]] and [[!index]]\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newMockVault(t, tt.notes, nil)
			v := newTestVault(t, cfg, nil)

			if err := CleanVault(cfg, v); err != nil {
				t.Fatalf("CleanVault() error = %v", err)
			}
			report := readReport(t, cfg)

			var wantDangling []string
			if tt.wantDangle {
				wantDangling = []string{"[[A.md]]"}
			}

			if got := report[danglingHeader]; !slices.Equal(got, wantDangling) {
				t.Errorf("%s = %v, want %v", danglingHeader, got, wantDangling)
			}
			if got := report[orphansHeader]; slices.Contains(got, "[[A.md]]") != tt.wantOrphan {
				t.Errorf("%s = %v, want A.md present = %v", orphansHeader, got, tt.wantOrphan)
			}
		})
	}
}

func TestCleanVaultSkipsDirectoriesInTheNoteDir(t *testing.T) {
	cfg := newMockVault(t, map[string]string{
		"A.md":        "no links\n",
		"subdir/B.md": "no links\n",
	}, nil)
	v := newTestVault(t, cfg, nil)

	if err := CleanVault(cfg, v); err != nil {
		t.Fatalf("CleanVault() error = %v", err)
	}

	// CleanVault does not recurse, and the subdirectory itself is not a note.
	want := []string{"[[A.md]]"}
	if got := readReport(t, cfg)[orphansHeader]; !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", orphansHeader, got, want)
	}
}
