package ui

import (
	"errors"
	"grout/utils"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type CollectionsSettingsInput struct {
	Config *utils.Config
}

type CollectionsSettingsOutput struct{}

type CollectionsSettingsScreen struct{}

func NewCollectionsSettingsScreen() *CollectionsSettingsScreen {
	return &CollectionsSettingsScreen{}
}

func (s *CollectionsSettingsScreen) Draw(input CollectionsSettingsInput) (ScreenResult[CollectionsSettingsOutput], error) {
	config := input.Config
	output := CollectionsSettingsOutput{}

	items := s.buildMenuItems(config)

	result, err := gaba.OptionsList(
		i18n.GetString("settings_collections"),
		gaba.OptionListSettings{
			FooterHelpItems: []gaba.FooterHelpItem{
				{ButtonName: "B", HelpText: i18n.GetString("button_back")},
				{ButtonName: "←→", HelpText: i18n.GetString("button_cycle")},
				{ButtonName: "Start", HelpText: i18n.GetString("button_save")},
			},
			InitialSelectedIndex: 0,
		},
		items,
	)

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		gaba.GetLogger().Error("Collections settings error", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	s.applySettings(config, result.Items)

	err = utils.SaveConfig(config)
	if err != nil {
		gaba.GetLogger().Error("Error saving collections settings", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	return success(output), nil
}

func (s *CollectionsSettingsScreen) buildMenuItems(config *utils.Config) []gaba.ItemWithOptions {
	return []gaba.ItemWithOptions{
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_show_collections")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("common_show"), Value: true},
				{DisplayName: i18n.GetString("common_hide"), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowCollections),
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_show_smart_collections")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("common_show"), Value: true},
				{DisplayName: i18n.GetString("common_hide"), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowSmartCollections),
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_show_virtual_collections")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("common_show"), Value: true},
				{DisplayName: i18n.GetString("common_hide"), Value: false},
			},
			SelectedOption: boolToIndex(!config.ShowVirtualCollections),
		},
		{
			Item: gaba.MenuItem{Text: i18n.GetString("settings_collection_view")},
			Options: []gaba.Option{
				{DisplayName: i18n.GetString("collection_view_platform"), Value: "platform"},
				{DisplayName: i18n.GetString("collection_view_unified"), Value: "unified"},
			},
			SelectedOption: collectionViewToIndex(config.CollectionView),
		},
	}
}

func (s *CollectionsSettingsScreen) applySettings(config *utils.Config, items []gaba.ItemWithOptions) {
	for _, item := range items {
		selectedText := item.Item.Text

		switch selectedText {
		case i18n.GetString("settings_show_collections"):
			if val, ok := item.Options[item.SelectedOption].Value.(bool); ok {
				config.ShowCollections = val
			}

		case i18n.GetString("settings_show_smart_collections"):
			if val, ok := item.Options[item.SelectedOption].Value.(bool); ok {
				config.ShowSmartCollections = val
			}

		case i18n.GetString("settings_show_virtual_collections"):
			if val, ok := item.Options[item.SelectedOption].Value.(bool); ok {
				config.ShowVirtualCollections = val
			}

		case i18n.GetString("settings_collection_view"):
			if val, ok := item.Options[item.SelectedOption].Value.(string); ok {
				config.CollectionView = val
			}
		}
	}
}
