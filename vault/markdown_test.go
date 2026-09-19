package vault

import (
	"maps"
	"slices"
	"testing"
)

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
			name:    "malformed yaml",
			content: "---\ntitle: [unclosed\n---\n",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPropsKeys(tt.content)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractPropsKeys() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ExtractPropsKeys() = nil, want %v", tt.want)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("ExtractPropsKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "several links in one document",
			content: "Blue sits between [[Red]] and [[Green]].\n",
			want:    []string{"Red", "Green"},
		},
		{
			name:    "alias and heading refs are returned verbatim",
			content: "[[Red|the warm one]] and [[Green#Shades]]\n",
			want:    []string{"Red|the warm one", "Green#Shades"},
		},
		{
			name:    "inner whitespace is kept",
			content: "[[  Red  ]]\n",
			want:    []string{"  Red  "},
		},
		{
			name: "repeats are not de-duplicated",
			// ContainsDangling resolves each one, so a note that links the same
			// missing target twice is still reported once, by that function.
			content: "[[Red]] then [[Red]] again\n",
			want:    []string{"Red", "Red"},
		},
		{
			name:    "no links at all",
			content: "plain body\n",
		},
		{
			name: "an empty link is not matched",
			// The pattern needs at least one character between the brackets.
			content: "[[]]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLinks(tt.content); !slices.Equal(got, tt.want) {
				t.Errorf("GetLinks() = %v, want %v", got, tt.want)
			}
		})
	}
}
