package ui

import (
	"errors"
	"fmt"
	"grout/romm"
	"grout/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	groutConstants "grout/constants"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type GameDetailsInput struct {
	Config   *utils.Config
	Host     romm.Host
	Platform romm.Platform
	Game     romm.Rom
}

type GameDetailsOutput struct {
	DownloadRequested bool
	Game              romm.Rom
	Platform          romm.Platform
}

type GameDetailsScreen struct{}

func NewGameDetailsScreen() *GameDetailsScreen {
	return &GameDetailsScreen{}
}

func (s *GameDetailsScreen) Draw(input GameDetailsInput) (ScreenResult[GameDetailsOutput], error) {
	logger := gaba.GetLogger()
	output := GameDetailsOutput{
		Game:     input.Game,
		Platform: input.Platform,
	}

	sections := s.buildSections(input)

	options := gaba.DefaultInfoScreenOptions()
	options.Sections = sections
	options.ShowThemeBackground = false
	options.ShowScrollbar = true

	result, err := gaba.DetailScreen(input.Game.Name, options, []gaba.FooterHelpItem{
		{ButtonName: "B", HelpText: i18n.GetString("button_back")},
		{ButtonName: "A", HelpText: i18n.GetString("button_download")},
	})

	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return back(output), nil
		}
		logger.Error("Detail screen error", "error", err)
		return withCode(output, gaba.ExitCodeError), err
	}

	if result.Action == gaba.DetailActionConfirmed {
		output.DownloadRequested = true
		return success(output), nil
	}

	return back(output), nil
}

func (s *GameDetailsScreen) buildSections(input GameDetailsInput) []gaba.Section {
	sections := make([]gaba.Section, 0)
	game := input.Game
	logger := gaba.GetLogger()

	coverImageData := s.fetchCoverImage(input.Host, game)
	if coverImageData != nil {
		detailsDir := filepath.Join(utils.TempDir(), "details")
		tempImagePath := filepath.Join(detailsDir, fmt.Sprintf("grout_cover_%d.jpg", game.ID))

		if err := os.MkdirAll(detailsDir, 0755); err != nil {
			logger.Warn("Failed to create details directory", "error", err)
		} else {
			if err := os.Chmod(detailsDir, 0755); err != nil {
				logger.Warn("Failed to set directory permissions", "error", err)
			}

			err := os.WriteFile(tempImagePath, coverImageData, 0644)
			if err == nil {
				utils.ProcessArtImage(tempImagePath)
				sections = append(sections, gaba.NewImageSection("", tempImagePath, 640, 480, constants.TextAlignCenter))
			} else {
				logger.Warn("Failed to write cover image", "error", err)
			}
		}
	}

	if game.Summary != "" {
		sections = append(sections, gaba.NewDescriptionSection("", game.Summary))
	}

	metadata := make([]gaba.MetadataItem, 0)

	if game.Metadatum.FirstReleaseDate > 0 {
		releaseDate := time.Unix(game.Metadatum.FirstReleaseDate/1000, 0)
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_release_date"),
			Value: releaseDate.Format("January 2, 2006"),
		})
	}

	if game.Metadatum.AverageRating > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_average_rating"),
			Value: fmt.Sprintf("%.1f/100", game.Metadatum.AverageRating),
		})
	}

	if len(game.Metadatum.Genres) > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_genres"),
			Value: strings.Join(game.Metadatum.Genres, ", "),
		})
	}

	if len(game.Metadatum.Companies) > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_companies"),
			Value: strings.Join(game.Metadatum.Companies, ", "),
		})
	}

	if len(game.Metadatum.GameModes) > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_game_modes"),
			Value: strings.Join(game.Metadatum.GameModes, ", "),
		})
	}

	if len(game.Regions) > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_regions"),
			Value: strings.Join(game.Regions, ", "),
		})
	}

	if len(game.Languages) > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_languages"),
			Value: strings.Join(game.Languages, ", "),
		})
	}

	if game.FsSizeBytes > 0 {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_file_size"),
			Value: utils.FormatBytes(game.FsSizeBytes),
		})
	}

	if game.Multi {
		metadata = append(metadata, gaba.MetadataItem{
			Label: i18n.GetString("game_details_type"),
			Value: i18n.GetString("game_details_multi_file_rom"),
		})
	}

	if len(metadata) > 0 {
		sections = append(sections, gaba.NewInfoSection("", metadata))
	}

	if len(sections) == 0 {
		logger.Warn("No sections available for game", "game", game.Name)
		sections = append(sections, gaba.NewInfoSection("", []gaba.MetadataItem{
			{Label: i18n.GetString("game_details_game"), Value: game.Name},
			{Label: i18n.GetString("game_details_platform"), Value: game.PlatformDisplayName},
		}))
	}

	qrcode, err := utils.CreateTempQRCode(game.GetGamePage(input.Host), 256)
	if err == nil {
		sections = append(sections, gaba.NewImageSection(
			i18n.GetString("game_details_qr_section"),
			qrcode,
			int32(256),
			int32(256),
			constants.TextAlignCenter,
		))

	} else {
		logger.Error("Unable to generate QR code", "error", err)
	}

	return sections
}

func (s *GameDetailsScreen) fetchCoverImage(host romm.Host, game romm.Rom) []byte {
	logger := gaba.GetLogger()

	// First, check if artwork is in the cache
	if utils.ArtworkExists(game.PlatformSlug, game.ID) {
		cachePath := utils.GetArtworkCachePath(game.PlatformSlug, game.ID)
		data, err := os.ReadFile(cachePath)
		if err == nil {
			logger.Debug("Using cached artwork for game details", "game", game.Name)
			return data
		}
		logger.Debug("Failed to read cached artwork, will fetch from server", "error", err)
	}

	// Not in cache, fetch from server
	var coverPath string
	if game.PathCoverLarge != "" {
		coverPath = game.PathCoverLarge
	} else if game.PathCoverSmall != "" {
		coverPath = game.PathCoverSmall
	} else if game.URLCover != "" {
		coverPath = game.URLCover
	} else {
		return nil
	}

	coverURL := host.URL() + coverPath
	imageData := s.fetchImageFromURL(host, coverURL)

	// Cache the artwork for future use
	if imageData != nil {
		if err := utils.EnsureArtworkCacheDir(game.PlatformSlug); err == nil {
			cachePath := utils.GetArtworkCachePath(game.PlatformSlug, game.ID)
			if err := os.WriteFile(cachePath, imageData, 0644); err == nil {
				utils.ProcessArtImage(cachePath)
				logger.Debug("Cached artwork from game details", "game", game.Name)
			}
		}
	}

	return imageData
}

func (s *GameDetailsScreen) fetchImageFromURL(host romm.Host, imageURL string) []byte {
	logger := gaba.GetLogger()

	imageURL = strings.ReplaceAll(imageURL, " ", "%20")

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		logger.Warn("Failed to create image request", "url", imageURL, "error", err)
		return nil
	}

	req.SetBasicAuth(host.Username, host.Password)

	client := &http.Client{Timeout: groutConstants.DefaultHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Failed to fetch image", "url", imageURL, "error", err)
		return nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("Image fetch failed with bad status", "url", imageURL, "status", resp.Status)
		return nil
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("Failed to read image data", "error", err)
		return nil
	}

	logger.Debug("Successfully fetched image", "url", imageURL, "size", len(imageData))
	return imageData
}
