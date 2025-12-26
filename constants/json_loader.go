package constants

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed bios cfw
var embeddedFiles embed.FS

func loadJSONMap[K comparable, V any](path string) (map[K]V, error) {
	var data []byte
	var err error

	// Check for override file in current working directory
	overridePath := filepath.Join("overrides", path)
	if fileData, readErr := os.ReadFile(overridePath); readErr == nil {
		data = fileData
	} else {
		// Fall back to embedded file
		data, err = embeddedFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
	}

	var result map[K]V
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return result, nil
}

func mustLoadJSONMap[K comparable, V any](path string) map[K]V {
	result, err := loadJSONMap[K, V](path)
	if err != nil {
		panic(err)
	}
	return result
}
