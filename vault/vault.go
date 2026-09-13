package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

type NoteID int32

type Note struct {
	Path    string
	Title   string
	Aliases []string
	Exists  bool
}

type Vault struct {
	path  string
	ids   map[string]NoteID
	notes []Note
	out   [][]NoteID
	in    [][]NoteID
}

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

func parseVault(vaultPath string) (Vault, error) {
	vault := Vault{}

	files, err := os.ReadDir(vaultPath)
	if err != nil {
		fmt.Println("Error opening vault directory:", err)
		return Vault{}, err
	}

	// fill in file information
	for _, f := range files {
		fPath := filepath.Join(vaultPath, f.Name())
		currID := vault.intern(fPath)

		// update note, out, in
		vault.notes[currID] = Note{
			Path:    fPath,
			Title:   f.Name(),
			Aliases: nil,
			Exists:  true,
		}
		
	}

	vault.path = path

	return vault
}
