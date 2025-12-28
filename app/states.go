package main

import (
	"fmt"
	"grout/constants"
	"grout/romm"
	"grout/ui"
	"grout/utils"
	"sync"
	"sync/atomic"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

var (
	autoSync     *utils.AutoSync
	autoSyncOnce sync.Once
)

type appStartTime struct {
	time.Time
	logged bool
	mu     sync.Mutex
}

const (
	platformSelection           gaba.StateName = "platform_selection"
	gameList                                   = "game_list"
	gameDetails                                = "game_details"
	collectionList                             = "collection_list"
	collectionPlatformSelection                = "collection_platform_selection"
	search                                     = "search"
	collectionSearch                           = "collection_search"
	settings                                   = "settings"
	collectionsSettings                        = "collections_settings"
	advancedSettings                           = "advanced_settings"
	settingsPlatformMapping                    = "platform_mapping"
	info                                       = "info"
	logoutConfirmation                         = "logout_confirmation"
	clearCacheConfirmation                     = "clear_cache_confirmation"
	saveSync                                   = "save_sync"
	syncIssues                                 = "sync_issues"
	biosDownload                               = "bios_download"
	artworkSync                                = "artwork_sync"
)

type (
	currentGamesList       []romm.Rom
	fullGamesList          []romm.Rom
	searchFilterString     string
	collectionSearchFilter string
	quitOnBackBool         bool

	showCollectionsBool bool

	gameListPosition struct {
		Index int
		Pos   int
	}

	platformListPosition struct {
		Index int
		Pos   int
	}

	collectionListPosition struct {
		Index int
		Pos   int
	}

	collectionPlatformListPosition struct {
		Index int
		Pos   int
	}

	settingsPosition struct {
		Index             int
		VisibleStartIndex int
	}

	collectionsSettingsPosition struct {
		Index int
	}

	advancedSettingsPosition struct {
		Index             int
		VisibleStartIndex int
	}

	infoPreviousState gaba.StateName

	cachedCollectionGames []romm.Rom
)

func buildFSM(config *utils.Config, cfw constants.CFW, platforms []romm.Platform, quitOnBack bool, showCollections bool, appStart time.Time) *gaba.FSM {
	fsm := gaba.NewFSM()

	gaba.Set(fsm.Context(), config)
	gaba.Set(fsm.Context(), cfw)
	gaba.Set(fsm.Context(), config.Hosts[0])
	gaba.Set(fsm.Context(), platforms)
	gaba.Set(fsm.Context(), quitOnBackBool(quitOnBack))
	gaba.Set(fsm.Context(), showCollectionsBool(showCollections))
	gaba.Set(fsm.Context(), searchFilterString(""))
	gaba.Set(fsm.Context(), &appStartTime{Time: appStart})

	gaba.AddState(fsm, platformSelection, func(ctx *gaba.Context) (ui.PlatformSelectionOutput, gaba.ExitCode) {
		platforms, _ := gaba.Get[[]romm.Platform](ctx)
		quitOnBack, _ := gaba.Get[quitOnBackBool](ctx)
		showCollections, _ := gaba.Get[showCollectionsBool](ctx)
		platPos, _ := gaba.Get[platformListPosition](ctx)

		// Log time to first platform menu draw (only once)
		if startTime, ok := gaba.Get[*appStartTime](ctx); ok && startTime != nil {
			startTime.mu.Lock()
			if !startTime.logged {
				gaba.GetLogger().Info("Time to platform menu", "seconds", fmt.Sprintf("%.2f", time.Since(startTime.Time).Seconds()))
				startTime.logged = true
			}
			startTime.mu.Unlock()
		}

		screen := ui.NewPlatformSelectionScreen()
		config, _ := gaba.Get[*utils.Config](ctx)

		// Start auto-sync on first platform menu view
		if config.SaveSyncMode == "automatic" {
			autoSyncOnce.Do(func() {
				host, _ := gaba.Get[romm.Host](ctx)
				autoSync = utils.NewAutoSync(host)
				utils.AddIcon(autoSync.Icon())
				autoSync.Start()
			})
		}

		// Determine the sync button visibility control
		// - "off": nil (never show)
		// - "manual": always true, shows "Sync" button
		// - "automatic": controlled by auto-sync (shows "Issues" when there are issues)
		var showSaveSync *atomic.Bool
		showSyncIssues := false
		switch config.SaveSyncMode {
		case "manual":
			showSaveSync = &atomic.Bool{}
			showSaveSync.Store(true)
		case "automatic":
			if autoSync != nil {
				showSaveSync = autoSync.ShowButton()
				showSyncIssues = true
			}
		}

		result, err := screen.Draw(ui.PlatformSelectionInput{
			Platforms:            platforms,
			QuitOnBack:           bool(quitOnBack),
			ShowCollections:      bool(showCollections),
			ShowSaveSync:         showSaveSync,
			ShowSyncIssues:       showSyncIssues,
			LastSelectedIndex:    platPos.Index,
			LastSelectedPosition: platPos.Pos,
		})

		if err != nil {
			return ui.PlatformSelectionOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, platformListPosition{
			Index: result.Value.LastSelectedIndex,
			Pos:   result.Value.LastSelectedPosition,
		})

		if len(result.Value.ReorderedPlatforms) > 0 {
			config, _ := gaba.Get[*utils.Config](ctx)

			var platformOrder []string
			for _, p := range result.Value.ReorderedPlatforms {
				platformOrder = append(platformOrder, p.Slug)
			}

			gaba.GetLogger().Debug("Saving platform order to config", "order", platformOrder)

			config.PlatformOrder = platformOrder
			if err := utils.SaveConfig(config); err != nil {
				gaba.GetLogger().Error("Failed to save platform order", "error", err)
			} else {
				gaba.GetLogger().Info("Platform order saved successfully", "order", platformOrder)
			}

			gaba.Set(ctx, result.Value.ReorderedPlatforms)
			gaba.GetLogger().Debug("Updated platforms in context")
		}

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, gameList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, searchFilterString(""))
			gaba.Set(ctx, currentGamesList(nil))
			gaba.Set(ctx, gameListPosition{Index: 0, Pos: 0})
			gaba.Set(ctx, ui.CollectionSelectionOutput{})
			return nil
		}).
		OnWithHook(constants.ExitCodeCollections, collectionList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, collectionListPosition{
				Index: 0,
				Pos:   0,
			})
			return nil
		}).
		On(gaba.ExitCodeAction, settings).
		On(constants.ExitCodeSaveSync, saveSync).
		On(constants.ExitCodeSyncIssues, syncIssues).
		Exit(gaba.ExitCodeQuit)

	gaba.AddState(fsm, collectionList, func(ctx *gaba.Context) (ui.CollectionSelectionOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		colPos, _ := gaba.Get[collectionListPosition](ctx)
		searchFilter, _ := gaba.Get[collectionSearchFilter](ctx)

		screen := ui.NewCollectionSelectionScreen()
		result, err := screen.Draw(ui.CollectionSelectionInput{
			Config:               config,
			Host:                 host,
			SearchFilter:         string(searchFilter),
			LastSelectedIndex:    colPos.Index,
			LastSelectedPosition: colPos.Pos,
		})

		if err != nil {
			return ui.CollectionSelectionOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, collectionListPosition{
			Index: result.Value.LastSelectedIndex,
			Pos:   result.Value.LastSelectedPosition,
		})
		gaba.Set(ctx, collectionSearchFilter(result.Value.SearchFilter))

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, collectionPlatformSelection, func(ctx *gaba.Context) error {
			gaba.Set(ctx, searchFilterString(""))
			gaba.Set(ctx, currentGamesList(nil))
			gaba.Set(ctx, gameListPosition{Index: 0, Pos: 0})
			gaba.Set(ctx, collectionPlatformListPosition{Index: 0, Pos: 0})
			gaba.Set(ctx, cachedCollectionGames(nil))
			gaba.Set(ctx, ui.PlatformSelectionOutput{})
			return nil
		}).
		On(constants.ExitCodeSearch, collectionSearch).
		OnWithHook(constants.ExitCodeClearSearch, collectionList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, collectionSearchFilter(""))
			gaba.Set(ctx, collectionListPosition{Index: 0, Pos: 0})
			return nil
		}).
		On(gaba.ExitCodeBack, platformSelection)

	gaba.AddState(fsm, collectionPlatformSelection, func(ctx *gaba.Context) (ui.CollectionPlatformSelectionOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		collection, _ := gaba.Get[ui.CollectionSelectionOutput](ctx)
		pos, _ := gaba.Get[collectionPlatformListPosition](ctx)
		cachedGames, _ := gaba.Get[cachedCollectionGames](ctx)

		screen := ui.NewCollectionPlatformSelectionScreen()
		result, err := screen.Draw(ui.CollectionPlatformSelectionInput{
			Config:               config,
			Host:                 host,
			Collection:           collection.SelectedCollection,
			CachedGames:          cachedGames,
			LastSelectedIndex:    pos.Index,
			LastSelectedPosition: pos.Pos,
		})

		if err != nil {
			return ui.CollectionPlatformSelectionOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, collectionPlatformListPosition{
			Index: result.Value.LastSelectedIndex,
			Pos:   result.Value.LastSelectedPosition,
		})

		gaba.Set(ctx, cachedCollectionGames(result.Value.AllGames))

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, gameList, func(ctx *gaba.Context) error {
			output, _ := gaba.Get[ui.CollectionPlatformSelectionOutput](ctx)
			config, _ := gaba.Get[*utils.Config](ctx)

			var finalGames []romm.Rom

			// In unified mode with Platform.ID == 0, use all games
			if config.CollectionView == "unified" && output.SelectedPlatform.ID == 0 {
				finalGames = output.AllGames
			} else {
				// Platform mode: filter by selected platform
				filteredGames := make([]romm.Rom, 0)
				for _, game := range output.AllGames {
					if game.PlatformID == output.SelectedPlatform.ID {
						filteredGames = append(filteredGames, game)
					}
				}
				finalGames = filteredGames
			}

			gaba.Set(ctx, searchFilterString(""))
			gaba.Set(ctx, fullGamesList(finalGames))
			gaba.Set(ctx, currentGamesList(finalGames))
			gaba.Set(ctx, gameListPosition{Index: 0, Pos: 0})
			return nil
		}).
		On(gaba.ExitCodeBack, collectionList)

	gaba.AddState(fsm, gameList, func(ctx *gaba.Context) (ui.GameListOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		platform, _ := gaba.Get[ui.PlatformSelectionOutput](ctx)
		collection, _ := gaba.Get[ui.CollectionSelectionOutput](ctx)
		collectionPlatform, _ := gaba.Get[ui.CollectionPlatformSelectionOutput](ctx)
		games, _ := gaba.Get[currentGamesList](ctx)
		filter, _ := gaba.Get[searchFilterString](ctx)
		pos, _ := gaba.Get[gameListPosition](ctx)

		var selectedPlatform romm.Platform
		var selectedCollection romm.Collection

		if collectionPlatform.SelectedPlatform.ID != 0 {
			selectedPlatform = collectionPlatform.SelectedPlatform
			selectedCollection = collectionPlatform.Collection
		} else {
			selectedPlatform = platform.SelectedPlatform
			selectedCollection = collection.SelectedCollection
		}

		screen := ui.NewGameListScreen()
		result, err := screen.Draw(ui.GameListInput{
			Config:               config,
			Host:                 host,
			Platform:             selectedPlatform,
			Collection:           selectedCollection,
			Games:                games,
			SearchFilter:         string(filter),
			LastSelectedIndex:    pos.Index,
			LastSelectedPosition: pos.Pos,
		})

		if err != nil {
			return ui.GameListOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, fullGamesList(result.Value.AllGames))
		gaba.Set(ctx, currentGamesList(result.Value.AllGames))
		gaba.Set(ctx, gameListPosition{
			Index: result.Value.LastSelectedIndex,
			Pos:   result.Value.LastSelectedPosition,
		})
		gaba.Set(ctx, searchFilterString(result.Value.SearchFilter))

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeSuccess, gameDetails).
		On(constants.ExitCodeSearch, search).
		On(constants.ExitCodeBIOS, biosDownload).
		OnWithHook(constants.ExitCodeClearSearch, gameList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, searchFilterString(""))
			fullGames, _ := gaba.Get[fullGamesList](ctx)
			gaba.Set(ctx, currentGamesList(fullGames))
			gaba.Set(ctx, gameListPosition{Index: 0, Pos: 0})
			return nil
		}).
		OnWithHook(gaba.ExitCodeBack, platformSelection, func(ctx *gaba.Context) error {
			gaba.Set(ctx, currentGamesList(nil))
			return nil
		}).
		On(constants.ExitCodeBackToCollectionPlatform, collectionPlatformSelection).
		OnWithHook(constants.ExitCodeBackToCollection, collectionList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, currentGamesList(nil))
			return nil
		}).
		On(constants.ExitCodeNoResults, search)

	gaba.AddState(fsm, gameDetails, func(ctx *gaba.Context) (ui.GameDetailsOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		gameListOutput, _ := gaba.Get[ui.GameListOutput](ctx)

		if !config.GameDetails || len(gameListOutput.SelectedGames) != 1 {
			filter, _ := gaba.Get[searchFilterString](ctx)
			downloadScreen := ui.NewDownloadScreen()
			downloadOutput := downloadScreen.Execute(*config, host, gameListOutput.Platform, gameListOutput.SelectedGames, gameListOutput.AllGames, string(filter))
			gaba.Set(ctx, currentGamesList(downloadOutput.AllGames))
			gaba.Set(ctx, searchFilterString(downloadOutput.SearchFilter))
			return ui.GameDetailsOutput{}, gaba.ExitCodeBack
		}

		screen := ui.NewGameDetailsScreen()
		result, err := screen.Draw(ui.GameDetailsInput{
			Config:   config,
			Host:     host,
			Platform: gameListOutput.Platform,
			Game:     gameListOutput.SelectedGames[0],
		})

		if err != nil {
			return ui.GameDetailsOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, gameList, func(ctx *gaba.Context) error {
			detailsOutput, _ := gaba.Get[ui.GameDetailsOutput](ctx)
			config, _ := gaba.Get[*utils.Config](ctx)
			host, _ := gaba.Get[romm.Host](ctx)
			gameListOutput, _ := gaba.Get[ui.GameListOutput](ctx)
			filter, _ := gaba.Get[searchFilterString](ctx)

			if detailsOutput.DownloadRequested {
				downloadScreen := ui.NewDownloadScreen()
				downloadOutput := downloadScreen.Execute(*config, host, detailsOutput.Platform, []romm.Rom{detailsOutput.Game}, gameListOutput.AllGames, string(filter))
				gaba.Set(ctx, currentGamesList(downloadOutput.AllGames))
				gaba.Set(ctx, searchFilterString(downloadOutput.SearchFilter))
			}

			return nil
		}).
		On(gaba.ExitCodeBack, gameList)

	gaba.AddState(fsm, search, func(ctx *gaba.Context) (ui.SearchOutput, gaba.ExitCode) {
		filter, _ := gaba.Get[searchFilterString](ctx)

		screen := ui.NewSearchScreen()
		result, err := screen.Draw(ui.SearchInput{
			InitialText: string(filter),
		})

		if err != nil {
			return ui.SearchOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, gameList, func(ctx *gaba.Context) error {
			output, _ := gaba.Get[ui.SearchOutput](ctx)
			gaba.Set(ctx, searchFilterString(output.Query))
			fullGames, _ := gaba.Get[fullGamesList](ctx)
			gaba.Set(ctx, currentGamesList(fullGames))
			gaba.Set(ctx, gameListPosition{Index: 0, Pos: 0})
			return nil
		}).
		OnWithHook(gaba.ExitCodeBack, gameList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, searchFilterString(""))
			fullGames, _ := gaba.Get[fullGamesList](ctx)
			gaba.Set(ctx, currentGamesList(fullGames))
			return nil
		})

	gaba.AddState(fsm, collectionSearch, func(ctx *gaba.Context) (ui.SearchOutput, gaba.ExitCode) {
		filter, _ := gaba.Get[collectionSearchFilter](ctx)

		screen := ui.NewSearchScreen()
		result, err := screen.Draw(ui.SearchInput{
			InitialText: string(filter),
		})

		if err != nil {
			return ui.SearchOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, collectionList, func(ctx *gaba.Context) error {
			output, _ := gaba.Get[ui.SearchOutput](ctx)
			gaba.Set(ctx, collectionSearchFilter(output.Query))
			gaba.Set(ctx, collectionListPosition{Index: 0, Pos: 0})
			return nil
		}).
		OnWithHook(gaba.ExitCodeBack, collectionList, func(ctx *gaba.Context) error {
			gaba.Set(ctx, collectionSearchFilter(""))
			return nil
		})

	gaba.AddState(fsm, settings, func(ctx *gaba.Context) (ui.SettingsOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		cfw, _ := gaba.Get[constants.CFW](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		pos, _ := gaba.Get[settingsPosition](ctx)

		screen := ui.NewSettingsScreen()
		result, err := screen.Draw(ui.SettingsInput{
			Config:                config,
			CFW:                   cfw,
			Host:                  host,
			LastSelectedIndex:     pos.Index,
			LastVisibleStartIndex: pos.VisibleStartIndex,
		})

		if err != nil {
			return ui.SettingsOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, settingsPosition{
			Index:             result.Value.LastSelectedIndex,
			VisibleStartIndex: result.Value.LastVisibleStartIndex,
		})

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, platformSelection, func(ctx *gaba.Context) error {
			output, _ := gaba.Get[ui.SettingsOutput](ctx)
			host, _ := gaba.Get[romm.Host](ctx)
			utils.SaveConfig(output.Config)
			gaba.Set(ctx, output.Config)
			gaba.Set(ctx, settingsPosition{Index: 0})

			// Update showCollections based on new settings
			showCollections := utils.ShowCollections(output.Config, host)
			gaba.Set(ctx, showCollectionsBool(showCollections))
			return nil
		}).
		On(constants.ExitCodeCollectionsSettings, collectionsSettings).
		On(constants.ExitCodeAdvancedSettings, advancedSettings).
		OnWithHook(constants.ExitCodeInfo, info, func(ctx *gaba.Context) error {
			gaba.Set(ctx, infoPreviousState(settings))
			return nil
		}).
		On(constants.ExitCodeSyncArtwork, artworkSync).
		OnWithHook(gaba.ExitCodeBack, platformSelection, func(ctx *gaba.Context) error {
			gaba.Set(ctx, settingsPosition{Index: 0})
			return nil
		})

	gaba.AddState(fsm, collectionsSettings, func(ctx *gaba.Context) (ui.CollectionsSettingsOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		pos, _ := gaba.Get[collectionsSettingsPosition](ctx)

		screen := ui.NewCollectionsSettingsScreen()
		result, err := screen.Draw(ui.CollectionsSettingsInput{
			Config: config,
		})

		if err != nil {
			return ui.CollectionsSettingsOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, collectionsSettingsPosition{Index: pos.Index})

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, settings, func(ctx *gaba.Context) error {
			config, _ := gaba.Get[*utils.Config](ctx)
			host, _ := gaba.Get[romm.Host](ctx)

			gaba.Set(ctx, config)
			gaba.Set(ctx, collectionsSettingsPosition{Index: 0})

			// Update showCollections based on new settings
			showCollections := utils.ShowCollections(config, host)
			gaba.Set(ctx, showCollectionsBool(showCollections))
			return nil
		}).
		OnWithHook(gaba.ExitCodeBack, settings, func(ctx *gaba.Context) error {
			gaba.Set(ctx, collectionsSettingsPosition{Index: 0})
			return nil
		})

	gaba.AddState(fsm, advancedSettings, func(ctx *gaba.Context) (ui.AdvancedSettingsOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		pos, _ := gaba.Get[advancedSettingsPosition](ctx)

		screen := ui.NewAdvancedSettingsScreen()
		result, err := screen.Draw(ui.AdvancedSettingsInput{
			Config:                config,
			Host:                  host,
			LastSelectedIndex:     pos.Index,
			LastVisibleStartIndex: pos.VisibleStartIndex,
		})

		if err != nil {
			return ui.AdvancedSettingsOutput{}, gaba.ExitCodeError
		}

		gaba.Set(ctx, advancedSettingsPosition{
			Index:             result.Value.LastSelectedIndex,
			VisibleStartIndex: result.Value.LastVisibleStartIndex,
		})

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, settings, func(ctx *gaba.Context) error {
			config, _ := gaba.Get[*utils.Config](ctx)

			gaba.Set(ctx, config)
			gaba.Set(ctx, advancedSettingsPosition{Index: 0, VisibleStartIndex: 0})
			return nil
		}).
		OnWithHook(constants.ExitCodeInfo, info, func(ctx *gaba.Context) error {
			gaba.Set(ctx, infoPreviousState(advancedSettings))
			return nil
		}).
		On(constants.ExitCodeEditMappings, settingsPlatformMapping).
		On(constants.ExitCodeClearCache, clearCacheConfirmation).
		OnWithHook(gaba.ExitCodeBack, settings, func(ctx *gaba.Context) error {
			gaba.Set(ctx, advancedSettingsPosition{Index: 0, VisibleStartIndex: 0})
			return nil
		})

	gaba.AddState(fsm, settingsPlatformMapping, func(ctx *gaba.Context) (ui.PlatformMappingOutput, gaba.ExitCode) {
		host, _ := gaba.Get[romm.Host](ctx)
		config, _ := gaba.Get[*utils.Config](ctx)
		cfw, _ := gaba.Get[constants.CFW](ctx)

		screen := ui.NewPlatformMappingScreen()
		result, err := screen.Draw(ui.PlatformMappingInput{
			Host:           host,
			ApiTimeout:     config.ApiTimeout,
			CFW:            cfw,
			RomDirectory:   utils.GetRomDirectory(),
			AutoSelect:     false,
			HideBackButton: false,
		})

		if err != nil {
			return ui.PlatformMappingOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		OnWithHook(gaba.ExitCodeSuccess, advancedSettings, func(ctx *gaba.Context) error {
			output, _ := gaba.Get[ui.PlatformMappingOutput](ctx)
			config, _ := gaba.Get[*utils.Config](ctx)
			host, _ := gaba.Get[romm.Host](ctx)

			config.DirectoryMappings = output.Mappings
			config.PlatformOrder = utils.PrunePlatformOrder(config.PlatformOrder, output.Mappings)
			utils.SaveConfig(config)
			gaba.Set(ctx, config)

			platforms, err := utils.GetMappedPlatforms(host, output.Mappings)
			if err != nil {
				gaba.GetLogger().Error("Failed to load platforms", "error", err)
				return err
			}
			gaba.Set(ctx, platforms)
			return nil
		}).
		On(gaba.ExitCodeBack, advancedSettings)

	gaba.AddState(fsm, info, func(ctx *gaba.Context) (ui.InfoOutput, gaba.ExitCode) {
		host, _ := gaba.Get[romm.Host](ctx)
		prevState, _ := gaba.Get[infoPreviousState](ctx)

		screen := ui.NewInfoScreen()
		result, err := screen.Draw(ui.InfoInput{
			Host:         host,
			FromAdvanced: prevState == advancedSettings,
		})

		if err != nil {
			return ui.InfoOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeBack, settings).
		On(constants.ExitCodeBackToAdvanced, advancedSettings).
		On(constants.ExitCodeLogoutConfirm, logoutConfirmation)

	gaba.AddState(fsm, logoutConfirmation, func(ctx *gaba.Context) (ui.LogoutConfirmationOutput, gaba.ExitCode) {
		screen := ui.NewLogoutConfirmationScreen()
		result, err := screen.Draw()

		if err != nil {
			return ui.LogoutConfirmationOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeBack, info).
		OnWithHook(constants.ExitCodeLogout, platformSelection, func(ctx *gaba.Context) error {
			config, _ := gaba.Get[*utils.Config](ctx)
			cfw, _ := gaba.Get[constants.CFW](ctx)

			// Clear config and save
			config.Hosts = nil
			config.DirectoryMappings = nil
			config.PlatformOrder = nil

			if err := utils.SaveConfig(config); err != nil {
				gaba.GetLogger().Error("Failed to save config after logout", "error", err)
				return err
			}

			gaba.GetLogger().Info("User logged out successfully")

			// Trigger login flow inline with empty host (no pre-filled values)
			loginConfig, err := ui.LoginFlow(romm.Host{})
			if err != nil {
				gaba.GetLogger().Error("Login flow failed after logout", "error", err)
				// LoginFlow calls os.Exit(1) on cancel, so this only happens on errors
				return err
			}

			// Update config with new login
			config.Hosts = loginConfig.Hosts
			if err := utils.SaveConfig(config); err != nil {
				gaba.GetLogger().Error("Failed to save config after re-login", "error", err)
				return err
			}

			// Update context with new config and host
			gaba.Set(ctx, config)
			gaba.Set(ctx, config.Hosts[0])

			// Check if platform mappings exist, prompt if needed
			if len(config.DirectoryMappings) == 0 {
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

			// Load platforms
			platforms, err := utils.GetMappedPlatforms(config.Hosts[0], config.DirectoryMappings)
			if err != nil {
				gaba.GetLogger().Error("Failed to load platforms after re-login", "error", err)
				return err
			}
			gaba.Set(ctx, platforms)

			// Clear state to start fresh
			gaba.Set(ctx, searchFilterString(""))
			gaba.Set(ctx, currentGamesList(nil))
			gaba.Set(ctx, platformListPosition{Index: 0, Pos: 0})

			return nil
		})

	gaba.AddState(fsm, clearCacheConfirmation, func(ctx *gaba.Context) (ui.ClearCacheConfirmationOutput, gaba.ExitCode) {
		screen := ui.NewClearCacheConfirmationScreen()
		result, err := screen.Draw()

		if err != nil {
			return ui.ClearCacheConfirmationOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeBack, advancedSettings).
		On(gaba.ExitCodeSuccess, advancedSettings)

	gaba.AddState(fsm, saveSync, func(ctx *gaba.Context) (ui.SaveSyncOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)

		screen := ui.NewSaveSyncScreen()
		result, err := screen.Draw(ui.SaveSyncInput{
			Config: config,
			Host:   host,
		})

		if err != nil {
			return ui.SaveSyncOutput{}, gaba.ExitCodeError
		}

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeBack, platformSelection)

	gaba.AddState(fsm, syncIssues, func(ctx *gaba.Context) (ui.SyncIssuesOutput, gaba.ExitCode) {
		host, _ := gaba.Get[romm.Host](ctx)

		if autoSync == nil {
			return ui.SyncIssuesOutput{}, gaba.ExitCodeBack
		}

		screen := ui.NewSyncIssuesScreen()
		result, err := screen.Draw(ui.SyncIssuesInput{
			Issues: autoSync.GetIssues(),
			Host:   host,
			OnIssueResolved: func(issue utils.SyncIssue) {
				autoSync.RemoveIssue(issue)
			},
		})

		if err != nil {
			return ui.SyncIssuesOutput{}, gaba.ExitCodeError
		}

		// If an issue was resolved
		if result.Value.Resolved {
			if result.Value.HasMoreIssues {
				// More issues remain, loop back to show updated list
				return result.Value, gaba.ExitCodeSuccess
			}
			// All issues resolved, mark complete (syncs already done)
			autoSync.MarkComplete()
			return result.Value, gaba.ExitCodeBack
		}

		return result.Value, result.ExitCode
	}).
		On(gaba.ExitCodeBack, platformSelection).
		On(gaba.ExitCodeSuccess, syncIssues)

	gaba.AddState(fsm, biosDownload, func(ctx *gaba.Context) (ui.BIOSDownloadOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)
		platform, _ := gaba.Get[ui.PlatformSelectionOutput](ctx)
		collectionPlatform, _ := gaba.Get[ui.CollectionPlatformSelectionOutput](ctx)

		var selectedPlatform romm.Platform
		if collectionPlatform.SelectedPlatform.ID != 0 {
			selectedPlatform = collectionPlatform.SelectedPlatform
		} else {
			selectedPlatform = platform.SelectedPlatform
		}

		screen := ui.NewBIOSDownloadScreen()
		output := screen.Execute(*config, host, selectedPlatform)

		return output, gaba.ExitCodeBack
	}).
		On(gaba.ExitCodeBack, gameList)

	gaba.AddState(fsm, artworkSync, func(ctx *gaba.Context) (ui.ArtworkSyncOutput, gaba.ExitCode) {
		config, _ := gaba.Get[*utils.Config](ctx)
		host, _ := gaba.Get[romm.Host](ctx)

		screen := ui.NewArtworkSyncScreen()
		output := screen.Execute(*config, host)

		return output, gaba.ExitCodeBack
	}).
		On(gaba.ExitCodeBack, settings)

	return fsm.Start(platformSelection)
}
