package ui

import (
	"errors"
	"fmt"
	"grout/constants"
	"grout/romm"
	"grout/utils"
	"slices"
	"strings"
	"sync"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	buttons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type CollectionSelectionInput struct {
	Config                   *utils.Config
	Host                     romm.Host
	SearchFilter             string
	LastSelectedIndex        int
	LastSelectedPosition     int
	CachedRegularCollections []romm.Collection
	CachedSmartCollections   []romm.Collection
	CachedVirtualCollections []romm.VirtualCollection
}

type CollectionSelectionOutput struct {
	SelectedCollection        romm.Collection
	SearchFilter              string
	LastSelectedIndex         int
	LastSelectedPosition      int
	FetchedRegularCollections []romm.Collection
	FetchedSmartCollections   []romm.Collection
	FetchedVirtualCollections []romm.VirtualCollection
}

type CollectionSelectionScreen struct{}

func NewCollectionSelectionScreen() *CollectionSelectionScreen {
	return &CollectionSelectionScreen{}
}

func (s *CollectionSelectionScreen) Draw(input CollectionSelectionInput) (ScreenResult[CollectionSelectionOutput], error) {
	output := CollectionSelectionOutput{
		SearchFilter:         input.SearchFilter,
		LastSelectedIndex:    input.LastSelectedIndex,
		LastSelectedPosition: input.LastSelectedPosition,
	}

	rc := utils.GetRommClient(input.Host)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var regularCollections []romm.Collection
	var smartCollections []romm.Collection
	var virtualCollections []romm.VirtualCollection

	// Fetch regular collections if enabled
	if input.Config.ShowCollections {
		// Use cached regular collections or fetch
		if len(input.CachedRegularCollections) > 0 {
			regularCollections = input.CachedRegularCollections
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				fetched, err := rc.GetCollections()
				if err == nil {
					mu.Lock()
					regularCollections = fetched
					mu.Unlock()
				}
			}()
		}
	}

	// Fetch smart collections if enabled
	if input.Config.ShowSmartCollections {
		// Use cached smart collections or fetch
		if len(input.CachedSmartCollections) > 0 {
			smartCollections = input.CachedSmartCollections
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				fetched, err := rc.GetSmartCollections()
				if err == nil {
					mu.Lock()
					smartCollections = fetched
					for i := range smartCollections {
						smartCollections[i].IsSmart = true
					}
					mu.Unlock()
				}
			}()
		}
	}

	// Fetch virtual collections if enabled
	if input.Config.ShowVirtualCollections {
		// Use cached virtual collections or fetch
		if len(input.CachedVirtualCollections) > 0 {
			virtualCollections = input.CachedVirtualCollections
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				fetched, err := rc.GetVirtualCollections()
				if err == nil {
					mu.Lock()
					virtualCollections = fetched
					mu.Unlock()
				}
			}()
		}
	}

	// Wait for all fetches to complete
	wg.Wait()

	// Store fetched collections for caching
	output.FetchedRegularCollections = regularCollections
	output.FetchedSmartCollections = smartCollections
	output.FetchedVirtualCollections = virtualCollections

	// Combine enabled collections based on current config
	var collections []romm.Collection

	if input.Config.ShowCollections {
		collections = append(collections, regularCollections...)
	}

	if input.Config.ShowSmartCollections {
		collections = append(collections, smartCollections...)
	}

	if input.Config.ShowVirtualCollections {
		for _, vc := range virtualCollections {
			collections = append(collections, vc.ToCollection())
		}
	}

	// Sort collections alphabetically
	slices.SortFunc(collections, func(a, b romm.Collection) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	displayCollections := collections
	if input.SearchFilter != "" {
		filteredCollections := make([]romm.Collection, 0)
		for _, collection := range collections {
			if strings.Contains(strings.ToLower(collection.Name), strings.ToLower(input.SearchFilter)) {
				filteredCollections = append(filteredCollections, collection)
			}
		}
		displayCollections = filteredCollections
	}

	if len(displayCollections) == 0 {
		return withCode(output, gaba.ExitCode(404)), nil
	}

	var menuItems []gaba.MenuItem
	for _, collection := range displayCollections {
		menuItems = append(menuItems, gaba.MenuItem{
			Text:     collection.Name,
			Selected: false,
			Focused:  false,
			Metadata: collection,
		})
	}

	footerItems := []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: i18n.GetString("button_back")},
		{ButtonName: "X", HelpText: i18n.GetString("button_search")},
		{ButtonName: "A", HelpText: i18n.GetString("button_select")},
	}

	title := "Collections"
	if input.SearchFilter != "" {
		title = fmt.Sprintf("[Search: \"%s\"] | Collections", input.SearchFilter)
	}

	options := gaba.DefaultListOptions(title, menuItems)
	options.ActionButton = buttons.VirtualButtonX
	options.FooterHelpItems = footerItems
	options.SelectedIndex = input.LastSelectedIndex
	options.VisibleStartIndex = max(0, input.LastSelectedIndex-input.LastSelectedPosition)

	sel, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			if input.SearchFilter != "" {
				output.SearchFilter = ""
				output.LastSelectedIndex = 0
				output.LastSelectedPosition = 0
				return withCode(output, constants.ExitCodeClearSearch), nil
			}
			return back(output), nil
		}
		return withCode(output, gaba.ExitCodeError), err
	}

	switch sel.Action {
	case gaba.ListActionSelected:
		collection := sel.Items[sel.Selected[0]].Metadata.(romm.Collection)

		output.SelectedCollection = collection
		output.LastSelectedIndex = sel.Selected[0]
		output.LastSelectedPosition = sel.VisiblePosition
		return success(output), nil

	case gaba.ListActionTriggered:
		return withCode(output, constants.ExitCodeSearch), nil

	default:
		return withCode(output, gaba.ExitCodeBack), nil
	}
}
