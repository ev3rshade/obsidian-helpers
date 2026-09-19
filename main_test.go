package main

import (
	"flag"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// These tests exercise package-level state (templateKeys) and the process-wide
// flag.CommandLine, so none of them may call t.Parallel.

// newMockVault builds a vault under a fresh temp directory and returns the
// vault root along with the note and template directories inside it:
//
//	vault/
//	|_ note_dir/   <- notes
//	|_ templates/  <- templates
//
// t.TempDir places this under $TMPDIR and removes it when the test finishes,
// so runs never collide with each other or with a real vault. Map keys are
// file names; a key containing a separator creates the parent directories.
func newMockVault(t *testing.T, notes, templates map[string]string) (vaultPath, noteDirPath, templateDirPath string) {
	t.Helper()

	vaultPath = filepath.Join(t.TempDir(), "vault")
	noteDirPath = filepath.Join(vaultPath, "note_dir")
	templateDirPath = filepath.Join(vaultPath, "templates")

	for dir, files := range map[string]map[string]string{
		noteDirPath:     notes,
		templateDirPath: templates,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
		for name, content := range files {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("WriteFile(%q) error = %v", path, err)
			}
		}
	}

	return vaultPath, noteDirPath, templateDirPath
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

// setTemplateKeys installs a template-key set for the duration of one test and
// restores the previous value afterwards. templateKeys is package-level state
// that initTemplateRegexArr only ever appends to, so every test that touches it
// has to put it back.
func setTemplateKeys(t *testing.T, keys []map[string]bool) {
	t.Helper()

	orig := templateKeys
	t.Cleanup(func() { templateKeys = orig })
	templateKeys = keys
}

// runWithArgs invokes run() with the given command line. run() defines its
// flags on the global flag.CommandLine, which can only be populated once per
// process, so a throwaway FlagSet is swapped in and restored afterwards.
// templateKeys is reset too, because run() appends to it via
// initTemplateRegexArr.
func runWithArgs(t *testing.T, args ...string) {
	t.Helper()

	setTemplateKeys(t, nil)

	origArgs, origFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = origArgs, origFlags })

	os.Args = append([]string{"go-obsidian-helper"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	run()
}

// readReport reads one of the generated report files as a sorted list of its
// lines, blank lines dropped. Sorting keeps the assertion independent of scan
// order. Lines are returned verbatim, including the "[[...]]" wrapping, so the
// report format is pinned rather than parsed away.
func readReport(t *testing.T, noteDirPath, name string) []string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(noteDirPath, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	var lines []string
	for _, line := range strings.Split(string(content), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	slices.Sort(lines)
	return lines
}

func TestExtractPropsKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]bool
		wantNil bool
	}{
		{
			name:    "keys of a normal frontmatter block",
			content: "---\ntitle: My Note\ntags: [a, b]\n---\n\nbody\n",
			want:    map[string]bool{"title": true, "tags": true},
		},
		{
			name:    "keys with null values still count",
			content: "---\ntitle:\ntags:\n---\n",
			want:    map[string]bool{"title": true, "tags": true},
		},
		{
			name:    "only the top level of a nested block",
			content: "---\nmeta:\n  author: me\ntitle: My Note\n---\n",
			want:    map[string]bool{"meta": true, "title": true},
		},
		{
			name:    "non-string keys are stringified",
			content: "---\n1: one\ntrue: yes\n---\n",
			want:    map[string]bool{"1": true, "true": true},
		},
		{
			name:    "empty frontmatter yields an empty, non-nil set",
			content: "---\n\n---\nbody\n",
			want:    map[string]bool{},
		},
		{
			name: "only the first block is read",
			// The regex is non-greedy, so a second "---" fence later in the
			// document (a horizontal rule, say) is ignored.
			content: "---\na: 1\n---\nbody\n---\nb: 2\n---\n",
			want:    map[string]bool{"a": true},
		},
		{
			name:    "no frontmatter",
			content: "# Heading\n\nplain body\n",
			wantNil: true,
		},
		{
			name:    "empty content",
			content: "",
			wantNil: true,
		},
		{
			name: "adjacent fences with nothing between them",
			// "---\n---" has no newline-terminated body, so the regex needs a
			// third fence to match and finds none.
			content: "---\n---\n",
			wantNil: true,
		},
		{
			name: "frontmatter must start at byte zero",
			// Documents current behavior: the regex is anchored with ^ and no
			// (?m) flag, so a leading blank line hides the block entirely.
			content: "\n---\ntitle: My Note\n---\n",
			wantNil: true,
		},
		{
			name: "crlf line endings are not recognised",
			// A real limitation: the fence pattern is \n-only, so a note saved
			// with Windows line endings looks like it has no frontmatter.
			content: "---\r\ntitle: My Note\r\n---\r\n",
			wantNil: true,
		},
		{
			name:    "malformed yaml",
			content: "---\ntitle: [unclosed\n---\n",
			wantNil: true,
		},
		{
			name: "duplicate keys are a yaml error",
			// yaml.v3 rejects duplicate mapping keys outright, so the whole
			// block is discarded rather than de-duplicated.
			content: "---\na: 1\na: 2\n---\n",
			wantNil: true,
		},
		{
			name:    "frontmatter that is a sequence, not a mapping",
			content: "---\n- a\n- b\n---\n",
			wantNil: true,
		},
		{
			name:    "frontmatter that is a bare scalar",
			content: "---\nhello\n---\n",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropsKeys(tt.content)

			if tt.wantNil {
				if got != nil {
					t.Errorf("extractPropsKeys() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("extractPropsKeys() = nil, want %v", tt.want)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("extractPropsKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsesTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template []map[string]bool
		content  string
		want     bool
	}{
		{
			name:     "note carrying exactly the template's keys",
			template: []map[string]bool{{"title": true, "tags": true}},
			content:  "---\ntitle: My Note\ntags: [a]\n---\n",
			want:     true,
		},
		{
			name: "extra keys are allowed",
			// Matching is by subset: the template's keys must all be present,
			// but the note may carry more.
			template: []map[string]bool{{"title": true}},
			content:  "---\ntitle: My Note\ntags: [a]\nextra: x\n---\n",
			want:     true,
		},
		{
			name:     "one missing key disqualifies the note",
			template: []map[string]bool{{"title": true, "tags": true}},
			content:  "---\ntitle: My Note\n---\n",
			want:     false,
		},
		{
			name:     "any one of several templates may match",
			template: []map[string]bool{{"tags": true, "due": true}, {"title": true}},
			content:  "---\ntitle: My Note\n---\n",
			want:     true,
		},
		{
			name:     "note without frontmatter matches nothing",
			template: []map[string]bool{{"title": true}},
			content:  "plain body\n",
			want:     false,
		},
		{
			name: "no templates loaded means nothing matches",
			// Worth knowing: when the template directory is empty or
			// unreadable, run() flags every single note as badly formatted.
			template: nil,
			content:  "---\ntitle: My Note\n---\n",
			want:     false,
		},
		{
			name: "a template with empty frontmatter matches every note",
			// Current behavior: an empty key set is vacuously a subset of any
			// note's keys, so one such template makes the check a no-op for
			// every note that has frontmatter at all.
			template: []map[string]bool{{}},
			content:  "---\nanything: 1\n---\n",
			want:     true,
		},
		{
			name: "a template with empty frontmatter still misses notes without any",
			// ...but a note with no frontmatter at all returns early, before
			// the template loop, so it is still reported.
			template: []map[string]bool{{}},
			content:  "plain body\n",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTemplateKeys(t, tt.template)

			if got := usesTemplate(tt.content); got != tt.want {
				t.Errorf("usesTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitTemplateRegexArr(t *testing.T) {
	setTemplateKeys(t, nil)

	_, _, templateDirPath := newMockVault(t, nil, map[string]string{
		"daily.md":   "---\ndate: 2026-09-19\nmood:\n---\n",
		"project.md": "---\nstatus: open\nowner: me\n---\n",
		// Skipped: no frontmatter to extract keys from.
		"README.md": "Templates live here.\n",
		// Skipped: initTemplateRegexArr does not descend into subdirectories.
		"archive/old.md": "---\nretired: true\n---\n",
	})

	if err := initTemplateRegexArr(templateDirPath); err != nil {
		t.Fatalf("initTemplateRegexArr() error = %v, want nil", err)
	}

	want := []map[string]bool{
		{"date": true, "mood": true},
		{"status": true, "owner": true},
	}
	if len(templateKeys) != len(want) {
		t.Fatalf("templateKeys = %v, want %d entries", templateKeys, len(want))
	}
	// os.ReadDir sorts its results, so daily.md precedes project.md.
	for i := range want {
		if !maps.Equal(templateKeys[i], want[i]) {
			t.Errorf("templateKeys[%d] = %v, want %v", i, templateKeys[i], want[i])
		}
	}
}

func TestInitTemplateRegexArrErrors(t *testing.T) {
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
				if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
					t.Fatalf("WriteFile(%q) error = %v", path, err)
				}
				return path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTemplateKeys(t, nil)

			if err := initTemplateRegexArr(tt.dir(t)); err == nil {
				t.Error("initTemplateRegexArr() error = nil, want an error")
			}
			if len(templateKeys) != 0 {
				t.Errorf("templateKeys = %v, want it left empty", templateKeys)
			}
		})
	}
}

func TestRunReportsOrphansAndDanglingLinks(t *testing.T) {
	vaultPath, noteDirPath, templateDirPath := newMockVault(t, colourVault, nil)

	runWithArgs(t, "-vaultPath", vaultPath, "-noteDir", "note_dir", "-templateDir", templateDirPath)

	tests := []struct {
		name    string
		outFile string
		want    []string
	}{
		{
			// Blue is the only note with a link that resolves. Red and Orange
			// have no links at all; Yellow's single link points at a note that
			// does not exist.
			name:    "orphans",
			outFile: orphansOutFile,
			want:    []string{"[[Orange.md]]", "[[Red.md]]"},
		},
		{
			name:    "dangling",
			outFile: danglingOutFile,
			want:    []string{"[[Yellow.md]]"},
		},
		{
			// No templates were loaded, so every note fails the format check.
			name:    "bad format",
			outFile: badFormatOutFile,
			want:    []string{"[[Blue.md]]", "[[Orange.md]]", "[[Red.md]]", "[[Yellow.md]]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readReport(t, noteDirPath, tt.outFile)
			if !slices.Equal(got, tt.want) {
				t.Errorf("%s = %v, want %v", tt.outFile, got, tt.want)
			}
		})
	}
}

func TestRunReportsBadFormatAgainstTemplates(t *testing.T) {
	vaultPath, noteDirPath, templateDirPath := newMockVault(t,
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

	runWithArgs(t, "-vaultPath", vaultPath, "-noteDir", "note_dir", "-templateDir", templateDirPath)

	want := []string{"[[MissingKey.md]]", "[[NoFrontmatter.md]]"}
	if got := readReport(t, noteDirPath, badFormatOutFile); !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", badFormatOutFile, got, want)
	}
}

func TestRunLinkResolution(t *testing.T) {
	tests := []struct {
		name       string
		notes      map[string]string
		wantDangle bool
		wantOrphan bool
	}{
		{
			name:       "plain link to an existing note",
			notes:      map[string]string{"A.md": "[[B]]\n", "B.md": "b\n"},
			wantDangle: false,
		},
		{
			name:       "link with an explicit .md extension",
			notes:      map[string]string{"A.md": "[[B.md]]\n", "B.md": "b\n"},
			wantDangle: false,
		},
		{
			name:       "alias is stripped before resolving",
			notes:      map[string]string{"A.md": "[[B|the other note]]\n", "B.md": "b\n"},
			wantDangle: false,
		},
		{
			name:       "heading reference is stripped before resolving",
			notes:      map[string]string{"A.md": "[[B#Some Heading]]\n", "B.md": "b\n"},
			wantDangle: false,
		},
		{
			name:       "surrounding whitespace is trimmed",
			notes:      map[string]string{"A.md": "[[  B  ]]\n", "B.md": "b\n"},
			wantDangle: false,
		},
		{
			name:       "link to a missing note",
			notes:      map[string]string{"A.md": "[[Nope]]\n"},
			wantDangle: true,
		},
		{
			name:       "one resolving link does not excuse a missing one",
			notes:      map[string]string{"A.md": "[[B]] and [[Nope]]\n", "B.md": "b\n"},
			wantDangle: true,
		},
		{
			name: "a note is listed once however many links dangle",
			// run() breaks out of the link loop on the first failure.
			notes:      map[string]string{"A.md": "[[X]] [[Y]] [[Z]]\n"},
			wantDangle: true,
		},
		{
			name:       "a note with no links is an orphan, not dangling",
			notes:      map[string]string{"A.md": "no links here\n"},
			wantOrphan: true,
		},
		{
			name: "an empty link is not matched",
			// The pattern requires at least one character between the
			// brackets, so "[[]]" yields no link and A.md reads as an orphan.
			notes:      map[string]string{"A.md": "[[]]\n"},
			wantOrphan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultPath, noteDirPath, templateDirPath := newMockVault(t, tt.notes, nil)

			runWithArgs(t, "-vaultPath", vaultPath, "-noteDir", "note_dir", "-templateDir", templateDirPath)

			var wantDangling []string
			if tt.wantDangle {
				wantDangling = []string{"[[A.md]]"}
			}

			if got := readReport(t, noteDirPath, danglingOutFile); !slices.Equal(got, wantDangling) {
				t.Errorf("%s = %v, want %v", danglingOutFile, got, wantDangling)
			}
			// B.md, where present, has no links of its own and would also be
			// an orphan, so only check A.md's membership.
			got := readReport(t, noteDirPath, orphansOutFile)
			if slices.Contains(got, "[[A.md]]") != tt.wantOrphan {
				t.Errorf("%s = %v, want A.md present = %v", orphansOutFile, got, tt.wantOrphan)
			}
		})
	}
}

func TestRunSkipsDirectoriesInTheNoteDir(t *testing.T) {
	vaultPath, noteDirPath, templateDirPath := newMockVault(t, map[string]string{
		"A.md":        "no links\n",
		"subdir/B.md": "no links\n",
	}, nil)

	runWithArgs(t, "-vaultPath", vaultPath, "-noteDir", "note_dir", "-templateDir", templateDirPath)

	// run() does not recurse, and the subdirectory itself is not a note.
	want := []string{"[[A.md]]"}
	if got := readReport(t, noteDirPath, orphansOutFile); !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", orphansOutFile, got, want)
	}
}