// Entry point of Obsidian helpers program
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Output file names
const orphansOutFile = "! orphans.md"
const danglingOutFile = "! dangling.md"

var linkPattern = regexp.MustCompile(`\[\[([^]]+)\]\]`)

func run() {

	vaultPath := flag.String("vaultPath", "/", "A path to an obsidian vault")
	noteDir := flag.String("noteDir", "/", "Path to the directory in the obsidian vault to modify")
	flag.Parse()

	pathToNotes := filepath.Join(*vaultPath, *noteDir)

	files, err := os.ReadDir(pathToNotes)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", pathToNotes)
		return
	}

	var orphans []string
	var dangling []string
	for _, file := range files {
		// The reports are written into the directory being scanned, so skip
		// them or each run reports the previous run's output as notes.
		if !file.IsDir() && file.Name() != orphansOutFile && file.Name() != danglingOutFile {
			content, err := os.ReadFile(filepath.Join(pathToNotes, file.Name()))
			if err != nil {
				fmt.Println("Error reading file:", err)
				continue
			}

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

	outFile, err := os.OpenFile(filepath.Join(pathToNotes, orphansOutFile), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	defer outFile.Close()

	for _, o := range orphans {
		_, err := outFile.WriteString(o + "\n")
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
		_, err := outFile.WriteString(d + "\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return
		}
	}
}

func main() {
	run()
}
