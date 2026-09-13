package util

import (
	"fmt"
	"os"
)

func WriteFiles(outFile string, files ...[]string) error {
	fout, err := os.OpenFile(outFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer fout.Close()

	for _, list := range files {
		for _, f := range list {
			_, err := fout.WriteString("[[" + f + "]]" + "\n")
			if err != nil {
				fmt.Print("Error: writing to file", err)
				return err
			}
		}
		_, err := fout.WriteString("\n\n")
		if err != nil {
			fmt.Print("Error: writing to file", err)
			return err
		}
	}

	return nil
}
