package vault

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	linkPattern = regexp.MustCompile(`\[\[([^]]+)\]\]`)
	// TODO: two known misses. The pattern is anchored at byte zero, so a note
	// with a leading blank line reads as having no frontmatter, and the fences
	// are \n-only, so a note saved with CRLF endings does too. Both end up
	// reported as badly formatted.
	propsPattern = regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
)

// ExtractPropsKeys returns the set of frontmatter keys declared at the top of
// content, or nil when content has no frontmatter block.
func ExtractPropsKeys(content string) map[string]bool {
	matches := propsPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal([]byte(matches[1]), &raw); err != nil {
		fmt.Println("Error parsing frontmatter:", err)
		return nil
	}

	keys := make(map[string]bool, len(raw))
	for k := range raw {
		keys[k] = true
	}
	return keys
}

// GetLinks returns the target of every wikilink in content.
func GetLinks(content string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)

	var links []string
	for _, m := range matches {
		links = append(links, m[1])
	}

	return links
}

// LinkTarget strips a wikilink's alias ("Target|Alias") and heading or block
// reference ("Target#Heading"), returning the bare note name with a .md
// extension. A link that is only a heading reference leaves an empty name and
// so resolves to ".md", which never exists.
func LinkTarget(link string) string {
	target := link
	if idx := strings.IndexAny(target, "|#"); idx != -1 {
		target = target[:idx]
	}
	target = strings.TrimSpace(target)

	if filepath.Ext(target) == "" {
		target += ".md"
	}

	return target
}
