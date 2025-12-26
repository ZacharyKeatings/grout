package ui

import (
	"errors"
	"grout/constants"
	"grout/romm"
	"grout/utils"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type AdvancedSettingsInput struct {
	Config                *utils.Config
	Host                  romm.Host
	LastSelectedIndex     int
	LastVisibleStartIndex int
}

type AdvancedSettingsOutput struct {
	InfoClicked           bool
	EditMappingsClicked   bool
	ClearCacheClicked     bool
	LastSelectedIndex     int
	LastVisibleStartIndex int
}

type AdvancedSettingsScreen struct{}

func NewAdvancedSettingsScreen() *AdvancedSettingsScreen {
	return &AdvancedSettingsScreen{}
}

func (s *AdvancedSettingsScreen) Draw(input AdvancedSettingsInput) (ScreenResult[AdvancedSettingsOutput], error) {
	config := input.Config
	output := AdvancedSettingsOutput{}

	items := s.buildMenuItems(config)

	result, err := gaba.OptionsList(
		i18n.GetString("settings_advanced"),
		gaba.OptionListSettings{
			FooterHelpItems: []gaba.FooterHelpItem{
				{ButtonName: "B", HelpText: i18n.GetString("button_back")},
				{ButtonName: "←→", HelpText: i18n.GetString("button_cycle")},
				{ButtonName: "Start", HelpText: i18n.GetString("button_save")},
			},
			InitialSelectedIndex: input.LastSelectedIndex,
			VisibleStartIndex:    input.LastVisibleStartIndex,
		},
		items,
	)

	if result != nil {
		output.LastSelectedIndex = result.Selected
		output.LastVisibleStartIndex = result.VisibleStartIndex
	}

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		gaba.GetLogger().Error("Advanced settings error", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	if result.Action == gaba.ListActionSelected {
		selectedText := items[result.Selected].Item.Text

		if selectedText == i18n.GetString("settings_info") {
			output.InfoClicked = true
			return withCode(output, constants.ExitCodeInfo), nil
		}

		if selectedText == i18n.GetString("settings_edit_mappings") {
			output.EditMappingsClicked = true
			return withCode(output, constants.ExitCodeEditMappings), nil
		}

		if selectedText == i18n.GetString("settings_clear_cache") {
			output.ClearCacheClicked = true
			return withCode(output, constants.ExitCodeClearCache), nil
		}
	}

	s.applySettings(config, result.Items)

	err = utils.SaveConfig(config)
	if err != nil {
		gaba.GetLogger().Error("Error saving advanced settings", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	return success(output), nil
}

func (s *AdvancedSettingsScreen) buildMenuItems(config *utils.Config) []gaba.ItemWithOptions {
	return []gaba.ItemWithOptions{
		{
			Item:    gaba.MenuItem{Text: i18n.GetString("settings_edit_mappings")},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_download_timeout")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("time_15_minutes"), Value: 15 * time.Minute},
				{DisplayName: i18n.GetString("time_30_minutes"), Value: 30 * time.Minute},
				{DisplayName: i18n.GetString("time_45_minutes"), Value: 45 * time.Minute},
				{DisplayName: i18n.GetString("time_60_minutes"), Value: 60 * time.Minute},
				{DisplayName: i18n.GetString("time_75_minutes"), Value: 75 * time.Minute},
				{DisplayName: i18n.GetString("time_90_minutes"), Value: 90 * time.Minute},
				{DisplayName: i18n.GetString("time_105_minutes"), Value: 105 * time.Minute},
				{DisplayName: i18n.GetString("time_120_minutes"), Value: 120 * time.Minute},
			},
			SelectedOption: s.findDownloadTimeoutIndex(config.DownloadTimeout),
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_api_timeout")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("time_15_seconds"), Value: 15 * time.Second},
				{DisplayName: i18n.GetString("time_30_seconds"), Value: 30 * time.Second},
				{DisplayName: i18n.GetString("time_45_seconds"), Value: 45 * time.Second},
				{DisplayName: i18n.GetString("time_60_seconds"), Value: 60 * time.Second},
				{DisplayName: i18n.GetString("time_75_seconds"), Value: 75 * time.Second},
				{DisplayName: i18n.GetString("time_90_seconds"), Value: 90 * time.Second},
				{DisplayName: i18n.GetString("time_120_seconds"), Value: 120 * time.Second},
				{DisplayName: i18n.GetString("time_180_seconds"), Value: 180 * time.Second},
				{DisplayName: i18n.GetString("time_240_seconds"), Value: 240 * time.Second},
				{DisplayName: i18n.GetString("time_300_seconds"), Value: 300 * time.Second},
			},
			SelectedOption: s.findApiTimeoutIndex(config.ApiTimeout),
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_log_level")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("log_level_debug"), Value: "DEBUG"},
				{DisplayName: i18n.GetString("log_level_error"), Value: "ERROR"},
			},
			SelectedOption: logLevelToIndex(config.LogLevel),
		},
		{
			Item:    gaba.MenuItem{Text: i18n.GetString("settings_clear_cache")},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		},
		{
			Item:    gaba.MenuItem{Text: i18n.GetString("settings_info")},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		},
	}
}

func (s *AdvancedSettingsScreen) applySettings(config *utils.Config, items []gaba.ItemWithOptions) {
	for _, item := range items {
		selectedText := item.Item.Text

		switch selectedText {
		case i18n.GetString("settings_download_timeout"):
			if val, ok := item.Options[item.SelectedOption].Value.(time.Duration); ok {
				config.DownloadTimeout = val
			}

		case i18n.GetString("settings_api_timeout"):
			if val, ok := item.Options[item.SelectedOption].Value.(time.Duration); ok {
				config.ApiTimeout = val
			}

		case i18n.GetString("settings_log_level"):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.LogLevel = val
			}
		}
	}
}

func (s *AdvancedSettingsScreen) findDownloadTimeoutIndex(timeout time.Duration) int {
	timeouts := []time.Duration{
		15 * time.Minute,
		30 * time.Minute,
		45 * time.Minute,
		60 * time.Minute,
		75 * time.Minute,
		90 * time.Minute,
		105 * time.Minute,
		120 * time.Minute,
	}
	for i, t := range timeouts {
		if t == timeout {
			return i
		}
	}
	return 0 // Default to 15 minutes
}

func (s *AdvancedSettingsScreen) findApiTimeoutIndex(timeout time.Duration) int {
	timeouts := []time.Duration{
		15 * time.Second,
		30 * time.Second,
		45 * time.Second,
		60 * time.Second,
		75 * time.Second,
		90 * time.Second,
		120 * time.Second,
		180 * time.Second,
		240 * time.Second,
		300 * time.Second,
	}
	for i, t := range timeouts {
		if t == timeout {
			return i
		}
	}
	return 0 // Default to 15 seconds
}
