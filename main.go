// Entry point of Obsidian helpers program
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Output file names
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

func run() {

	vaultPath := flag.String("vaultPath", "/", "A path to an obsidian vault")
	noteDir := flag.String("noteDir", "/", "Path to the directory in the obsidian vault to modify")
	templateDir := flag.String("templateDir", "/", "Path to the directory containing Obsidian templates")

	flag.Parse()

	pathToNotes := filepath.Join(*vaultPath, *noteDir)

	initTemplateRegexArr(*templateDir)

	files, err := os.ReadDir(pathToNotes)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", pathToNotes)
		return
	}

	var orphans []string
	var dangling []string
	var badFormat []string
	for _, file := range files {
		// The reports are written into the directory being scanned, so skip
		// them or each run reports the previous run's output as notes.
		if !file.IsDir() && file.Name() != orphansOutFile && file.Name() != danglingOutFile {
			content, err := os.ReadFile(filepath.Join(pathToNotes, file.Name()))
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

				_, err := os.Stat(filepath.Join(pathToNotes, candidate))
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

	outFile, err := os.OpenFile(filepath.Join(pathToNotes, badFormatOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	outFile, err = os.OpenFile(filepath.Join(pathToNotes, orphansOutFile), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0645)
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

	outFile, err = os.OpenFile(filepath.Join(pathToNotes, danglingOutFile), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
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
