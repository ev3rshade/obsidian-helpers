package vault

import (
	"path/filepath"
	"strings"

	"github.com/ev3rshade/go-obsidian-helper/types"
)

// UsesTemplate reports whether content carries every frontmatter key of at
// least one of the vault's templates.
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

// ContainsDangling reports whether any link points at a note the vault does
// not contain.
func ContainsDangling(cfg types.Config, vault *Vault, links []string) bool {
	for _, l := range links {
		// ignore tag and index level links
		if strings.Contains(l, "+") || strings.Contains(l, "!") {
			continue
		}

		id, ok := vault.ids[filepath.Join(cfg.NotesPath, LinkTarget(l))]
		if !ok || !vault.notes[id].Exists {
			return true
		}
	}

	return false
}
