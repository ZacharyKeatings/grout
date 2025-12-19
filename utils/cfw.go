package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"grout/constants"
	"grout/romm"
)

func GetCFW() constants.CFW {
	cfwEnv := strings.ToUpper(os.Getenv("CFW"))
	cfw := constants.CFW(cfwEnv)

	switch cfw {
	case constants.MuOS, constants.NextUI, constants.Knulli:
		return cfw
	default:
		LogStandardFatal(
			fmt.Sprintf("Unsupported CFW: '%s'. Valid options: NextUI, muOS, Knulli", cfwEnv),
			nil,
		)
		return ""
	}
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
		return filepath.Join(getBasePath(constants.NextUI), "Roms")
	case constants.Knulli:
		return filepath.Join(getBasePath(constants.Knulli), "roms")

	}

	return ""
}

func GetPlatformRomDirectory(config Config, platform romm.Platform) string {
	rp := config.DirectoryMappings[platform.Slug].RelativePath

	if rp == "" {
		rp = RomMSlugToCFW(platform.Slug)
	}

	return filepath.Join(GetRomDirectory(), rp)
}

func GetArtDirectory(config Config, platform romm.Platform) string {
	switch GetCFW() {
	case constants.NextUI:
		romDir := GetPlatformRomDirectory(config, platform)
		return filepath.Join(romDir, ".media")
	case constants.Knulli:
		romDir := GetPlatformRomDirectory(config, platform)
		return filepath.Join(romDir, "images")
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

func GetPlatformMap(cfw constants.CFW) map[string][]string {
	switch cfw {
	case constants.MuOS:
		return constants.MuOSPlatforms
	case constants.NextUI:
		return constants.NextUIPlatforms
	case constants.Knulli:
		return constants.KnulliPlatforms
	default:
		return nil
	}
}

func GetSaveDirectoriesMap(cfw constants.CFW) map[string][]string {
	switch cfw {
	case constants.MuOS:
		return constants.MuOSSaveDirectories
	case constants.NextUI:
		return constants.NextUISaveDirectories
	case constants.Knulli:
		return constants.KnulliSaveDirectories
	default:
		return nil
	}
}

func GetSaveDirectoriesForSlug(slug string) []string {
	saveDirectoriesMap := GetSaveDirectoriesMap(GetCFW())
	if saveDirectoriesMap == nil {
		return nil
	}
	return saveDirectoriesMap[slug]
}

func RomMSlugToCFW(slug string) string {
	cfwPlatformMap := GetPlatformMap(GetCFW())
	if cfwPlatformMap == nil {
		return strings.ToLower(slug)
	}

	if value, ok := cfwPlatformMap[slug]; ok {
		if len(value) > 0 {
			return value[0]
		}

		return ""
	}

	return strings.ToLower(slug)
}

func RomFolderBase(path string) string {
	switch GetCFW() {
	case constants.MuOS, constants.Knulli:
		return path
	case constants.NextUI:
		return ParseTag(path)
	default:
		return path
	}
}

func getBasePath(cfw constants.CFW) string {
	switch cfw {
	case constants.MuOS:
		if os.Getenv("MUOS_BASE_PATH") != "" {
			return os.Getenv("MUOS_BASE_PATH")
		}
		// Hack to see if there is actually content
		sd2InfoDir := filepath.Join(constants.MuOSSD2, "MuOS", "info")
		if _, err := os.Stat(sd2InfoDir); err == nil {
			return constants.MuOSSD2
		}
		return constants.MuOSSD1

	case constants.NextUI:
		if os.Getenv("NEXTUI_BASE_PATH") != "" {
			return os.Getenv("NEXTUI_BASE_PATH")
		}
		return "/mnt/SDCARD"

	case constants.Knulli:
		if os.Getenv("KNULLI_BASE_PATH") != "" {
			return os.Getenv("KNULLI_BASE_PATH")
		}
		return "/userdata"

	default:
		return ""
	}
}

func getMuOSInfoDirectory() string {
	return filepath.Join(getBasePath(constants.MuOS), "MUOS", "info")
}

func getSaveDirectory() string {
	cfw := GetCFW()
	switch cfw {
	case constants.MuOS:
		return filepath.Join(getBasePath(cfw), "MUOS", "save", "file")
	case constants.NextUI:
		return filepath.Join(getBasePath(cfw), "Saves")
	case constants.Knulli:
		return filepath.Join(getBasePath(cfw), "saves")
	}

	return ""
}
