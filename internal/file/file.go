// Package file provides file-related utilities.
package file

import "os"

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
