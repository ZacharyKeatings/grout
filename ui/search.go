package ui

import (
	"errors"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

// SearchInput contains data needed to render the search screen
type SearchInput struct {
	InitialText string // Pre-populate the keyboard with this text
}

// SearchOutput contains the result of the search screen
type SearchOutput struct {
	Query string
}

// SearchScreen displays a keyboard for entering search queries
type SearchScreen struct{}

func NewSearchScreen() *SearchScreen {
	return &SearchScreen{}
}

func (s *SearchScreen) Draw(input SearchInput) (gaba.ScreenResult[SearchOutput], error) {
	res, err := gaba.Keyboard(input.InitialText)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			// User cancelled - not an error, just go back
			return gaba.Back[SearchOutput](), nil
		}
		gaba.GetLogger().Error("Error with keyboard", "error", err)
		return gaba.WithCode(SearchOutput{}, gaba.ExitCodeError), err
	}

	return gaba.Success(SearchOutput{Query: res.Text}), nil
}
