package main

import (
	"grout/constants"
	"grout/constants/cfw/muos"
	"grout/resources"
	"grout/romm"
	"grout/ui"
	"grout/utils"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

func setup() *utils.Config {
	cfw := utils.GetCFW()

	// Set up input mapping for muOS with auto-detection
	if cfw == constants.MuOS && !utils.IsDevelopment() {
		if cwd, err := os.Getwd(); err == nil {
			cwdMappingPath := filepath.Join(cwd, "input_mapping.json")
			if _, err := os.Stat(cwdMappingPath); err == nil {
				// User-provided mapping takes priority
				os.Setenv("INPUT_MAPPING_PATH", cwdMappingPath)
			} else {
				// Use embedded mapping with auto-detection
				if mappingBytes, err := muos.GetInputMappingBytes(); err == nil {
					gaba.SetInputMappingBytes(mappingBytes)
				}
			}
		}
	}

	gaba.Init(gaba.Options{
		WindowTitle:          "Grout",
		PrimaryThemeColorHex: 0x007C77,
		ShowBackground:       true,
		IsNextUI:             cfw == constants.NextUI,
		LogFilename:          "grout.log",
	})

	gaba.SetLogLevel(slog.LevelDebug)

	localeFiles, err := resources.GetLocaleMessageFiles()
	if err != nil {
		utils.LogStandardFatal("Failed to load locale files", err)
	}
	if err := i18n.InitI18NFromBytes(localeFiles); err != nil {
		utils.LogStandardFatal("Failed to initialize i18n", err)
	}

	splashBytes, err := resources.GetSplashImageBytes()
	if err != nil {
		utils.LogStandardFatal("Failed to load splash image", err)
	}

	gaba.ProcessMessage("", gaba.ProcessMessageOptions{
		ImageBytes:  splashBytes,
		ImageWidth:  768,
		ImageHeight: 540,
	}, func() (interface{}, error) {
		time.Sleep(750 * time.Millisecond)
		return nil, nil
	})

	gaba.GetLogger().Debug("Loading configuration from config.json")
	config, err := utils.LoadConfig()
	isFirstLaunch := err != nil || (len(config.Hosts) == 0 && config.Language == "")

	if isFirstLaunch {
		gaba.GetLogger().Debug("First launch detected, showing language selection")
		languageScreen := ui.NewLanguageSelectionScreen()
		selectedLanguage, langErr := languageScreen.Draw()
		if langErr != nil {
			gaba.GetLogger().Error("Language selection failed", "error", langErr)
			// Default to English if selection fails
			selectedLanguage = "en"
		}
		gaba.GetLogger().Debug("Language selected", "language", selectedLanguage)

		// Set the language immediately
		if err := i18n.SetWithCode(selectedLanguage); err != nil {
			gaba.GetLogger().Error("Failed to set language", "error", err, "language", selectedLanguage)
		}

		// Update config with selected language
		if config == nil {
			config = &utils.Config{}
		}
		config.Language = selectedLanguage
	}

	if err != nil || len(config.Hosts) == 0 {
		gaba.GetLogger().Debug("No RomM Host Configured", "error", err)
		gaba.GetLogger().Debug("Starting login flow for initial setup")
		loginConfig, loginErr := ui.LoginFlow(romm.Host{})
		if loginErr != nil {
			gaba.GetLogger().Error("Login flow failed", "error", loginErr)
			utils.LogStandardFatal("Login failed", loginErr)
		}
		gaba.GetLogger().Debug("Login successful, saving configuration")
		config.Hosts = loginConfig.Hosts
		utils.SaveConfig(config)
	} else {
		gaba.GetLogger().Debug("Configuration loaded successfully", "host_count", len(config.Hosts))
	}

	if config.LogLevel != "" {
		gaba.SetRawLogLevel(config.LogLevel)
	}

	if config.Language != "" && !isFirstLaunch {
		if err := i18n.SetWithCode(config.Language); err != nil {
			gaba.GetLogger().Error("Failed to set language", "error", err, "language", config.Language)
		}
	}

	if config.DirectoryMappings == nil || len(config.DirectoryMappings) == 0 {
		screen := ui.NewPlatformMappingScreen()
		result, err := screen.Draw(ui.PlatformMappingInput{
			Host:           config.Hosts[0],
			ApiTimeout:     config.ApiTimeout,
			CFW:            cfw,
			RomDirectory:   utils.GetRomDirectory(),
			AutoSelect:     false,
			HideBackButton: true,
		})

		if err == nil && result.ExitCode == gaba.ExitCodeSuccess {
			config.DirectoryMappings = result.Value.Mappings
			utils.SaveConfig(config)
		}
	}

	gaba.GetLogger().Debug("Configuration Loaded!", "config", config.ToLoggable())
	return config
}
