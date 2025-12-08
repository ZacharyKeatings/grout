package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

func DeleteFile(path string) bool {
	logger := gaba.GetLogger()

	err := os.Remove(path)
	if err != nil {
		logger.Error("Issue removing file",
			"path", path,
			"error", err)
		return false
	} else {
		logger.Debug("Removed file", "path", path)
		return true
	}
}

// ExtractZip extracts a zip file to a destination directory
func ExtractZip(zipData []byte, destDir string) error {
	logger := gaba.GetLogger()

	// Create a reader from the zip data
	reader, err := zip.NewReader(&bytesReaderAt{zipData}, int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to read zip data: %w", err)
	}

	// Create the destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract each file
	for _, file := range reader.File {
		filePath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			// Create directory
			if err := os.MkdirAll(filePath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", filePath, err)
		}

		// Extract file
		if err := extractFile(file, filePath); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}

		logger.Debug("Extracted file", "file", file.Name, "dest", filePath)
	}

	return nil
}

// extractFile extracts a single file from a zip archive
func extractFile(file *zip.File, destPath string) error {
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// bytesReaderAt implements io.ReaderAt for a byte slice
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = copy(p, r.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
