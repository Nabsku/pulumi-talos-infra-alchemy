// Package file provides file-related utilities.
package file

import (
	"fmt"
	"os"
	"strings"
)

func WriteToFile(fileName, content string) error {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}

	return nil
}

func GatherPatchFilesInDir(configDir string) ([]string, error) {
	f, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	var files []string

	for _, file := range f {
		if !strings.Contains(file.Name(), "yaml") {
			continue
		}
		if !file.IsDir() {
			files = append(files, fmt.Sprintf("%s/%s", configDir, file.Name()))
		}
	}

	return files, nil
}
