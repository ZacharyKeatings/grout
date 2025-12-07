package ui

import (
	"errors"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
	"github.com/brandonkowalski/go-romm"
)

// PlatformSelectionInput contains data needed to render the platform selection screen
type PlatformSelectionInput struct {
	Platforms  []romm.Platform
	QuitOnBack bool // If true, back button quits the app; if false, it navigates back
}

// PlatformSelectionOutput contains the result of the platform selection screen
type PlatformSelectionOutput struct {
	SelectedPlatform romm.Platform
}

// PlatformSelectionScreen displays a list of platforms to choose from
type PlatformSelectionScreen struct{}

func NewPlatformSelectionScreen() *PlatformSelectionScreen {
	return &PlatformSelectionScreen{}
}

func (s *PlatformSelectionScreen) Draw(input PlatformSelectionInput) (gaba.ScreenResult[PlatformSelectionOutput], error) {
	// Handle empty platforms
	if len(input.Platforms) == 0 {
		return gaba.WithCode(PlatformSelectionOutput{}, gaba.ExitCode(404)), nil
	}

	// Build menu items
	menuItems := make([]gaba.MenuItem, len(input.Platforms))
	for i, platform := range input.Platforms {
		menuItems[i] = gaba.MenuItem{
			Text:     platform.Name,
			Selected: false,
			Focused:  false,
			Metadata: platform,
		}
	}

	// Configure footer based on navigation mode
	var footerItems []gaba.FooterHelpItem
	if input.QuitOnBack {
		footerItems = []gaba.FooterHelpItem{
			{ButtonName: "B", HelpText: "Quit"},
			{ButtonName: "X", HelpText: "Settings"},
			{ButtonName: "A", HelpText: "Select"},
		}
	} else {
		footerItems = []gaba.FooterHelpItem{
			{ButtonName: "B", HelpText: "Back"},
			{ButtonName: "A", HelpText: "Select"},
		}
	}

	// Configure list options
	options := gaba.DefaultListOptions("Grout", menuItems)
	options.EnableAction = input.QuitOnBack
	options.FooterHelpItems = footerItems

	sel, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return gaba.Back(PlatformSelectionOutput{}), nil
		}
		return gaba.WithCode(PlatformSelectionOutput{}, gaba.ExitCodeError), err
	}

	switch sel.Action {
	case gaba.ListActionSelected:
		platform := sel.Items[sel.Selected[0]].Metadata.(romm.Platform)
		return gaba.Success(PlatformSelectionOutput{SelectedPlatform: platform}), nil

	case gaba.ListActionTriggered:
		// Settings action (X button) - only available when QuitOnBack is true
		if input.QuitOnBack {
			return gaba.WithCode(PlatformSelectionOutput{}, gaba.ExitCodeSettings), nil
		}
	}

	// Back/cancel
	return gaba.WithCode(PlatformSelectionOutput{}, gaba.ExitCodeBack), nil
}
