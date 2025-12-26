package ui

import (
	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

type LanguageSelectionScreen struct{}

func NewLanguageSelectionScreen() *LanguageSelectionScreen {
	return &LanguageSelectionScreen{}
}

func (s *LanguageSelectionScreen) Draw() (string, error) {
	options := []gaba.SelectionOption{
		{DisplayName: "English", Description: "Welcome to Grout!", Value: "en"},
		{DisplayName: "Deutsch", Description: "Willkommen bei Grout!", Value: "de"},
		{DisplayName: "Español", Description: "¡Bienvenido a Grout!", Value: "es"},
		{DisplayName: "Français", Description: "Bienvenue sur Grout!", Value: "fr"},
		{DisplayName: "Italiano", Description: "Benvenuto su Grout!", Value: "it"},
		{DisplayName: "Português", Description: "Bem-vindo ao Grout!", Value: "pt"},
		{DisplayName: "Русский", Description: "Добро пожаловать в Grout!", Value: "ru"},
		{DisplayName: "日本語", Description: "Groutへようこそ！", Value: "ja"},
	}

	result, err := gaba.SelectionMessage(
		"Welcome to Grout!",
		options,
		[]gaba.FooterHelpItem{
			{ButtonName: "←→", HelpText: "Select"},
			{ButtonName: "A", HelpText: "Confirm"},
		},
		gaba.SelectionMessageSettings{
			DisableBackButton: true,
		},
	)

	if err != nil {
		return "", err
	}

	return result.SelectedValue.(string), nil
}
