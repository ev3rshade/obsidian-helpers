package vault

import (
	"testing"
)

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
			// unreadable, CleanVault flags every single note as badly formatted.
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
			vault := &Vault{templateKeys: tt.template}

			if got := UsesTemplate(tt.content, vault); got != tt.want {
				t.Errorf("UsesTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsDangling(t *testing.T) {
	// Red.md is the one note on disk; Sub is a directory, not a note.
	cfg := newMockVault(t, map[string]string{
		"Red.md":     "red\n",
		"Sub/Sub.md": "sub\n",
	}, nil)
	vault := newTestVault(t, cfg, nil)

	tests := []struct {
		name  string
		links []string
		want  bool
	}{
		{
			name:  "no links",
			links: nil,
		},
		{
			name:  "link to an existing note",
			links: []string{"Red"},
		},
		{
			name:  "link that already carries the .md extension",
			links: []string{"Red.md"},
		},
		{
			name:  "alias is stripped before resolving",
			links: []string{"Red|the warm one"},
		},
		{
			name:  "heading reference is stripped before resolving",
			links: []string{"Red#Shades"},
		},
		{
			name:  "surrounding whitespace is trimmed",
			links: []string{"  Red  "},
		},
		{
			name:  "link to a missing note",
			links: []string{"Green"},
			want:  true,
		},
		{
			name:  "one resolving link does not excuse a missing one",
			links: []string{"Red", "Green"},
			want:  true,
		},
		{
			name: "a non-md extension is resolved as written",
			// filepath.Ext is non-empty, so ".md" is not appended and the
			// target has to exist under exactly that name.
			links: []string{"Red.txt"},
			want:  true,
		},
		{
			name: "a link to a subdirectory is dangling",
			// "Sub" has no extension, so it resolves to "Sub.md", which the
			// vault does not index even though the directory exists.
			links: []string{"Sub"},
			want:  true,
		},
		{
			name: "a bare heading reference resolves to \".md\"",
			// Stripping at "#" leaves an empty target, so the candidate is the
			// hidden file ".md", which is never there.
			links: []string{"#Shades"},
			want:  true,
		},
		{
			name:  "tag links are skipped",
			links: []string{"+tag"},
		},
		{
			name:  "index links are skipped",
			links: []string{"!index"},
		},
		{
			name: "the skip is a substring test, not a prefix test",
			// Current behavior worth knowing: any target containing "+" or "!"
			// anywhere is exempted, so a genuinely missing note named "C++" is
			// never reported.
			links: []string{"C++"},
		},
		{
			name:  "a skipped link does not mask a real one",
			links: []string{"+tag", "Green"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsDangling(cfg, vault, tt.links); got != tt.want {
				t.Errorf("ContainsDangling(%v) = %v, want %v", tt.links, got, tt.want)
			}
		})
	}
}
