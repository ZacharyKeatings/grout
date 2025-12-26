package main

import (
	"grout/romm"
	"grout/ui"
	"grout/utils"
	"os"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	_ "github.com/UncleJunVIP/certifiable"
)

func main() {
	defer cleanup()

	config := setup()

	logger := gaba.GetLogger()
	logger.Debug("Starting Grout")

	cfw := utils.GetCFW()

	quitOnBack := len(config.Hosts) == 1

	var platforms []romm.Platform
	var err error

	for {
		logger.Debug("Validating connection to RomM server")
		errorMsgKey := validateConnection(config.Hosts[0])

		if errorMsgKey == "" {
			platforms, err = utils.GetMappedPlatforms(config.Hosts[0], config.DirectoryMappings)
			if err != nil {
				errorMsgKey = classifyConnectionError(err)
			} else {
				break
			}
		}

		logger.Error("Failed to connect to RomM", "error_key", errorMsgKey)

		result, optErr := gaba.OptionsList(
			i18n.GetString(errorMsgKey),
			gaba.OptionListSettings{
				DisableBackButton: true,
				FooterHelpItems: []gaba.FooterHelpItem{
					{ButtonName: "←→", HelpText: i18n.GetString("button_cycle")},
					{ButtonName: "A", HelpText: i18n.GetString("button_select")},
				},
			},
			[]gaba.ItemWithOptions{
				{
					Item: gaba.MenuItem{Text: i18n.GetString("startup_error_action_retry")},
					Options: []gaba.Option{
						{DisplayName: i18n.GetString("startup_error_action_retry"), Value: "retry"},
					},
				},
				{
					Item: gaba.MenuItem{Text: i18n.GetString("startup_error_action_relogin")},
					Options: []gaba.Option{
						{DisplayName: i18n.GetString("startup_error_action_relogin"), Value: "relogin"},
					},
				},
				{
					Item: gaba.MenuItem{Text: i18n.GetString("startup_error_action_exit")},
					Options: []gaba.Option{
						{DisplayName: i18n.GetString("startup_error_action_exit"), Value: "exit"},
					},
				},
			},
		)

		if optErr != nil {
			logger.Error("Options list error", "error", optErr)
			os.Exit(1)
		}

		action := result.Items[result.Selected].Value().(string)

		switch action {
		case "retry":
			continue
		case "relogin":
			loginConfig, loginErr := ui.LoginFlow(config.Hosts[0])
			if loginErr != nil {
				logger.Error("Login flow failed", "error", loginErr)
				os.Exit(1)
			}
			config = loginConfig
			utils.SaveConfig(config)
			continue
		case "exit":
			os.Exit(1)
		}
	}

	platforms = utils.SortPlatformsByOrder(platforms, config.PlatformOrder)

	showCollections := utils.ShowCollections(config, config.Hosts[0])

	fsm := buildFSM(config, cfw, platforms, quitOnBack, showCollections)

	if err := fsm.Run(); err != nil {
		logger.Error("FSM error", "error", err)
	}
}

func cleanup() {
	if err := os.RemoveAll(".tmp"); err != nil {
		gaba.GetLogger().Error("Failed to clean .tmp directory", "error", err)
	}
	gaba.Close()
}
