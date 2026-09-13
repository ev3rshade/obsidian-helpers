package main

import (
	"flag"
	"fmt"
	"go-obsidian-helper/types"
	"go-obsidian-helper/vault"
)

func parseFlags() *types.Config {
	var cfg types.Config

	flag.StringVar(&cfg.VaultPath, "vaultPath", "/", "the path to the directory containing obsidian files")
	flag.StringVar(&cfg.TemplatesPath, "templatesPath", "/", "the path to the directory containing obsidian templates")
	flag.StringVar(&cfg.NotesPath, "notesPath", "/", "the path to the directory containing obsidian notes")
	flag.Parse()

	return &cfg
}

func run() error {
	cfg := parseFlags()

	myVault, err := vault.ParseVault(*cfg)
	if err != nil {
		return err
	}

	err = vault.CleanVault(*cfg, &myVault)

	fmt.Println("Success!")
	return nil
}

func main() {
	run()
}
