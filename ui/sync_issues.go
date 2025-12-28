package ui

import (
	"errors"
	"grout/romm"
	"grout/utils"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type SyncIssuesInput struct {
	Issues          []utils.SyncIssue
	Host            romm.Host
	OnIssueResolved func(issue utils.SyncIssue) // Called when an issue is resolved
}

type SyncIssuesOutput struct {
	Resolved      bool            // true if any issues were resolved
	ResolvedIssue utils.SyncIssue // the issue that was resolved
	HasMoreIssues bool            // true if there are remaining issues
}

type SyncIssuesScreen struct{}

func NewSyncIssuesScreen() *SyncIssuesScreen {
	return &SyncIssuesScreen{}
}

func (s *SyncIssuesScreen) Draw(input SyncIssuesInput) (ScreenResult[SyncIssuesOutput], error) {
	output := SyncIssuesOutput{}

	if len(input.Issues) == 0 {
		return back(output), nil
	}

	// Get platform names for display
	rc := utils.GetRommClient(input.Host)
	platforms, err := rc.GetPlatforms()
	if err != nil {
		gaba.GetLogger().Warn("Failed to fetch platforms, using slugs", "error", err)
	}

	platformNames := make(map[string]string)
	for _, p := range platforms {
		platformNames[p.Slug] = p.Name
	}

	// Build menu items from issues
	var menuItems []gaba.MenuItem
	for _, issue := range input.Issues {
		displayName := issue.Sync.GameBase
		if issue.Sync.RomName != "" {
			displayName = issue.Sync.RomName
		}

		// Add status suffix to the display name
		if issue.NeedsEmulator {
			displayName = displayName + " - " + i18n.Localize(&goi18n.Message{ID: "sync_issue_needs_emulator", Other: "Select emulator"}, nil)
		} else if issue.ErrorMessage != "" {
			displayName = displayName + " - " + i18n.Localize(&goi18n.Message{ID: "sync_issue_failed", Other: "Failed"}, nil)
		}

		menuItems = append(menuItems, gaba.MenuItem{
			Text:     displayName,
			Selected: false,
			Focused:  false,
			Metadata: issue,
		})
	}

	footerItems := []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: i18n.Localize(&goi18n.Message{ID: "button_back", Other: "Back"}, nil)},
		{ButtonName: "A", HelpText: i18n.Localize(&goi18n.Message{ID: "button_select", Other: "Select"}, nil)},
	}

	title := i18n.Localize(&goi18n.Message{ID: "sync_issues_title", Other: "Sync Issues"}, nil)
	options := gaba.DefaultListOptions(title, menuItems)
	options.SmallTitle = true
	options.FooterHelpItems = footerItems
	options.StatusBar = utils.StatusBar()

	sel, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		return withCode(output, gaba.ExitCodeError), err
	}

	switch sel.Action {
	case gaba.ListActionSelected:
		selectedIssue := sel.Items[sel.Selected[0]].Metadata.(utils.SyncIssue)

		var resolved bool
		if selectedIssue.NeedsEmulator {
			// Show emulator selection
			resolved = s.handleEmulatorSelection(selectedIssue, input.Host, platformNames)
		} else if selectedIssue.ErrorMessage != "" {
			// Show error details and offer retry
			resolved = s.handleFailedSync(selectedIssue, input.Host)
		}

		if resolved {
			output.Resolved = true
			output.ResolvedIssue = selectedIssue
			// Call the callback to remove this issue from the list
			if input.OnIssueResolved != nil {
				input.OnIssueResolved(selectedIssue)
			}
			// Check if there are more issues (subtract 1 for the one we just resolved)
			output.HasMoreIssues = len(input.Issues) > 1
		}

		return success(output), nil

	default:
		return back(output), nil
	}
}

func (s *SyncIssuesScreen) handleEmulatorSelection(issue utils.SyncIssue, host romm.Host, platformNames map[string]string) bool {
	logger := gaba.GetLogger()

	dirInfos := utils.GetEmulatorDirectoriesWithStatus(issue.Sync.Slug)
	if len(dirInfos) == 0 {
		logger.Error("No emulator directories found", "slug", issue.Sync.Slug)
		return false
	}

	uiChoices := make([]EmulatorChoice, len(dirInfos))
	for i, info := range dirInfos {
		displayName := info.DirectoryName
		if i == 0 {
			displayName = info.DirectoryName + i18n.Localize(&goi18n.Message{ID: "emulator_default", Other: " (Default)"}, nil)
		}
		uiChoices[i] = EmulatorChoice{
			DirectoryName:    info.DirectoryName,
			DisplayName:      displayName,
			HasExistingSaves: info.HasSaves,
			SaveCount:        info.SaveCount,
		}
	}

	platformName := platformNames[issue.Sync.Slug]
	if platformName == "" {
		platformName = issue.Sync.Slug
	}

	screen := NewEmulatorSelectionScreen()
	selResult, err := screen.Draw(EmulatorSelectionInput{
		PlatformSlug:    issue.Sync.Slug,
		PlatformName:    platformName,
		EmulatorChoices: uiChoices,
	})

	if err != nil || selResult.ExitCode != gaba.ExitCodeSuccess {
		logger.Debug("User cancelled emulator selection")
		return false
	}

	// Execute the sync with selected emulator
	syncCopy := issue.Sync
	syncCopy.SetSelectedEmulator(selResult.Value.SelectedEmulator)

	result := syncCopy.Execute(host)
	if result.Success {
		logger.Debug("Sync completed successfully", "game", issue.Sync.GameBase)
		return true
	}

	logger.Error("Sync failed", "game", issue.Sync.GameBase, "error", result.Error)
	return false
}

func (s *SyncIssuesScreen) handleFailedSync(issue utils.SyncIssue, host romm.Host) bool {
	logger := gaba.GetLogger()

	// Show error and ask if user wants to retry
	errorMsg := i18n.Localize(&goi18n.Message{
		ID:    "sync_issue_error_details",
		Other: "{{.Game}} failed to sync:\n{{.Error}}\n\nRetry?",
	}, map[string]interface{}{
		"Game":  issue.Sync.GameBase,
		"Error": issue.ErrorMessage,
	})

	footerItems := []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: i18n.Localize(&goi18n.Message{ID: "button_cancel", Other: "Cancel"}, nil)},
		{ButtonName: "A", HelpText: i18n.Localize(&goi18n.Message{ID: "sync_issue_retry", Other: "Retry"}, nil)},
	}

	result, err := gaba.ConfirmationMessage(errorMsg, footerItems, gaba.MessageOptions{})
	if err != nil || result == nil || !result.Confirmed {
		return false
	}

	// Retry the sync
	syncResult := issue.Sync.Execute(host)
	if syncResult.Success {
		logger.Debug("Retry successful", "game", issue.Sync.GameBase)
		return true
	}

	logger.Error("Retry failed", "game", issue.Sync.GameBase, "error", syncResult.Error)
	return false
}
