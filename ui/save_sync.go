package ui

import (
	"grout/romm"
	"grout/utils"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type SaveSyncInput struct {
	Config *utils.Config
	Host   romm.Host
}

type SaveSyncOutput struct{}

type SaveSyncScreen struct{}

func NewSaveSyncScreen() *SaveSyncScreen {
	return &SaveSyncScreen{}
}

func (s *SaveSyncScreen) Draw(input SaveSyncInput) (ScreenResult[SaveSyncOutput], error) {
	output := SaveSyncOutput{}

	type scanResult struct {
		Syncs     []utils.SaveSync
		Unmatched []utils.UnmatchedSave
	}

	scanData, _ := gaba.ProcessMessage(i18n.GetString("save_sync_scanning"), gaba.ProcessMessageOptions{}, func() (interface{}, error) {
		syncs, unmatched, err := utils.FindSaveSyncs(input.Host)
		if err != nil {
			gaba.GetLogger().Error("Unable to scan save files!", "error", err)
			return nil, nil
		}

		return scanResult{Syncs: syncs, Unmatched: unmatched}, nil
	})

	var results []utils.SyncResult
	var unmatched []utils.UnmatchedSave

	if scan, ok := scanData.(scanResult); ok {
		unmatched = scan.Unmatched
		results = make([]utils.SyncResult, 0, len(scan.Syncs))

		rc := utils.GetRommClient(input.Host)
		platforms, err := rc.GetPlatforms()
		if err != nil {
			gaba.GetLogger().Warn("Failed to fetch platforms, using slugs", "error", err)
		}

		platformNames := make(map[string]string)
		for _, p := range platforms {
			platformNames[p.Slug] = p.Name
		}

		syncs := make([]*utils.SaveSync, len(scan.Syncs))
		for i := range scan.Syncs {
			syncs[i] = &scan.Syncs[i]
		}

		syncsByPlatform := make(map[string][]*utils.SaveSync)
		for _, s := range syncs {
			syncsByPlatform[s.Slug] = append(syncsByPlatform[s.Slug], s)
		}

		emulatorSelections := make(map[string]string)

		for slug, platformSyncs := range syncsByPlatform {
			needsSelection := false
			for _, s := range platformSyncs {
				if s.NeedsEmulatorSelection() {
					needsSelection = true
					break
				}
			}

			dirInfos := utils.GetEmulatorDirectoriesWithStatus(slug)

			if needsSelection {

				uiChoices := make([]EmulatorChoice, len(dirInfos))
				for i, info := range dirInfos {
					displayName := info.DirectoryName
					if i == 0 {
						displayName = info.DirectoryName + i18n.GetString("emulator_default")
					}
					uiChoices[i] = EmulatorChoice{
						DirectoryName:    info.DirectoryName,
						DisplayName:      displayName,
						HasExistingSaves: info.HasSaves,
						SaveCount:        info.SaveCount,
					}
				}

				platformName := platformNames[slug]
				if platformName == "" {
					platformName = slug
				}

				screen := NewEmulatorSelectionScreen()
				selResult, err := screen.Draw(EmulatorSelectionInput{
					PlatformSlug:    slug,
					PlatformName:    platformName,
					EmulatorChoices: uiChoices,
				})

				if err != nil || selResult.ExitCode != gaba.ExitCodeSuccess {
					gaba.GetLogger().Debug("User cancelled emulator selection", "platform", slug)
					continue
				}

				emulatorSelections[slug] = selResult.Value.SelectedEmulator
				gaba.GetLogger().Debug("Stored emulator selection for platform", "slug", slug, "selectedEmulator", selResult.Value.SelectedEmulator)
			} else {
				nonEmptyDirs := make([]utils.EmulatorDirectoryInfo, 0)
				for _, info := range dirInfos {
					if info.HasSaves {
						nonEmptyDirs = append(nonEmptyDirs, info)
					}
				}

				if len(nonEmptyDirs) == 1 {
					emulatorSelections[slug] = nonEmptyDirs[0].DirectoryName
					gaba.GetLogger().Debug("Auto-selected single non-empty directory", "slug", slug, "selectedEmulator", nonEmptyDirs[0].DirectoryName)
				}
			}
		}

		for _, s := range syncs {
			if s.NeedsEmulatorSelection() {
				if _, ok := emulatorSelections[s.Slug]; !ok {
					gaba.GetLogger().Debug("Skipping sync due to cancelled emulator selection", "game", s.GameBase)
					continue
				}
			}

			if selectedEmulator, ok := emulatorSelections[s.Slug]; ok {
				gaba.GetLogger().Debug("Applying emulator selection to sync", "game", s.GameBase, "slug", s.Slug, "selectedEmulator", selectedEmulator)
				s.SetSelectedEmulator(selectedEmulator)
			}

			gaba.GetLogger().Debug("Syncing save file", "save_info", s)
			result := s.Execute(input.Host)
			results = append(results, result)
			gaba.GetLogger().Debug("Added result to list",
				"resultsCount", len(results),
				"action", result.Action,
				"success", result.Success,
				"gameName", result.GameName,
				"romDisplayName", result.RomDisplayName)
			if !result.Success {
				gaba.GetLogger().Error("Unable to sync save!", "game", s.GameBase, "error", result.Error)
			} else {
				gaba.GetLogger().Debug("Save synced!", "save_info", s)
			}
		}
	}

	if len(results) > 0 || len(unmatched) > 0 {
		reportScreen := newSyncReportScreen()
		_, err := reportScreen.draw(syncReportInput{
			Results:   results,
			Unmatched: unmatched,
		})
		if err != nil {
			gaba.GetLogger().Error("Error showing sync report", "error", err)
		}
	} else {
		gaba.ProcessMessage(i18n.GetString("save_sync_up_to_date"), gaba.ProcessMessageOptions{}, func() (interface{}, error) {
			time.Sleep(time.Second * 2)
			return nil, nil
		})
	}

	return back(output), nil
}
