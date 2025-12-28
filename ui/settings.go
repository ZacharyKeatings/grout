package ui

import (
	"errors"
	"grout/constants"
	"grout/romm"
	"grout/utils"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type SettingsInput struct {
	Config                *utils.Config
	CFW                   constants.CFW
	Host                  romm.Host
	LastSelectedIndex     int
	LastVisibleStartIndex int
}

type SettingsOutput struct {
	Config                     *utils.Config
	EditMappingsClicked        bool
	InfoClicked                bool
	SyncArtworkClicked         bool
	CollectionsSettingsClicked bool
	AdvancedSettingsClicked    bool
	LastSelectedIndex          int
	LastVisibleStartIndex      int
}

type SettingsScreen struct{}

func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{}
}

type SettingType string

const (
	SettingEditMappings        SettingType = "edit_mappings"
	SettingGameDetails         SettingType = "game_details"
	SettingCollections         SettingType = "collections"
	SettingSmartCollections    SettingType = "smart_collections"
	SettingVirtualCollections  SettingType = "virtual_collections"
	SettingCollectionView      SettingType = "collection_view"
	SettingCollectionsSettings SettingType = "collections_settings"
	SettingAdvancedSettings    SettingType = "advanced_settings"
	SettingDownloadedGames     SettingType = "downloaded_games"
	SettingSaveSync            SettingType = "save_sync"
	SettingShowBIOS            SettingType = "show_bios"
	SettingDownloadArt         SettingType = "download_art"
	SettingBoxArt              SettingType = "box_art"
	SettingSyncArtwork         SettingType = "sync_artwork"
	SettingUnzipDownloads      SettingType = "unzip_downloads"
	SettingDownloadTimeout     SettingType = "download_timeout"
	SettingAPITimeout          SettingType = "api_timeout"
	SettingLanguage            SettingType = "language"
	SettingLogLevel            SettingType = "log_level"
	SettingInfo                SettingType = "info"
)

var settingsOrder = []SettingType{

	SettingBoxArt,
	SettingGameDetails,
	SettingShowBIOS,

	SettingDownloadedGames,
	SettingDownloadArt,
	SettingUnzipDownloads,

	SettingSaveSync,

	SettingCollectionsSettings,
	SettingSyncArtwork,

	SettingLanguage,
	SettingAdvancedSettings,
}

var (
	apiTimeoutOptions = []struct {
		Message *goi18n.Message
		Value   time.Duration
	}{
		{&goi18n.Message{ID: "time_15_seconds", Other: "15 Seconds"}, 15 * time.Second},
		{&goi18n.Message{ID: "time_30_seconds", Other: "30 Seconds"}, 30 * time.Second},
		{&goi18n.Message{ID: "time_45_seconds", Other: "45 Seconds"}, 45 * time.Second},
		{&goi18n.Message{ID: "time_60_seconds", Other: "60 Seconds"}, 60 * time.Second},
		{&goi18n.Message{ID: "time_75_seconds", Other: "75 Seconds"}, 75 * time.Second},
		{&goi18n.Message{ID: "time_90_seconds", Other: "90 Seconds"}, 90 * time.Second},
		{&goi18n.Message{ID: "time_120_seconds", Other: "120 Seconds"}, 120 * time.Second},
		{&goi18n.Message{ID: "time_180_seconds", Other: "180 Seconds"}, 180 * time.Second},
		{&goi18n.Message{ID: "time_240_seconds", Other: "240 Seconds"}, 240 * time.Second},
		{&goi18n.Message{ID: "time_300_seconds", Other: "300 Seconds"}, 300 * time.Second},
	}

	downloadTimeoutOptions = []struct {
		Message *goi18n.Message
		Value   time.Duration
	}{
		{&goi18n.Message{ID: "time_15_minutes", Other: "15 Minutes"}, 15 * time.Minute},
		{&goi18n.Message{ID: "time_30_minutes", Other: "30 Minutes"}, 30 * time.Minute},
		{&goi18n.Message{ID: "time_45_minutes", Other: "45 Minutes"}, 45 * time.Minute},
		{&goi18n.Message{ID: "time_60_minutes", Other: "60 Minutes"}, 60 * time.Minute},
		{&goi18n.Message{ID: "time_75_minutes", Other: "75 Minutes"}, 75 * time.Minute},
		{&goi18n.Message{ID: "time_90_minutes", Other: "90 Minutes"}, 90 * time.Minute},
		{&goi18n.Message{ID: "time_105_minutes", Other: "105 Minutes"}, 105 * time.Minute},
		{&goi18n.Message{ID: "time_120_minutes", Other: "120 Minutes"}, 120 * time.Minute},
	}
)

