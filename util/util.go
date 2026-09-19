// Package util holds filesystem helpers shared across the helper's packages.
package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFiles writes each list of note names to outFile as wikilinks,
// separating the lists with blank lines. Any existing file is truncated.
func WriteFiles(outFile string, files ...[]string) (err error) {
	fout, err := os.OpenFile(outFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer func() {
		if cerr := fout.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", outFile, cerr)
		}
	}()

	for _, list := range files {
		for _, f := range list {
			if _, err := fout.WriteString("[[" + f + "]]" + "\n"); err != nil {
				fmt.Print("Error: writing to file", err)
				return err
			}
		}
		if _, err := fout.WriteString("\n\n"); err != nil {
			fmt.Print("Error: writing to file", err)
			return err
		}
	}

	return nil
}

// ReadDirFiles returns the non-directory entries of dir by name.
func ReadDirFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	return names, nil
}

// ReadFile returns the contents of the named file inside dir.
func ReadFile(dir, name string) (string, error) {
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}

	return string(content), nil
}
