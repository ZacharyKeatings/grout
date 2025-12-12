package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"grout/constants"
	"grout/romm"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

type LocalRomFile struct {
	RomID        int
	RomName      string
	Slug         string
	Path         string
	FileName     string
	SHA1         string
	LastModified time.Time
	RemoteSaves  []romm.Save
	SaveFile     *LocalSave
}

func (lrf LocalRomFile) SyncAction() SyncAction {
	if lrf.SaveFile == nil && len(lrf.RemoteSaves) == 0 {
		return Skip
	}
	if lrf.SaveFile != nil && len(lrf.RemoteSaves) == 0 {
		return Upload
	}
	if lrf.SaveFile == nil && len(lrf.RemoteSaves) > 0 {
		return Download
	}

	switch lrf.SaveFile.LastModified.Compare(lrf.LastRemoteSave().UpdatedAt) {
	case -1:
		return Download
	case 0:
		return Skip
	case 1:
		return Upload
	default:
		return Skip
	}
}

func (lrf LocalRomFile) LastRemoteSave() romm.Save {
	slices.SortFunc(lrf.RemoteSaves, func(s1 romm.Save, s2 romm.Save) int {
		return s1.UpdatedAt.Compare(s2.UpdatedAt)
	})

	return lrf.RemoteSaves[0]
}

func ScanAllRoms() map[string][]LocalRomFile {
	logger := gaba.GetLogger()
	result := make(map[string][]LocalRomFile)

	var platformMap map[string][]string
	switch GetCFW() {
	case constants.MuOS:
		platformMap = constants.MuOSPlatforms
	case constants.NextUI:
		platformMap = constants.NextUIPlatforms
	default:
		logger.Warn("Unknown CFW, cannot scan ROMs")
		return result
	}

	baseRomDir := GetRomDirectory()
	logger.Debug("Starting ROM scan", "baseDir", baseRomDir)

	for slug := range platformMap {
		romFolderName := RomMSlugToCFW(slug)
		if romFolderName == "" {
			logger.Debug("No ROM folder mapping for slug", "slug", slug)
			continue
		}

		romDir := filepath.Join(baseRomDir, romFolderName)

		if _, err := os.Stat(romDir); os.IsNotExist(err) {
			logger.Debug("ROM directory does not exist", "slug", slug, "path", romDir)
			continue
		}

		saveFiles := FindSaveFiles(slug)
		saveFileMap := make(map[string]*LocalSave)
		for i := range saveFiles {
			baseName := strings.TrimSuffix(filepath.Base(saveFiles[i].Path), filepath.Ext(saveFiles[i].Path))
			saveFileMap[baseName] = &saveFiles[i]
		}

		roms := scanRomDirectory(slug, romDir, saveFileMap)
		if len(roms) > 0 {
			result[slug] = roms
			logger.Debug("Found ROMs for platform", "slug", slug, "count", len(roms))
		}
	}

	totalRoms := 0
	for _, roms := range result {
		totalRoms += len(roms)
	}
	logger.Debug("Completed ROM scan", "platforms", len(result), "totalRoms", totalRoms)

	return result
}

func scanRomDirectory(slug, romDir string, saveFileMap map[string]*LocalSave) []LocalRomFile {
	logger := gaba.GetLogger()
	var roms []LocalRomFile

	entries, err := os.ReadDir(romDir)
	if err != nil {
		logger.Error("Failed to read ROM directory", "path", romDir, "error", err)
		return roms
	}

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if shouldSkipFile(entry.Name()) {
			continue
		}

		romPath := filepath.Join(romDir, entry.Name())

		fileInfo, err := entry.Info()
		if err != nil {
			logger.Warn("Failed to get file info", "file", entry.Name(), "error", err)
			continue
		}

		hash, err := calculateRomSHA1(romPath)
		if err != nil {
			logger.Warn("Failed to calculate SHA1 for ROM", "path", romPath, "error", err)
		}

		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		var saveFile *LocalSave
		if sf, found := saveFileMap[baseName]; found {
			saveFile = sf
		}

		rom := LocalRomFile{
			Slug:         slug,
			Path:         romPath,
			FileName:     entry.Name(),
			SHA1:         hash,
			LastModified: fileInfo.ModTime(),
			SaveFile:     saveFile,
		}

		roms = append(roms, rom)
	}

	return roms
}

func shouldSkipFile(filename string) bool {
	skipExtensions := []string{
		".txt", ".nfo", ".diz", ".db",
		".ini", ".cfg", ".conf",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".m3u",                   // Playlist files
		".cue",                   // Some systems use .cue but we may want to include these
		".srm", ".sav", ".state", // Save files
	}

	skipNames := []string{
		"desktop.ini",
		"thumbs.db",
		".ds_store",
	}

	lowerName := strings.ToLower(filename)

	// Check skip names
	for _, skip := range skipNames {
		if lowerName == skip {
			return true
		}
	}

	// Check skip extensions
	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}

	return false
}

func calculateRomSHA1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