func (s *SettingsScreen) Draw(input SettingsInput) (ScreenResult[SettingsOutput], error) {
	config := input.Config
	output := SettingsOutput{Config: config}

	items := s.buildMenuItems(config)

	result, err := gaba.OptionsList(
		i18n.Localize(&goi18n.Message{ID: "settings_title", Other: "Settings"}, nil),
		gaba.OptionListSettings{
			FooterHelpItems: []gaba.FooterHelpItem{
				{ButtonName: "B", HelpText: i18n.Localize(&goi18n.Message{ID: "button_cancel", Other: "Cancel"}, nil)},
				{ButtonName: icons.LeftRight, HelpText: i18n.Localize(&goi18n.Message{ID: "button_cycle", Other: "Cycle"}, nil)},
				{ButtonName: icons.Start, HelpText: i18n.Localize(&goi18n.Message{ID: "button_save", Other: "Save"}, nil)},
			},
			InitialSelectedIndex: input.LastSelectedIndex,
			VisibleStartIndex:    input.LastVisibleStartIndex,
			StatusBar:            utils.StatusBar(),
			SmallTitle:           true,
		},
		items,
	)

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(SettingsOutput{}), nil
		}
		return withCode(SettingsOutput{}, gaba.ExitCodeError), err
	}

	output.LastSelectedIndex = result.Selected
	output.LastVisibleStartIndex = result.VisibleStartIndex

	if result.Action == gaba.ListActionSelected {
		selectedText := items[result.Selected].Item.Text

		if selectedText == i18n.Localize(&goi18n.Message{ID: "settings_sync_artwork", Other: "Cache Artwork"}, nil) {
			output.SyncArtworkClicked = true
			return withCode(output, constants.ExitCodeSyncArtwork), nil
		}

		if selectedText == i18n.Localize(&goi18n.Message{ID: "settings_edit_mappings", Other: "Directory Mappings"}, nil) {
			output.EditMappingsClicked = true
			return withCode(output, constants.ExitCodeEditMappings), nil
		}

		if selectedText == i18n.Localize(&goi18n.Message{ID: "settings_info", Other: "Grout Info"}, nil) {
			output.InfoClicked = true
			return withCode(output, constants.ExitCodeInfo), nil
		}

		if selectedText == i18n.Localize(&goi18n.Message{ID: "settings_collections", Other: "Collections Settings"}, nil) {
			output.CollectionsSettingsClicked = true
			return withCode(output, constants.ExitCodeCollectionsSettings), nil
		}

		if selectedText == i18n.Localize(&goi18n.Message{ID: "settings_advanced", Other: "Advanced"}, nil) {
			output.AdvancedSettingsClicked = true
			return withCode(output, constants.ExitCodeAdvancedSettings), nil
		}
	}

	s.applySettings(config, result.Items)

	output.Config = config
	return success(output), nil
}

func (s *SettingsScreen) buildMenuItems(config *utils.Config) []gaba.ItemWithOptions {
	items := make([]gaba.ItemWithOptions, 0, len(settingsOrder))
	for _, settingType := range settingsOrder {
		items = append(items, s.buildMenuItem(settingType, config))
	}
	return items
}

