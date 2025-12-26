package ui

import (
	"errors"
	"grout/constants"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	buttons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type LogoutConfirmationOutput struct {
	Confirmed bool
}

type LogoutConfirmationScreen struct{}

func NewLogoutConfirmationScreen() *LogoutConfirmationScreen {
	return &LogoutConfirmationScreen{}
}

func (s *LogoutConfirmationScreen) Draw() (ScreenResult[LogoutConfirmationOutput], error) {
	output := LogoutConfirmationOutput{}

	_, err := gaba.ConfirmationMessage(
		i18n.GetString("logout_confirm_message"),
		[]gaba.FooterHelpItem{
			{ButtonName: "B", HelpText: i18n.GetString("button_cancel")},
			{ButtonName: "Y", HelpText: i18n.GetString("button_confirm")},
		},
		gaba.MessageOptions{
			ConfirmButton: buttons.VirtualButtonY,
		},
	)

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil // B button - cancel
		}
		return withCode(output, gaba.ExitCodeError), err
	}

	// A button pressed - confirm logout
	output.Confirmed = true
	return withCode(output, constants.ExitCodeLogout), nil
}
