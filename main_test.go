package main

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// newMockVault builds a vault under a fresh temp directory and returns its root:
//
//	vault/
//	|_ note_dir/
//	    |_ Blue.md    links to Red.md
//	    |_ Red.md     no links
//	    |_ Yellow.md  dangling link to Green.md
//	    |_ Orange.md  no links
//
// t.TempDir places this under $TMPDIR (/tmp on Linux) and removes it when the
// test finishes, so runs never collide with each other or with a real vault.
func newMockVault(t *testing.T) string {
	t.Helper()

	vaultPath := filepath.Join(t.TempDir(), "vault")
	noteDirPath := filepath.Join(vaultPath, "note_dir")
	if err := os.MkdirAll(noteDirPath, 0755); err != nil {
		t.Fatalf("creating note dir: %v", err)
	}

	notes := map[string]string{
		"Blue.md":   "Blue is next to [[Red]] on the wheel.\n",
		"Red.md":    "Red is a primary colour.\n",
		"Yellow.md": "Yellow mixed with blue makes [[Green]].\n",
		"Orange.md": "Orange is a secondary colour.\n",
	}
	for name, content := range notes {
		if err := os.WriteFile(filepath.Join(noteDirPath, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	return vaultPath
}

// runWithArgs invokes run() with the given command line. run() defines its
// flags on the global flag.CommandLine, which can only be populated once per
// process, so the set flag.CommandLine, which can only be populated once per is swapped for a throwaway one and restored afterwards.
func runWithArgs(t *testing.T, args ...string) {
	t.Helper()

	origArgs, origFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = origArgs, origFlags })

	os.Args = append([]string{"go-obsidian-helper"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	run()
}

// readReport reads one of the generated report files as a sorted list of note
// names. Sorting keeps the assertion independent of scan order.
func readReport(t *testing.T, noteDirPath, name string) []string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(noteDirPath, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	var names []string
	for _, line := range strings.Split(string(content), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	slices.Sort(names)
	return names
}

func TestRunReportsOrphansAndDanglingLinks(t *testing.T) {
	vaultPath := newMockVault(t)
	noteDirPath := filepath.Join(vaultPath, "note_dir")

	runWithArgs(t, "-vaultPath", vaultPath, "-noteDir", "note_dir")

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
			want:    []string{"Orange.md", "Red.md"},
		},
		{
			name:    "dangling",
			outFile: danglingOutFile,
			want:    []string{"Yellow.md"},
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