func (s *SettingsScreen) buildMenuItem(settingType SettingType, config *utils.Config) gaba.ItemWithOptions {
	switch settingType {
	case SettingEditMappings:
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_edit_mappings", Other: "Directory Mappings"}, nil)},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}

	case SettingGameDetails:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_show_game_details", Other: "Game Details"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.GameDetails),
		}

	case SettingCollections:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_show_collections", Other: "Collections"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowCollections),
		}

	case SettingSmartCollections:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_show_smart_collections", Other: "Smart Collections"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowSmartCollections),
		}

	case SettingVirtualCollections:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_show_virtual_collections", Other: "Virtual Collections"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowVirtualCollections),
		}

	case SettingCollectionView:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_collection_view", Other: "Collection View"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "collection_view_platform", Other: "Platform"}, nil), Value: "platform"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "collection_view_unified", Other: "Unified"}, nil), Value: "unified"},
			},
			SelectedOption: collectionViewToIndex(config.CollectionView),
		}

	case SettingCollectionsSettings:
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_collections", Other: "Collections Settings"}, nil)},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}

	case SettingDownloadedGames:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_downloaded_games", Other: "Downloaded Games"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "downloaded_games_do_nothing", Other: "Do Nothing"}, nil), Value: "do_nothing"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "downloaded_games_mark", Other: "Mark"}, nil), Value: "mark"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "downloaded_games_filter", Other: "Filter"}, nil), Value: "filter"},
			},
			SelectedOption: s.downloadedGamesActionToIndex(config.DownloadedGames),
		}

	case SettingSaveSync:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_save_sync", Other: "Save Sync"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "save_sync_mode_off", Other: "Off"}, nil), Value: "off"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "save_sync_mode_manual", Other: "Manual"}, nil), Value: "manual"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "save_sync_mode_automatic", Other: "Automatic"}, nil), Value: "automatic"},
				// {DisplayName: i18n.Localize(&goi18n.Message{ID: "save_sync_mode_daemon", Other: "Daemon"}, nil), Value: "daemon"},
			},
			SelectedOption: saveSyncModeToIndex(config.SaveSyncMode),
		}

	case SettingShowBIOS:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_show_bios", Other: "BIOS Menu"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowBIOSDownload),
		}

	case SettingDownloadArt:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_download_art", Other: "Download Art"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_true", Other: "True"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_false", Other: "False"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.DownloadArt),
		}

	case SettingBoxArt:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_box_art", Other: "Box Art"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_show", Other: "Show"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_hide", Other: "Hide"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowBoxArt),
		}

	case SettingSyncArtwork:
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_sync_artwork", Other: "Cache Artwork"}, nil)},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}

	case SettingUnzipDownloads:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_unzip_downloads", Other: "Unzip Downloads"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_true", Other: "True"}, nil), Value: true},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "common_false", Other: "False"}, nil), Value: false},
			},
			SelectedOption: boolToIndex(!config.UnzipDownloads),
		}

	case SettingDownloadTimeout:
		return gaba.ItemWithOptions{
			Item:           gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_download_timeout", Other: "Download Timeout"}, nil)},
			Options:        s.buildDownloadTimeoutOptions(),
			SelectedOption: s.findDownloadTimeoutIndex(config.DownloadTimeout),
		}

	case SettingAPITimeout:
		return gaba.ItemWithOptions{
			Item:           gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_api_timeout", Other: "API Timeout"}, nil)},
			Options:        s.buildApiTimeoutOptions(),
			SelectedOption: s.findApiTimeoutIndex(config.ApiTimeout),
		}

	case SettingLanguage:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_language", Other: "Language"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_english", Other: "English"}, nil), Value: "en"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_german", Other: "Deutsch"}, nil), Value: "de"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_spanish", Other: "Español"}, nil), Value: "es"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_french", Other: "Français"}, nil), Value: "fr"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_italian", Other: "Italiano"}, nil), Value: "it"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_portuguese", Other: "Português"}, nil), Value: "pt"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_russian", Other: "Русский"}, nil), Value: "ru"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "settings_language_japanese", Other: "日本語"}, nil), Value: "ja"},
			},
			SelectedOption: languageToIndex(config.Language),
		}

	case SettingLogLevel:
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_log_level", Other: "Log Level"}, nil)},
			Options: []gaba.Option{
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "log_level_debug", Other: "Debug"}, nil), Value: "DEBUG"},
				{DisplayName: i18n.Localize(&goi18n.Message{ID: "log_level_error", Other: "Error"}, nil), Value: "ERROR"},
			},
			SelectedOption: logLevelToIndex(config.LogLevel),
		}

	case SettingInfo:
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_info", Other: "Grout Info"}, nil)},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}

	case SettingAdvancedSettings:
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "settings_advanced", Other: "Advanced"}, nil)},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}

	default:
		// Should never happen, but return a safe default
		return gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: "Unknown Setting"},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		}
	}
}

