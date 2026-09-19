// Package vault models an Obsidian vault and the notes inside it.
package vault

import (
	"fmt"
	"path/filepath"

	"github.com/ev3rshade/go-obsidian-helper/types"
	"github.com/ev3rshade/go-obsidian-helper/util"
)

// CLEANFILE is the report written at the root of the vault.
const CLEANFILE = "!! CLEANFILE.md"

type NoteID int32

type Note struct {
	Path    string
	Title   string
	Aliases []string
	Exists  bool
}

type Vault struct {
	path         string
	ids          map[string]NoteID
	notes        []Note
	templateKeys []map[string]bool
	out          [][]NoteID
	in           [][]NoteID
}

// intern returns the ID for path, allocating one the first time path is seen.
func (vault *Vault) intern(path string) NoteID {
	if id, ok := vault.ids[path]; ok {
		return id
	}

	id := NoteID(len(vault.notes))
	vault.ids[path] = id
	vault.notes = append(vault.notes, Note{})
	vault.out = append(vault.out, nil)
	vault.in = append(vault.in, nil)
	return id
}

// ParseVault reads the templates and every note named by cfg into a Vault.
func ParseVault(cfg types.Config) (Vault, error) {
	vault := Vault{
		path: cfg.VaultPath,
		ids:  make(map[string]NoteID),
	}

	var err error
	vault.templateKeys, err = loadTemplateKeys(cfg.TemplatesPath)
	if err != nil {
		fmt.Println("Error reading in templates:", err)
		return Vault{}, err
	}

	names, err := util.ReadDirFiles(cfg.NotesPath)
	if err != nil {
		fmt.Println("Error opening vault directory:", err)
		return Vault{}, err
	}

	for _, name := range names {
		fPath := filepath.Join(cfg.NotesPath, name)
		currID := vault.intern(fPath)

		vault.notes[currID] = Note{
			Path:    fPath,
			Title:   name,
			Aliases: nil,
			Exists:  true,
		}
	}

	return vault, nil
}

// loadTemplateKeys returns the frontmatter key set of every template in
// templateDir.
func loadTemplateKeys(templateDir string) ([]map[string]bool, error) {
	names, err := util.ReadDirFiles(templateDir)
	if err != nil {
		return nil, err
	}

	var templateKeys []map[string]bool
	for _, name := range names {
		content, err := util.ReadFile(templateDir, name)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return nil, err
		}

		keys := ExtractPropsKeys(content)
		if keys == nil {
			continue
		}

		templateKeys = append(templateKeys, keys)
	}

	return templateKeys, nil
}

// CleanVault classifies every note under cfg.NotesPath and writes the report
// to CLEANFILE at the root of the vault.
func CleanVault(cfg types.Config, vault *Vault) error {
	names, err := util.ReadDirFiles(cfg.NotesPath)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", cfg.NotesPath)
		return err
	}

	orphans := []string{"------ORPHANS------"}
	dangling := []string{"------DANGLING-----"}
	badFormat := []string{"-----BADFORMAT-----"}

	for _, name := range names {
		// The report lives in the vault root, but skip it anyway so a report
		// written beside the notes is never classified as a note itself.
		if name == CLEANFILE {
			continue
		}

		content, err := util.ReadFile(cfg.NotesPath, name)
		if err != nil {
			fmt.Println("Error reading file:", err)
			continue
		}

		// 1 find all links in the file
		links := GetLinks(content)

		// 2 check to see if the file follows a recent template
		if !UsesTemplate(content, vault) {
			badFormat = append(badFormat, name)
		}

		// 3 Check if the file contains dangling links
		if ContainsDangling(cfg, vault, links) {
			dangling = append(dangling, name)
		}

		// 4 Check if the file is an orphan
		if len(links) == 0 {
			orphans = append(orphans, name)
		}
	}

	return util.WriteFiles(filepath.Join(cfg.VaultPath, CLEANFILE), badFormat, dangling, orphans)
}
