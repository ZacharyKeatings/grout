package ui

import (
	"errors"
	"grout/utils"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	buttons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type ClearCacheConfirmationOutput struct {
	Confirmed bool
}

type ClearCacheConfirmationScreen struct{}

func NewClearCacheConfirmationScreen() *ClearCacheConfirmationScreen {
	return &ClearCacheConfirmationScreen{}
}

func (s *ClearCacheConfirmationScreen) Draw() (ScreenResult[ClearCacheConfirmationOutput], error) {
	output := ClearCacheConfirmationOutput{}

	_, err := gaba.ConfirmationMessage(
		i18n.GetString("clear_cache_confirm_message"),
		[]gaba.FooterHelpItem{
			{ButtonName: "B", HelpText: i18n.GetString("button_cancel")},
			{ButtonName: "X", HelpText: i18n.GetString("button_confirm")},
		},
		gaba.MessageOptions{
			ConfirmButton: buttons.VirtualButtonX,
		},
	)

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil // B button - cancel
		}
		return withCode(output, gaba.ExitCodeError), err
	}

	// Y button pressed - clear cache
	err = utils.ClearArtworkCache()
	if err != nil {
		gaba.GetLogger().Error("Failed to clear cache", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	output.Confirmed = true
	return success(output), nil
}
