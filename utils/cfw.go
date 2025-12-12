package utils

import (
	"fmt"
	"grout/constants"
	"os"
	"path/filepath"
	"strings"

	"grout/romm"
)

func GetCFW() constants.CFW {
	cfw := strings.ToLower(os.Getenv("CFW"))
	switch cfw {
	case "muos":
		return constants.MuOS
	case "nextui":
		return constants.NextUI
	default:
		LogStandardFatal(fmt.Sprintf("Unsupported CFW: %s", cfw), nil)
	}
	return ""
}

func GetRomDirectory() string {
	if os.Getenv("ROM_DIRECTORY") != "" {
		return os.Getenv("ROM_DIRECTORY")
	}

	cfw := GetCFW()

	switch cfw {
	case constants.MuOS:
		return constants.MuOSRomsFolderUnion
	case constants.NextUI:
		return constants.NextUIRomsFolder
	}

	return ""
}

func GetSaveDirectory() string {
	switch GetCFW() {
	case constants.MuOS:
		return filepath.Join(GetMuOSBasePath(), "save", "file")

	case constants.NextUI:
		return constants.NextUISavesFolder
	}

	return ""
}

func GetMuOSBasePath() string {
	if os.Getenv("MUOS_BASE_PATH") != "" {
		return os.Getenv("MUOS_BASE_PATH")
	}

	// Hack to see if there is actually content
	sd2InfoDir := filepath.Join(constants.MuOSSD2, "MuOS", "info")
	if _, err := os.Stat(sd2InfoDir); err == nil {
		return constants.MuOSSD2
	}

	return constants.MuOSSD1
}

func GetMuOSInfoDirectory() string {
	return filepath.Join(GetMuOSBasePath(), "info")
}

func GetPlatformRomDirectory(config Config, platform romm.Platform) string {
	rp := config.DirectoryMappings[platform.Slug].RelativePath

	if rp == "" {
		rp = RomMSlugToCFW(platform.Slug)
	}

	return filepath.Join(GetRomDirectory(), rp)
}

func RomMSlugToCFW(slug string) string {
	var cfwPlatformMap map[string][]string

	switch GetCFW() {
	case constants.MuOS:
		cfwPlatformMap = constants.MuOSPlatforms
	case constants.NextUI:
		cfwPlatformMap = constants.NextUIPlatforms
	}

	if value, ok := cfwPlatformMap[slug]; ok {
		if len(value) > 0 {
			return value[0]
		}

		return ""
	} else {
		return strings.ToLower(slug)
	}
}

func RomFolderBase(path string) string {
	switch GetCFW() {
	case constants.MuOS:
		return path
	case constants.NextUI:
		_, tag := NameCleaner(path, true)
		return tag
	default:
		return path
	}
}

func GetArtDirectory(config Config, platform romm.Platform) string {
	switch GetCFW() {
	case constants.NextUI:
		romDir := GetPlatformRomDirectory(config, platform)
		return filepath.Join(romDir, ".media")
	case constants.MuOS:
		systemName, exists := constants.MuOSArtDirectory[platform.Slug]
		if !exists {
			// Fallback to platform display name if not in map
			systemName = platform.Name
		}
		muosInfoDir := GetMuOSInfoDirectory()
		return filepath.Join(muosInfoDir, "catalogue", systemName, "box")
	default:
		return ""
	}
}
