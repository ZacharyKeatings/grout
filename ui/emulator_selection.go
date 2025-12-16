package ui

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

type EmulatorSelectionInput struct {
	PlatformSlug         string
	PlatformName         string
	EmulatorChoices      []EmulatorChoice
	LastSelectedIndex    int
	LastSelectedPosition int
}

type EmulatorChoice struct {
	DirectoryName    string
	DisplayName      string
	HasExistingSaves bool
	SaveCount        int
}

type EmulatorSelectionOutput struct {
	SelectedEmulator     string
	LastSelectedIndex    int
	LastSelectedPosition int
}

type EmulatorSelectionScreen struct{}

func NewEmulatorSelectionScreen() *EmulatorSelectionScreen {
	return &EmulatorSelectionScreen{}
}

func (s *EmulatorSelectionScreen) Draw(input EmulatorSelectionInput) (ScreenResult[EmulatorSelectionOutput], error) {
	output := EmulatorSelectionOutput{
		LastSelectedIndex:    input.LastSelectedIndex,
		LastSelectedPosition: input.LastSelectedPosition,
	}

	sortedChoices := make([]EmulatorChoice, len(input.EmulatorChoices))

	if len(input.EmulatorChoices) > 0 {
		sortedChoices[0] = input.EmulatorChoices[0]

		rest := make([]EmulatorChoice, len(input.EmulatorChoices)-1)
		copy(rest, input.EmulatorChoices[1:])

		sort.Slice(rest, func(i, j int) bool {
			return strings.ToLower(rest[i].DirectoryName) < strings.ToLower(rest[j].DirectoryName)
		})

		copy(sortedChoices[1:], rest)
	} else {
		copy(sortedChoices, input.EmulatorChoices)
	}

	var menuItems []gaba.MenuItem
	for _, choice := range sortedChoices {
		displayText := choice.DisplayName
		if choice.HasExistingSaves {
			displayText = fmt.Sprintf("%s (%d saves)", choice.DisplayName, choice.SaveCount)
		}

		menuItems = append(menuItems, gaba.MenuItem{
			Text:     displayText,
			Selected: false,
			Focused:  false,
			Metadata: choice.DirectoryName,
		})
	}

	footerItems := []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: "Cancel"},
		{ButtonName: "A", HelpText: "Select"},
	}

	title := fmt.Sprintf("Select %s Emulator", input.PlatformName)
	options := gaba.DefaultListOptions(title, menuItems)
	options.SmallTitle = true
	options.FooterHelpItems = footerItems
	options.SelectedIndex = input.LastSelectedIndex
	options.VisibleStartIndex = max(0, input.LastSelectedIndex-input.LastSelectedPosition)

	sel, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		return withCode(output, gaba.ExitCodeError), err
	}

	switch sel.Action {
	case gaba.ListActionSelected:
		selectedEmulator := sel.Items[sel.Selected[0]].Metadata.(string)

		output.SelectedEmulator = selectedEmulator
		output.LastSelectedIndex = sel.Selected[0]
		output.LastSelectedPosition = sel.VisiblePosition
		return success(output), nil

	default:
		return back(output), nil
	}
}
