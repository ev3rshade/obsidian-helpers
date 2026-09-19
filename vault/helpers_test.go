package vault

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ev3rshade/go-obsidian-helper/types"
)

// newMockVault builds a vault under a fresh temp directory and returns a
// Config pointing at it:
//
//	vault/
//	|_ note_dir/   <- notes
//	|_ templates/  <- templates
//
// t.TempDir places this under $TMPDIR and removes it when the test finishes,
// so runs never collide with each other or with a real vault. Map keys are
// file names; a key containing a separator creates the parent directories.
func newMockVault(t *testing.T, notes, templates map[string]string) types.Config {
	t.Helper()

	cfg := types.Config{
		VaultPath: filepath.Join(t.TempDir(), "vault"),
	}
	cfg.NotesPath = filepath.Join(cfg.VaultPath, "note_dir")
	cfg.TemplatesPath = filepath.Join(cfg.VaultPath, "templates")

	for dir, files := range map[string]map[string]string{
		cfg.NotesPath:     notes,
		cfg.TemplatesPath: templates,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
		for name, content := range files {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("WriteFile(%q) error = %v", path, err)
			}
		}
	}

	return cfg
}

// colourVault is the shared fixture for the orphan/dangling tests:
//
//	Blue.md    links to Red.md, which exists
//	Red.md     no links
//	Yellow.md  dangling link to Green.md, which does not exist
//	Orange.md  no links
var colourVault = map[string]string{
	"Blue.md":   "Blue is next to [[Red]] on the wheel.\n",
	"Red.md":    "Red is a primary colour.\n",
	"Yellow.md": "Yellow mixed with blue makes [[Green]].\n",
	"Orange.md": "Orange is a secondary colour.\n",
}

// newTestVault builds a Vault over cfg's notes with the given template keys,
// without going through the template directory on disk.
func newTestVault(t *testing.T, cfg types.Config, templateKeys []map[string]bool) *Vault {
	t.Helper()

	v, err := ParseVault(cfg)
	if err != nil {
		t.Fatalf("ParseVault() error = %v", err)
	}
	v.templateKeys = templateKeys
	return &v
}

// Section headers CleanVault writes into the report, in the order it writes
// them.
const (
	badFormatHeader = "-----BADFORMAT-----"
	danglingHeader  = "------DANGLING-----"
	orphansHeader   = "------ORPHANS------"
)

// readReport reads the generated CLEANFILE and returns each section as a
// sorted list of its entries, keyed by section header. Sorting keeps the
// assertion independent of scan order. Entries are returned verbatim,
// including the "[[...]]" wrapping, so the report format is pinned rather than
// parsed away.
func readReport(t *testing.T, cfg types.Config) map[string][]string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(cfg.VaultPath, CLEANFILE))
	if err != nil {
		t.Fatalf("reading %s: %v", CLEANFILE, err)
	}

	sections := make(map[string][]string)
	var current string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case "[[" + badFormatHeader + "]]":
			current = badFormatHeader
			sections[current] = nil
		case "[[" + danglingHeader + "]]":
			current = danglingHeader
			sections[current] = nil
		case "[[" + orphansHeader + "]]":
			current = orphansHeader
			sections[current] = nil
		default:
			sections[current] = append(sections[current], line)
		}
	}

	for _, entries := range sections {
		slices.Sort(entries)
	}
	return sections
}
