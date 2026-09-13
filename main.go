package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	
)

const orphansOutFile = "! orphans.md"
const danglingOutFile = "! dangling.md"
const badFormatOutFile = "! badFormat.md"

var linkPattern = regexp.MustCompile(`\[\[([^]]+)\]\]`)

type Config struct {
	Path        string
	TemplateDir string
}

var templateKeys []map[string]bool


func initTemplateRegexArr(templateDir string) error {

	templates, err := os.ReadDir(templateDir)
	if err != nil {
		return err
	}

	for _, t := range templates {
		if !t.IsDir() {
			content, err := os.ReadFile(filepath.Join(templateDir, t.Name()))
			if err != nil {
				fmt.Println("Error reading file:", err)
				return err
			}

			keys := extractPropsKeys(string(content))
			if keys == nil {
				continue
			}

			fmt.Println("Template keys:", keys)
			templateKeys = append(templateKeys, keys)
		}
	}

	return nil
}



func writeFiles(cfg *Config, outFile string, files []string) error {
	fout, err := os.OpenFile(filepath.Join(cfg.Path, outFile), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer fout.Close()

	for _, f := range files {
		_, err := fout.WriteString("[[" + f + "]]" + "\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return err
		}
	}

	return nil
}

func parseFlags() *Config {
	var cfg Config

	flag.StringVar(&cfg.Path, "path", "/", "the path to the directory containing obsidian files")
	flag.StringVar(&cfg.TemplateDir, "templateDir", "/", "the path to the directory containing obsidian templates")
	flag.Parse()

	return &cfg
}

func run() {

	cfg := parseFlags()
	initTemplateRegexArr(cfg.TemplateDir)

	files, err := os.ReadDir(cfg.Path)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", cfg.Path)
		return
	}

	var orphans []string
	var dangling []string
	var badFormat []string
	for _, file := range files {
		if !file.IsDir() {
			content, err := os.ReadFile(filepath.Join(cfg.Path, file.Name()))
			if err != nil {
				fmt.Println("Error reading file:", err)
				continue
			}
			
			// 1 find all links in the file
			links := getLinks(string(content))

			// 2 check to see if the file follows a recent template
			if !usesTemplate(string(content)) {
				badFormat = append(badFormat, file.Name())
			}

			// 3 Check if the file contains dangling links			
			if containsDangling(cfg, links) {
				dangling = append(dangling, file.Name())
			}

			// 4 Check if the file is an orphan
			if len(links) == 0 && isOrphan(file.Name()) {
				orphans = append(orphans, file.Name())
			}
		}
	}

	err = writeFiles(cfg, badFormatOutFile, badFormat)
	if err != nil {
		fmt.Println("Error writing files:", err)
	}

	err = writeFiles(cfg, danglingOutFile, dangling)
	if err != nil {
		fmt.Println("Error writing files:", err)
	}
	
	err = writeFiles(cfg, orphansOutFile, orphans)
	if err != nil {
		fmt.Println("Error writing files:", err)
	}
}

func main() {
	run()
}