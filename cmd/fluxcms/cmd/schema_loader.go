package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infrasutra/fsl/parser"
)

func collectFSLFiles(paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", path, err)
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".fsl") {
					files = append(files, p)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("error walking directory: %w", err)
			}
			continue
		}

		files = append(files, path)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .fsl files found")
	}

	return files, nil
}

func loadSchemas(path string) ([]*parser.Schema, error) {
	files, err := collectFSLFiles([]string{path})
	if err != nil {
		return nil, err
	}

	fileContents := make(map[string]string, len(files))
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return nil, fmt.Errorf("cannot read %s: %w", file, readErr)
		}
		fileContents[file] = string(content)
	}

	results := parseFilesWithWorkspaceTypes(fileContents)

	var schemas []*parser.Schema
	for _, file := range files {
		result := results[file]
		if !result.Valid {
			message := "unknown error"
			if len(result.Diagnostics) > 0 {
				message = result.Diagnostics[0].Message
			}
			return nil, fmt.Errorf("schema validation failed in %s: %s", file, message)
		}

		schemas = append(schemas, result.Schema)
	}

	return schemas, nil
}
