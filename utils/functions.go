package utils

import (
	"encoding/json"
	"fmt"
	"grout/models"
	"grout/state"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
	"github.com/brandonkowalski/go-romm"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func init() {}

func GetCFW() models.CFW {
	cfw := strings.ToLower(os.Getenv("CFW"))
	switch cfw {
	case "muos":
		return models.MUOS
	case "nextui":
		return models.NEXTUI
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
	case models.MUOS:
		return muOSRomsFolderUnion
	case models.NEXTUI:
		return nextUIRomsFolder
	}

	return ""
}

func GetPlatformRomDirectory(platform romm.Platform) string {
	config := state.GetAppState().Config
	return filepath.Join(GetRomDirectory(), config.DirectoryMappings[platform.Slug].RelativePath)
}

func LoadConfig() (*models.Config, error) {
	configFiles := []string{"config.json", "config.yml"}

	var data []byte
	var err error
	var foundFile string

	for _, filename := range configFiles {
		data, err = os.ReadFile(filename)
		if err == nil {
			foundFile = filename
			break
		}
	}

	if foundFile == "" {
		return nil, fmt.Errorf("no config file found (tried: %s)", strings.Join(configFiles, ", "))
	}

	var config models.Config

	ext := strings.ToLower(filepath.Ext(foundFile))

	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unknown config file type: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", foundFile, err)
	}

	if ext == ".yaml" || ext == ".yml" {
		gaba.GetLogger().Info("Migrating config to JSON")
		_ = SaveConfig(&config)
	}

	if config.ApiTimeout == 0 {
		config.ApiTimeout = 30 * time.Minute
	}

	if config.DownloadTimeout == 0 {
		config.DownloadTimeout = 60 * time.Minute
	}

	return &config, nil
}

func SaveConfig(config *models.Config) error {
	configFiles := []string{"config.json", "config.yml"}

	var existingFile string
	var configType string

	for _, filename := range configFiles {
		if _, err := os.Stat(filename); err == nil {
			existingFile = filename
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".json":
				configType = "json"
			case ".yml":
				configType = "yml"
			}
			break
		}
	}

	if existingFile == "" {
		existingFile = "config.json"
		configType = "json"
	}

	viper.SetConfigName(strings.TrimSuffix(filepath.Base(existingFile), filepath.Ext(existingFile)))
	viper.SetConfigType(configType)
	viper.AddConfigPath(".")

	if config.LogLevel == "" {
		config.LogLevel = "ERROR"
	}

	viper.Set("hosts", config.Hosts)
	viper.Set("directory_mappings", config.DirectoryMappings)
	viper.Set("download_art", config.DownloadArt)
	viper.Set("show_game_details", config.ShowGameDetails)
	viper.Set("api_timeout", config.ApiTimeout)
	viper.Set("download_timeout", config.DownloadTimeout)
	viper.Set("log_level", config.LogLevel)

	gaba.SetRawLogLevel(config.LogLevel)

	newConfig := viper.AllSettings()

	pretty, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		gaba.GetLogger().Error("Failed to marshal config to JSON", "error", err)
		return err
	}

	err = os.WriteFile("config.json", pretty, 0644)
	if err != nil {
		gaba.GetLogger().Error("Failed to write config file", "error", err)
		return err
	}

	_ = os.Remove("config.yml")

	return nil
}

func GetMappedPlatforms(host models.Host, mappings map[string]models.DirectoryMapping) []romm.Platform {
	c := romm.NewClient(host.URL(), romm.WithBasicAuth(host.Username, host.Password))

	rommPlatforms, err := c.GetPlatforms()
	if err != nil {
		LogStandardFatal(fmt.Sprintf("Failed to get platforms from RomM: %s", err), nil)
	}

	var platforms []romm.Platform

	for _, platform := range rommPlatforms {
		_, exists := mappings[platform.Slug]
		if exists {
			platforms = append(platforms, romm.Platform{
				Name: platform.Name,
				ID:   platform.ID,
				Slug: platform.Slug,
			})
		}
	}

	return platforms
}

func RomMSlugToCFW(slug string) string {
	var cfwPlatformMap map[string][]string

	switch GetCFW() {
	case models.MUOS:
		cfwPlatformMap = muOSPlatforms
	case models.NEXTUI:
		cfwPlatformMap = nextUIPlatforms
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
	case models.MUOS:
		return path
	case models.NEXTUI:
		_, tag := ItemNameCleaner(path, true)
		return tag
	default:
		return path
	}
}

func ParseTag(input string) string {
	cleaned := filepath.Clean(input)

	tags := TagRegex.FindAllStringSubmatch(cleaned, -1)

	var foundTags []string
	foundTag := ""

	if len(tags) > 0 {
		for _, tagPair := range tags {
			foundTags = append(foundTags, tagPair[0])
		}

		foundTag = strings.Join(foundTags, " ")
	}

	foundTag = strings.ReplaceAll(foundTag, "(", "")
	foundTag = strings.ReplaceAll(foundTag, ")", "")

	return foundTag
}

func ItemNameCleaner(filename string, stripTag bool) (string, string) {
	cleaned := filepath.Clean(filename)

	tags := TagRegex.FindAllStringSubmatch(cleaned, -1)

	var foundTags []string
	foundTag := ""

	if len(tags) > 0 {
		for _, tagPair := range tags {
			foundTags = append(foundTags, tagPair[0])
		}

		foundTag = strings.Join(foundTags, " ")
	}

	if stripTag {
		for _, tag := range foundTags {
			cleaned = strings.ReplaceAll(cleaned, tag, "")
		}
	}

	orderedFolderRegex := OrderedFolderRegex.FindStringSubmatch(cleaned)

	if len(orderedFolderRegex) > 0 {
		cleaned = strings.ReplaceAll(cleaned, orderedFolderRegex[0], "")
	}

	cleaned = strings.ReplaceAll(cleaned, path.Ext(cleaned), "")

	cleaned = strings.TrimSpace(cleaned)

	foundTag = strings.ReplaceAll(foundTag, "(", "")
	foundTag = strings.ReplaceAll(foundTag, ")", "")

	return cleaned, foundTag
}

func IsConnectedToInternet() bool {
	timeout := 5 * time.Second
	_, err := net.DialTimeout("tcp", "8.8.8.8:53", timeout)
	return err == nil
}

func LogStandardFatal(msg string, err error) {
	log.SetOutput(os.Stderr)
	log.Fatalf("%s: %v", msg, err)
}
