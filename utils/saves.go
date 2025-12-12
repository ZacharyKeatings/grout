package utils

import (
	"fmt"
	"grout/constants"
	"os"
	"path/filepath"
	"strings"
	"time"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

const ROMM_ISO8601 = "2006-01-02 15-04-05-000"

type LocalSave struct {
	Slug         string
	Path         string
	LastModified time.Time
}

func (lc LocalSave) Backup() error {
	ext := filepath.Ext(lc.Path)
	base := strings.ReplaceAll(filepath.Base(lc.Path), ext, "")

	lm := lc.LastModified.Format(ROMM_ISO8601)

	bfn := fmt.Sprintf("%s [%s]%s", base, lm, ext)
	bfn = strings.ReplaceAll(bfn, ":", "-")

	bsd := filepath.Dir(lc.Path)
	dest := filepath.Join(bsd, ".backup", bfn)

	return CopyFile(lc.Path, dest)
}

func FindSaveFiles(slug string) []LocalSave {
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
		return []LocalSave{}
	}

	var allSaveFiles []LocalSave

	for _, saveFolder := range saveFolders {
		sd := filepath.Join(bsd, saveFolder)

		if _, err := os.Stat(sd); os.IsNotExist(err) {
			logger.Debug("Save directory does not exist", "path", sd)
			continue
		}

		entries, err := os.ReadDir(sd)
		if err != nil {
			logger.Error("Failed to read save directory", "path", sd, "error", err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				savePath := filepath.Join(sd, entry.Name())

				fileInfo, err := entry.Info()
				if err != nil {
					logger.Warn("Failed to get file info", "file", entry.Name(), "error", err)
					continue
				}

				saveFile := LocalSave{
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
