package ui

import (
	"grout/romm"
	"grout/utils"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
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

	type syncScanResult struct {
		Results   []utils.SyncResult
		Unmatched []utils.UnmatchedSave
	}

	// Step 1: Scan for syncs (no UI nesting here)
	type scanResult struct {
		Syncs     []interface{} // Store as interface{} to avoid exposing internal type
		Unmatched []utils.UnmatchedSave
	}

	scanData, _ := gaba.ProcessMessage("Scanning save files...", gaba.ProcessMessageOptions{}, func() (interface{}, error) {
		syncs, unmatched, err := utils.FindSaveSyncs(input.Host)
		if err != nil {
			gaba.GetLogger().Error("Unable to scan save files!", "error", err)
			return nil, nil
		}

		// Convert to interface{} slice
		syncInterfaces := make([]interface{}, len(syncs))
		for i, s := range syncs {
			syncInterfaces[i] = s
		}

		return scanResult{Syncs: syncInterfaces, Unmatched: unmatched}, nil
	})

	var results []utils.SyncResult
	var unmatched []utils.UnmatchedSave

	if scan, ok := scanData.(scanResult); ok {
		unmatched = scan.Unmatched
		results = make([]utils.SyncResult, 0, len(scan.Syncs))

		// Get platform names for better UX
		rc := utils.GetRommClient(input.Host)
		platforms, err := rc.GetPlatforms()
		if err != nil {
			gaba.GetLogger().Warn("Failed to fetch platforms, using slugs", "error", err)
		}

		// Build slug -> platform name map
		platformNames := make(map[string]string)
		for _, p := range platforms {
			platformNames[p.Slug] = p.Name
		}

		// Convert to slice of SaveSync pointers
		syncs := make([]*utils.SaveSync, len(scan.Syncs))
		for i, syncInterface := range scan.Syncs {
			sync := syncInterface.(utils.SaveSync)
			syncs[i] = &sync
		}

		// Group syncs by platform slug
		syncsByPlatform := make(map[string][]*utils.SaveSync)
		for _, s := range syncs {
			slug := s.GetSlug()
			syncsByPlatform[slug] = append(syncsByPlatform[slug], s)
		}

		// Step 2: Handle emulator selection once per platform
		emulatorSelections := make(map[string]string) // slug -> selected emulator

		for slug, platformSyncs := range syncsByPlatform {
			// Check if any sync in this platform needs selection
			needsSelection := false
			for _, s := range platformSyncs {
				if s.NeedsEmulatorSelection() {
					needsSelection = true
					break
				}
			}

			dirInfos := utils.GetEmulatorDirectoriesWithStatus(slug)

			if needsSelection {

				// Convert to UI choices with default marked
				uiChoices := make([]EmulatorChoice, len(dirInfos))
				for i, info := range dirInfos {
					displayName := info.DirectoryName
					if i == 0 {
						// First one is the default
						displayName = info.DirectoryName + " (Default)"
					}
					uiChoices[i] = EmulatorChoice{
						DirectoryName:    info.DirectoryName,
						DisplayName:      displayName,
						HasExistingSaves: info.HasSaves,
						SaveCount:        info.SaveCount,
					}
				}

				// Get platform name for display
				platformName := platformNames[slug]
				if platformName == "" {
					platformName = slug
				}

				// Show emulator selection UI (outside ProcessMessage - no nesting!)
				screen := NewEmulatorSelectionScreen()
				selResult, err := screen.Draw(EmulatorSelectionInput{
					PlatformSlug:    slug,
					PlatformName:    platformName,
					EmulatorChoices: uiChoices,
				})

				if err != nil || selResult.ExitCode != gaba.ExitCodeSuccess {
					// User cancelled - skip all syncs for this platform (don't add to results)
					gaba.GetLogger().Debug("User cancelled emulator selection", "platform", slug)
					continue
				}

				emulatorSelections[slug] = selResult.Value.SelectedEmulator
				gaba.GetLogger().Debug("Stored emulator selection for platform", "slug", slug, "selectedEmulator", selResult.Value.SelectedEmulator)
			} else {
				// Auto-select if there's exactly one non-empty directory
				nonEmptyDirs := make([]utils.EmulatorDirectoryInfo, 0)
				for _, info := range dirInfos {
					if info.HasSaves {
						nonEmptyDirs = append(nonEmptyDirs, info)
					}
				}

				if len(nonEmptyDirs) == 1 {
					// Auto-select the single non-empty directory
					emulatorSelections[slug] = nonEmptyDirs[0].DirectoryName
					gaba.GetLogger().Debug("Auto-selected single non-empty directory", "slug", slug, "selectedEmulator", nonEmptyDirs[0].DirectoryName)
				}
			}
		}

		// Step 3: Execute all syncs
		for _, s := range syncs {
			// Skip syncs that needed selection but user cancelled
			if s.NeedsEmulatorSelection() {
				if _, ok := emulatorSelections[s.GetSlug()]; !ok {
					// User cancelled for this platform - skip this sync
					gaba.GetLogger().Debug("Skipping sync due to cancelled emulator selection", "game", s.GetGameBase())
					continue
				}
			}

			// Apply emulator selection if one was made for this platform
			if selectedEmulator, ok := emulatorSelections[s.GetSlug()]; ok {
				gaba.GetLogger().Debug("Applying emulator selection to sync", "game", s.GetGameBase(), "slug", s.GetSlug(), "selectedEmulator", selectedEmulator)
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
				gaba.GetLogger().Error("Unable to sync save!", "game", s.GetGameBase(), "error", result.Error)
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
		gaba.ProcessMessage("Everything is up to date!\nGo play some games!", gaba.ProcessMessageOptions{}, func() (interface{}, error) {
			time.Sleep(time.Second * 2)
			return nil, nil
		})
	}

	return back(output), nil
}
