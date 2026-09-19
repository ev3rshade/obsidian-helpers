package util

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const outFile = "! report.md"

func TestWriteFiles(t *testing.T) {
	tests := []struct {
		name  string
		lists [][]string
		want  string
	}{
		{
			name:  "one entry per line, each wrapped in brackets",
			lists: [][]string{{"Red.md", "Green.md"}},
			want:  "[[Red.md]]\n[[Green.md]]\n\n\n",
		},
		{
			name: "no entries still creates an empty report",
			// CleanVault calls WriteFiles unconditionally, so a clean vault
			// leaves a report holding only the section separators.
			lists: [][]string{nil},
			want:  "\n\n",
		},
		{
			name: "names are written verbatim",
			// Nothing is escaped or trimmed on the way out.
			lists: [][]string{{"! odd [name].md"}},
			want:  "[[! odd [name].md]]\n\n\n",
		},
		{
			name: "each list is separated by a blank line",
			// This is how the three report sections are kept apart inside the
			// single CLEANFILE.
			lists: [][]string{{"A.md"}, {"B.md"}},
			want:  "[[A.md]]\n\n\n[[B.md]]\n\n\n",
		},
		{
			name:  "no lists at all writes an empty file",
			lists: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), outFile)

			if err := WriteFiles(path, tt.lists...); err != nil {
				t.Fatalf("WriteFiles() error = %v, want nil", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", outFile, err)
			}
			if string(got) != tt.want {
				t.Errorf("%s = %q, want %q", outFile, got, tt.want)
			}
		})
	}
}

func TestWriteFilesTruncatesAnExistingReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), outFile)

	if err := WriteFiles(path, []string{"Red.md", "Green.md"}); err != nil {
		t.Fatalf("first WriteFiles() error = %v, want nil", err)
	}
	// O_TRUNC, so the second run replaces the first run's findings instead of
	// appending to them.
	if err := WriteFiles(path, []string{"Blue.md"}); err != nil {
		t.Fatalf("second WriteFiles() error = %v, want nil", err)
	}

	want := "[[Blue.md]]\n\n\n"
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", outFile, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", outFile, got, want)
	}
}

func TestWriteFilesReturnsAnErrorWhenTheReportCannotBeOpened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", outFile)

	if err := WriteFiles(path, []string{"Red.md"}); err == nil {
		t.Error("WriteFiles() error = nil, want an error")
	}
}

func TestReadDirFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.md", "a.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("Mkdir error = %v", err)
	}

	got, err := ReadDirFiles(dir)
	if err != nil {
		t.Fatalf("ReadDirFiles() error = %v", err)
	}

	// Directories are dropped, and os.ReadDir sorts what it returns.
	want := []string{"a.md", "b.md"}
	if !slices.Equal(got, want) {
		t.Errorf("ReadDirFiles() = %v, want %v", got, want)
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	got, err := ReadFile(dir, "a.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if want := "hello\n"; got != want {
		t.Errorf("ReadFile() = %q, want %q", got, want)
	}
}
