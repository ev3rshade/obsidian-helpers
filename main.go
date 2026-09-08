package main

import (
	"flag"
	"gopkg.in/yaml.v3"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

func extractPropsKeys(content string) map[string]bool {
	propsRegex := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
	matches := propsRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil
	}

	var raw map[string]interface{}
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

func usesTemplate(content string) bool {
	noteKeys := extractPropsKeys(content)
	if noteKeys == nil {
		return false
	}

	for _, required := range templateKeys {
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

			// template checker
			if !usesTemplate(string(content)) {
				badFormat = append(badFormat, file.Name())
			}

			// orphans and dangling links

			matches := linkPattern.FindAllStringSubmatch(string(content), -1)

			var links []string
			for _, m := range matches {
				links = append(links, m[1])
			}

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

				_, err := os.Stat(filepath.Join(cfg.Path, candidate))
				if err != nil {
					dangling = append(dangling, file.Name())
					break
				}
			}

			if len(links) == 0 {
				orphans = append(orphans, file.Name())
			}
		}
	}

	outFile, err := os.OpenFile(filepath.Join(cfg.Path, badFormatOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer outFile.Close()

	for _, bf := range badFormat {
		_, err := outFile.WriteString("[[" + bf + "]]" + "\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return
		}
	}

	outFile, err = os.OpenFile(filepath.Join(cfg.Path, orphansOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer outFile.Close()

	for _, o := range orphans {
		_, err := outFile.WriteString("[[" + o + "]]" + "\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return
		}
	}

	outFile, err = os.OpenFile(filepath.Join(cfg.Path, danglingOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer outFile.Close()

	for _, d := range dangling {
		_, err := outFile.WriteString("[[" + d + "]]" + "\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return
		}
	}
}

func main() {
	run()
}
