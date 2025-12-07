package models

import "github.com/brandonkowalski/go-romm"

type AppState struct {
	Config      *Config
	HostIndices map[string]int

	CurrentFullGamesList []romm.SimpleRom
	LastSelectedIndex    int
	LastSelectedPosition int
}
