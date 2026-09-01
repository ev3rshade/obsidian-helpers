package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const orphansOutFile = "! orphans.md"
const danglingOutFile = "! dangling.md"

var linkPattern = regexp.MustCompile(`\[\[([^]]+)\]\]`)

type Config struct {
	Path string
}

func parseFlags() *Config {
	var cfg Config

	flag.StringVar(&cfg.Path, "path", "/", "the path to the directory containing obsidian files")
	flag.Parse()

	return &cfg
}

func run() {

	cfg := parseFlags()

	// err := os.Mkdir("orphans", 0755)
	// if err != nil {
	// 	fmt.Println("Error: failed to make folder")
	// 	return
	// }
	// err = os.Mkdir("dangling links", 0755)
	// if err != nil {
	// 	fmt.Println("Error: failed to make folder")
	// 	return
	// }

	files, err := os.ReadDir(cfg.Path)
	if err != nil {
		fmt.Println("Error: couldn't open to read files", cfg.Path)
		return
	}

	var orphans []string
	var dangling []string
	for _, file := range files {
		if !file.IsDir() {
			content, err := os.ReadFile(filepath.Join(cfg.Path, file.Name()))
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

	outFile, err := os.OpenFile(filepath.Join(cfg.Path, orphansOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	outFile, err = os.OpenFile(filepath.Join(cfg.Path, danglingOutFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
