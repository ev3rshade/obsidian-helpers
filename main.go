// Command go-obsidian-helper reports orphaned notes, dangling links and
// badly formatted notes in an Obsidian vault.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ev3rshade/go-obsidian-helper/types"
	"github.com/ev3rshade/go-obsidian-helper/vault"
)

func parseFlags() types.Config {
	var cfg types.Config

	flag.StringVar(&cfg.VaultPath, "vaultPath", "/", "the path to the directory containing obsidian files")
	flag.StringVar(&cfg.TemplatesPath, "templatesPath", "/", "the path to the directory containing obsidian templates")
	flag.StringVar(&cfg.NotesPath, "notesPath", "/", "the path to the directory containing obsidian notes")
	flag.Parse()

	return cfg
}

func main() {
	cfg := parseFlags()

	myVault, err := vault.ParseVault(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := vault.CleanVault(cfg, &myVault); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Println("Success!")
}
