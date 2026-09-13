package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"go-obsidian-helper/types"
	"go-obsidian-helper/util"
)

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

const CLEANFILE = "!! CLEANFILE.md"

func (vault *Vault) intern(path string) NoteID {
	var id NoteID
	if id, ok := vault.ids[path]; ok {
		return id
	}
	vault.ids[path] = id
	vault.notes = append(vault.notes, Note{})
	vault.out = append(vault.out, nil)
	vault.in = append(vault.in, nil)
	return id
}

func ParseVault(cfg types.Config) (Vault, error) {
	var vault Vault
	vault.ids = make(map[string]NoteID)

	vault.path = cfg.VaultPath
	var err error
	vault.templateKeys, err = initTemplateRegexArr(cfg.TemplatesPath)
	if err != nil {
		fmt.Println("Error reading in templates:", err)
		return Vault{}, err
	}

	files, err := os.ReadDir(cfg.NotesPath)
	if err != nil {
		fmt.Println("Error opening vault directory:", err)
		return Vault{}, err
	}

	// fill in file information
	for _, f := range files {
		fPath := filepath.Join(cfg.NotesPath, f.Name())
		currID := vault.intern(fPath)

		// update id, note, out, in
		vault.notes[currID] = Note{
			Path:    fPath,
			Title:   f.Name(),
			Aliases: nil,
			Exists:  true,
		}
	}

	return vault, nil
}

func initTemplateRegexArr(templateDir string) ([]map[string]bool, error) {
	var templateKeys []map[string]bool

	templates, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}

	for _, t := range templates {
		if !t.IsDir() {
			content, err := os.ReadFile(filepath.Join(templateDir, t.Name()))
			if err != nil {
				fmt.Println("Error reading file:", err)
				return nil, err
			}

			keys := ExtractPropsKeys(string(content))
			if keys == nil {
				continue
			}

			fmt.Println("Template keys:", keys)
			templateKeys = append(templateKeys, keys)
		}
	}

	return templateKeys, nil
}

func CleanVault(cfg types.Config, vault *Vault) error {
	files, err := os.ReadDir(cfg.NotesPath)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", cfg.NotesPath)
		return err
	}

	orphans := []string{"------ORPHANS------"}
	dangling := []string{"------DANGLING-----"}
	badFormat := []string{"-----BADFORMAT-----"}
	for _, file := range files {
		if !file.IsDir() {
			content, err := os.ReadFile(filepath.Join(cfg.NotesPath, file.Name()))
			if err != nil {
				fmt.Println("Error reading file:", err)
				continue
			}

			// 1 find all links in the file
			links := GetLinks(string(content))

			// 2 check to see if the file follows a recent template
			if !UsesTemplate(string(content), vault) {
				badFormat = append(badFormat, file.Name())
			}

			// 3 Check if the file contains dangling links
			if ContainsDangling(cfg, vault, links) {
				dangling = append(dangling, file.Name())
			}

			// 4 Check if the file is an orphan
			if len(links) == 0 && IsOrphan(file.Name()) {
				orphans = append(orphans, file.Name())
			}
		}

		util.WriteFiles(filepath.Join(cfg.VaultPath, CLEANFILE), badFormat, dangling, orphans)
	}
	return nil
}
