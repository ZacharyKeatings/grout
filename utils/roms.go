package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"grout/constants"
	"grout/romm"
	"io"
	"os"
	"path/filepath"
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
	SaveFile     *LocalSaveFile
}

func (lrf LocalRomFile) SyncAction() SyncAction {
	if lrf.SaveFile == nil && len(lrf.RemoteSaves) == 0 {
		return Nothing
	}
	if lrf.SaveFile != nil && len(lrf.RemoteSaves) == 0 {
		return Upload
	}
	if lrf.SaveFile == nil && len(lrf.RemoteSaves) > 0 {
		return Download
	}

	lastRemoteSaveUploadTime := lrf.RemoteSaves[0].UpdatedAt
	if ts, ok := remoteSaveUploadTime(lrf.RemoteSaves[0]); ok {
		lastRemoteSaveUploadTime = ts
	}

	for i := 1; i < len(lrf.RemoteSaves); i++ {
		cur := lrf.RemoteSaves[i].UpdatedAt
		if ts, ok := remoteSaveUploadTime(lrf.RemoteSaves[i]); ok {
			cur = ts
		}
		if cur.After(lastRemoteSaveUploadTime) {
			lastRemoteSaveUploadTime = cur
		}
	}

	switch lrf.SaveFile.LastModified.Compare(lastRemoteSaveUploadTime) {
	case -1:
		return Download
	case 0:
		return Nothing
	case 1:
		return Upload
	default:
		return Nothing
	}
}

// ScanAllRoms scans all ROM directories and returns ROMs organized by platform slug
func ScanAllRoms() map[string][]LocalRomFile {
	logger := gaba.GetLogger()
	result := make(map[string][]LocalRomFile)

	// Get platform slugs based on CFW
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

	// Scan each platform
	for slug := range platformMap {
		// Get the ROM folder name for this platform
		romFolderName := RomMSlugToCFW(slug)
		if romFolderName == "" {
			logger.Debug("No ROM folder mapping for slug", "slug", slug)
			continue
		}

		romDir := filepath.Join(baseRomDir, romFolderName)

		// Check if directory exists
		if _, err := os.Stat(romDir); os.IsNotExist(err) {
			logger.Debug("ROM directory does not exist", "slug", slug, "path", romDir)
			continue
		}

		// First, find all save files for this platform
		saveFiles := FindSaveFiles(slug)
		saveFileMap := make(map[string]*LocalSaveFile)
		for i := range saveFiles {
			// Get base name from the save file path
			baseName := strings.TrimSuffix(filepath.Base(saveFiles[i].Path), filepath.Ext(saveFiles[i].Path))
			saveFileMap[baseName] = &saveFiles[i]
		}

		// Scan the ROM directory and associate save files
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

// scanRomDirectory scans a single ROM directory and returns LocalRomFile entries
func scanRomDirectory(slug, romDir string, saveFileMap map[string]*LocalSaveFile) []LocalRomFile {
	logger := gaba.GetLogger()
	var roms []LocalRomFile

	entries, err := os.ReadDir(romDir)
	if err != nil {
		logger.Error("Failed to read ROM directory", "path", romDir, "error", err)
		return roms
	}

	for _, entry := range entries {
		// Skip directories and hidden files
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip common non-ROM files
		if shouldSkipFile(entry.Name()) {
			continue
		}

		romPath := filepath.Join(romDir, entry.Name())

		// Get file info
		fileInfo, err := entry.Info()
		if err != nil {
			logger.Warn("Failed to get file info", "file", entry.Name(), "error", err)
			continue
		}

		// Calculate SHA1
		hash, err := calculateRomSHA1(romPath)
		if err != nil {
			logger.Warn("Failed to calculate SHA1 for ROM", "path", romPath, "error", err)
			// Continue anyway with empty hash
		}

		// Look up associated save file
		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		var saveFile *LocalSaveFile
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

// shouldSkipFile checks if a file should be skipped during ROM scanning
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

// calculateRomSHA1 computes the SHA1 hash of a ROM file
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
