package utils

import (
	"grout/constants"
	"grout/romm"
	"os"
	"path/filepath"
	"strings"
	"time"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

type LocalSaveFile struct {
	Slug         string
	Path         string
	LastModified time.Time
}

type SyncAction string

const (
	Download SyncAction = "DOWNLOAD"
	Upload              = "UPLOAD"
	Nothing             = "NOTHING"
)

func FindSaveFiles(slug string) []LocalSaveFile {
	logger := gaba.GetLogger()

	bsd := GetSaveDirectory()
	var saveFolders []string

	switch GetCFW() {
	case constants.MuOS:
		saveFolders = constants.MuOSSaveDirectories[slug]
	case constants.NextUI:
		saveFolder := constants.NextUISaves[slug]
		if saveFolder != "" {
			saveFolders = []string{saveFolder}
		}
	}

	if len(saveFolders) == 0 {
		logger.Debug("No save folder mapping for slug", "slug", slug)
		return []LocalSaveFile{}
	}

	var allSaveFiles []LocalSaveFile

	for _, saveFolder := range saveFolders {
		sd := filepath.Join(bsd, saveFolder)

		// Check if directory exists
		if _, err := os.Stat(sd); os.IsNotExist(err) {
			logger.Debug("Save directory does not exist", "path", sd)
			continue
		}

		// Read directory contents
		entries, err := os.ReadDir(sd)
		if err != nil {
			logger.Error("Failed to read save directory", "path", sd, "error", err)
			continue
		}

		// Process each file
		for _, entry := range entries {
			if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				savePath := filepath.Join(sd, entry.Name())

				// Get file info for last modified time
				fileInfo, err := entry.Info()
				if err != nil {
					logger.Warn("Failed to get file info", "file", entry.Name(), "error", err)
					continue
				}

				saveFile := LocalSaveFile{
					Slug:         slug,
					Path:         savePath,
					LastModified: fileInfo.ModTime(),
				}

				allSaveFiles = append(allSaveFiles, saveFile)
			}
		}

		logger.Debug("Found save files in directory", "path", sd, "count", len(entries))
	}

	logger.Debug("Found total save files", "slug", slug, "count", len(allSaveFiles))
	return allSaveFiles
}

func remoteSaveUploadTime(s romm.Save) (time.Time, bool) {
	_, afterL, ok := strings.Cut(s.FileName, "[")
	if !ok {
		return time.Time{}, false
	}

	stamp, _, ok := strings.Cut(afterL, "]")
	if !ok {
		return time.Time{}, false
	}

	// Layout matches: YYYY-MM-DD HH-MM-SS-mmm
	parsed, err := time.Parse("2006-01-02 15-04-05-000", strings.TrimSpace(stamp))
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}
