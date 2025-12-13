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
		return filepath.Join(getNextUIBasePath(), "Roms")
	}

	return ""
}

func getSaveDirectory() string {
	switch GetCFW() {
	case constants.MuOS:
		return filepath.Join(getMuOSBasePath(), "MUOS", "save", "file")

	case constants.NextUI:
		return filepath.Join(getNextUIBasePath(), "Saves")
	}

	return ""
}

func getMuOSBasePath() string {
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

func getNextUIBasePath() string {
	if os.Getenv("NEXTUI_BASE_PATH") != "" {
		return os.Getenv("NEXTUI_BASE_PATH")
	}

	return "/mnt/SDCARD"
}

func getMuOSInfoDirectory() string {
	return filepath.Join(getMuOSBasePath(), "MUOS", "info")
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
		return ParseTag(path)
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
			systemName = platform.Name
		}
		muosInfoDir := getMuOSInfoDirectory()
		return filepath.Join(muosInfoDir, "catalogue", systemName, "box")
	default:
		return ""
	}
}
