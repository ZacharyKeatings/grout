package ui

import (
	"errors"
	"fmt"
	"grout/utils"
	"path/filepath"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type syncReportInput struct {
	Results   []utils.SyncResult
	Unmatched []utils.UnmatchedSave
}

type syncReportOutput struct{}

type SyncReportScreen struct{}

func newSyncReportScreen() *SyncReportScreen {
	return &SyncReportScreen{}
}

func (s *SyncReportScreen) draw(input syncReportInput) (ScreenResult[syncReportOutput], error) {
	logger := gaba.GetLogger()
	output := syncReportOutput{}

	sections := s.buildSections(input.Results, input.Unmatched)

	options := gaba.DefaultInfoScreenOptions()
	options.Sections = sections
	options.ShowThemeBackground = false
	options.ShowScrollbar = true

	result, err := gaba.DetailScreen(i18n.GetString("save_sync_summary"), options, []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: i18n.GetString("button_close")},
	})

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		logger.Error("Detail screen error", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	if result.Action == gaba.DetailActionCancelled {
		return back(output), nil
	}

	return success(output), nil
}

func (s *SyncReportScreen) buildSections(results []utils.SyncResult, unmatched []utils.UnmatchedSave) []gaba.Section {
	logger := gaba.GetLogger()
	logger.Debug("Building sync report", "totalResults", len(results), "unmatched", len(unmatched))

	sections := make([]gaba.Section, 0)

	uploadedCount := 0
	downloadedCount := 0
	skippedCount := 0
	failedCount := 0

	for _, r := range results {
		if !r.Success {
			failedCount++
			continue
		}
		switch r.Action {
		case utils.Upload:
			uploadedCount++
		case utils.Download:
			downloadedCount++
		case utils.Skip:
			skippedCount++
		}
	}

	summary := []gaba.MetadataItem{
		{Label: i18n.GetString("save_sync_total_processed"), Value: fmt.Sprintf("%d", len(results))},
	}

	if downloadedCount > 0 {
		summary = append(summary, gaba.MetadataItem{Label: i18n.GetString("save_sync_downloaded"), Value: fmt.Sprintf("%d", downloadedCount)})
	}

	if uploadedCount > 0 {
		summary = append(summary, gaba.MetadataItem{
			Label: i18n.GetString("save_sync_uploaded"), Value: fmt.Sprintf("%d", uploadedCount)})
	}

	if skippedCount > 0 {
		summary = append(summary, gaba.MetadataItem{
			Label: i18n.GetString("save_sync_skipped"), Value: fmt.Sprintf("%d", skippedCount)})
	}

	if failedCount > 0 {
		summary = append(summary, gaba.MetadataItem{
			Label: i18n.GetString("save_sync_failed"), Value: fmt.Sprintf("%d", failedCount)})
	}

	sections = append(sections, gaba.NewInfoSection(i18n.GetString("save_sync_summary_section"), summary))

	if downloadedCount > 0 {
		downloadedFiles := ""
		for _, r := range results {
			if r.Success && r.Action == utils.Download {
				if downloadedFiles != "" {
					downloadedFiles += "\n"
				}
				displayName := r.RomDisplayName
				if displayName == "" {
					displayName = filepath.Base(r.FilePath)
				}
				downloadedFiles += displayName
			}
		}
		sections = append(sections, gaba.NewDescriptionSection(i18n.GetString("save_sync_downloaded"), downloadedFiles))
	}

	if uploadedCount > 0 {
		uploadedFiles := ""
		for _, r := range results {
			if r.Success && r.Action == utils.Upload {
				if uploadedFiles != "" {
					uploadedFiles += "\n"
				}
				displayName := r.RomDisplayName
				if displayName == "" {
					displayName = filepath.Base(r.FilePath)
				}
				logger.Debug("Upload result for report",
					"gameName", r.GameName,
					"romDisplayName", r.RomDisplayName,
					"filePath", r.FilePath,
					"displayName", displayName)
				uploadedFiles += displayName
			}
		}
		sections = append(sections, gaba.NewDescriptionSection(i18n.GetString("save_sync_uploaded"), uploadedFiles))
	}

	if failedCount > 0 {
		failedFiles := ""
		for _, r := range results {
			if !r.Success {
				if failedFiles != "" {
					failedFiles += "\n"
				}
				errorMsg := r.Error
				if errorMsg == "" {
					errorMsg = i18n.GetString("save_sync_unknown_error")
				}
				displayName := r.RomDisplayName
				if displayName == "" {
					displayName = r.GameName
				}
				failedFiles += fmt.Sprintf("%s (%s): %s", displayName, r.Action, errorMsg)
			}
		}
		sections = append(sections, gaba.NewDescriptionSection(i18n.GetString("save_sync_failed"), failedFiles))
	}

	// Display unmatched saves (ROM not found in RomM)
	if len(unmatched) > 0 {
		unmatchedText := ""
		for _, u := range unmatched {
			if unmatchedText != "" {
				unmatchedText += "\n"
			}
			unmatchedText += i18n.GetStringWithData("save_sync_rom_not_found", map[string]interface{}{"Name": filepath.Base(u.SavePath)})
		}
		sections = append(sections, gaba.NewDescriptionSection(i18n.GetString("save_sync_unmatched_saves"), unmatchedText))
	}

	return sections
}