func (s *SettingsScreen) buildApiTimeoutOptions() []gaba.Option {
	options := make([]gaba.Option, len(apiTimeoutOptions))
	for i, opt := range apiTimeoutOptions {
		options[i] = gaba.Option{DisplayName: i18n.Localize(opt.Message, nil), Value: opt.Value}
	}
	return options
}

func (s *SettingsScreen) buildDownloadTimeoutOptions() []gaba.Option {
	options := make([]gaba.Option, len(downloadTimeoutOptions))
	for i, opt := range downloadTimeoutOptions {
		options[i] = gaba.Option{DisplayName: i18n.Localize(opt.Message, nil), Value: opt.Value}
	}
	return options
}

func (s *SettingsScreen) findApiTimeoutIndex(timeout time.Duration) int {
	for i, opt := range apiTimeoutOptions {
		if opt.Value == timeout {
			return i
		}
	}
	return 0
}

func (s *SettingsScreen) findDownloadTimeoutIndex(timeout time.Duration) int {
	for i, opt := range downloadTimeoutOptions {
		if opt.Value == timeout {
			return i
		}
	}
	return 0
}

func (s *SettingsScreen) applySettings(config *utils.Config, items []gaba.ItemWithOptions) {
	for _, item := range items {
		text := item.Item.Text
		switch text {
		case i18n.Localize(&goi18n.Message{ID: "settings_download_art", Other: "Download Art"}, nil):
			config.DownloadArt = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_box_art", Other: "Box Art"}, nil):
			config.ShowBoxArt = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_auto_sync_saves", Other: "Auto Sync Saves"}, nil):
			config.AutoSyncSaves = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_save_sync", Other: "Save Sync"}, nil):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.SaveSyncMode = val
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_show_bios", Other: "BIOS Menu"}, nil):
			config.ShowBIOSDownload = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_unzip_downloads", Other: "Unzip Downloads"}, nil):
			config.UnzipDownloads = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_show_game_details", Other: "Game Details"}, nil):
			config.GameDetails = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_show_collections", Other: "Collections"}, nil):
			config.ShowCollections = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_show_smart_collections", Other: "Smart Collections"}, nil):
			config.ShowSmartCollections = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_show_virtual_collections", Other: "Virtual Collections"}, nil):
			config.ShowVirtualCollections = item.SelectedOption == 0
		case i18n.Localize(&goi18n.Message{ID: "settings_api_timeout", Other: "API Timeout"}, nil):
			idx := item.SelectedOption
			if idx < len(apiTimeoutOptions) {
				config.ApiTimeout = apiTimeoutOptions[idx].Value
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_download_timeout", Other: "Download Timeout"}, nil):
			idx := item.SelectedOption
			if idx < len(downloadTimeoutOptions) {
				config.DownloadTimeout = downloadTimeoutOptions[idx].Value
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_log_level", Other: "Log Level"}, nil):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.LogLevel = val
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_language", Other: "Language"}, nil):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.Language = val
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_downloaded_games", Other: "Downloaded Games"}, nil):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.DownloadedGames = val
			}
		case i18n.Localize(&goi18n.Message{ID: "settings_collection_view", Other: "Collection View"}, nil):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.CollectionView = val
			}
		}
	}
}

func boolToIndex(b bool) int {
	if b {
		return 1
	}
	return 0
}

func logLevelToIndex(level string) int {
	switch level {
	case "DEBUG":
		return 0
	case "ERROR":
		return 1
	default:
		return 0
	}
}

func languageToIndex(lang string) int {
	switch lang {
	case "en":
		return 0
	case "de":
		return 1
	case "es":
		return 2
	case "fr":
		return 3
	case "it":
		return 4
	case "pt":
		return 5
	case "ru":
		return 6
	case "ja":
		return 7
	default:
		return 0
	}
}

func saveSyncModeToIndex(mode string) int {
	switch mode {
	case "off":
		return 0
	case "manual":
		return 1
	case "automatic":
		return 2
	// case "daemon":
	// 	return 3
	default:
		return 0
	}
}

func (s *SettingsScreen) downloadedGamesActionToIndex(action string) int {
	switch action {
	case "do_nothing":
		return 0
	case "mark":
		return 1
	case "filter":
		return 2
	default:
		return 0
	}
}

func collectionViewToIndex(view string) int {
	switch view {
	case "platform":
		return 0
	case "unified":
		return 1
	default:
		return 0
	}
}
