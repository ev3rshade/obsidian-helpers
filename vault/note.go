package vault

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"gopkg.in/yaml.v3"
)

var linkPattern = regexp.MustCompile(`\[\[([^]]+)\]\]`)

func ExtractPropsKeys(content string) map[string]bool {
	propsRegex := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
	matches := propsRegex.FindStringSubmatch(content)
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


func UsesTemplate(content string, vault *Vault) bool {
	noteKeys := ExtractPropsKeys(content)
	if noteKeys == nil {
		return false
	}

	for _, required := range vault.templateKeys {
		match := true
		for k := range required {
			if !noteKeys[k] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

func GetLinks(content string) []string {
	matches := linkPattern.FindAllStringSubmatch(string(content), -1)
	
	var links []string
	for _, m := range matches {
		links = append(links, m[1])
	}

	return links
}

func ContainsDangling(vault *Vault, links []string) bool {
	for _, l := range links {

		// ignore tag and index level links
		if strings.Contains(l, "+") || strings.Contains(l, "!") {
			continue
		}

		// strip alias ("Target|Alias") and heading/block refs ("Target#Heading")
		target := l
		if idx := strings.IndexAny(target, "|#"); idx != -1 {
			target = target[:idx]
		}
		target = strings.TrimSpace(target)

		candidate := target
		if filepath.Ext(candidate) == "" {
			candidate += ".md"
		}

		id := vault.ids[candidate]
		if !vault.notes[id].Exists {
			return true
		}
	}

	return false
}

func IsOrphan(filename string) bool {
	return len(filename) > 0
}