package ui

import (
	"encoding/base64"
	"fmt"
	"grout/models"
	"grout/utils"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"grout/romm"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

type DownloadInput struct {
	Config        models.Config
	Host          models.Host
	Platform      romm.Platform
	SelectedGames []romm.Rom
	AllGames      []romm.Rom
	SearchFilter  string
}

type DownloadOutput struct {
	DownloadedGames []romm.Rom
	Platform        romm.Platform
	AllGames        []romm.Rom
	SearchFilter    string
}

type DownloadScreen struct{}

func NewDownloadScreen() *DownloadScreen {
	return &DownloadScreen{}
}

func (s *DownloadScreen) Execute(config models.Config, host models.Host, platform romm.Platform, selectedGames []romm.Rom, allGames []romm.Rom, searchFilter string) DownloadOutput {
	result, err := s.Draw(DownloadInput{
		Config:        config,
		Host:          host,
		Platform:      platform,
		SelectedGames: selectedGames,
		AllGames:      allGames,
		SearchFilter:  searchFilter,
	})

	if err != nil {
		gaba.GetLogger().Error("Download failed", "error", err)
		return DownloadOutput{
			AllGames:     allGames,
			Platform:     platform,
			SearchFilter: searchFilter,
		}
	}

	if result.ExitCode == gaba.ExitCodeSuccess && len(result.Value.DownloadedGames) > 0 {
		gaba.GetLogger().Debug("Successfully downloaded games", "count", len(result.Value.DownloadedGames))
	}

	return result.Value
}

func (s *DownloadScreen) Draw(input DownloadInput) (ScreenResult[DownloadOutput], error) {
	logger := gaba.GetLogger()

	output := DownloadOutput{
		Platform:     input.Platform,
		AllGames:     input.AllGames,
		SearchFilter: input.SearchFilter,
	}

	downloads := s.buildDownloads(input.Config, input.Host, input.Platform, input.SelectedGames)

	headers := make(map[string]string)
	auth := input.Host.Username + ":" + input.Host.Password
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	headers["Authorization"] = authHeader

	logger.Debug("RomM Auth Header", "header", authHeader)

	slices.SortFunc(downloads, func(a, b gaba.Download) int {
		return strings.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})

	logger.Debug("Starting ROM download", "downloads", downloads)

	res, err := gaba.DownloadManager(downloads, headers, input.Config.DownloadArt)
	if err != nil {
		logger.Error("Error downloading", "error", err)
		return WithCode(output, gaba.ExitCodeError), err
	}

	if len(res.Failed) > 0 {
		for _, g := range downloads {
			failedMatch := slices.ContainsFunc(res.Failed, func(de gaba.DownloadError) bool {
				return de.Download.DisplayName == g.DisplayName
			})
			if failedMatch {
				utils.DeleteFile(g.Location)
			}
		}
	}

	if len(res.Completed) == 0 {
		return WithCode(output, gaba.ExitCodeError), nil
	}

	// Process multi-file ROM downloads: extract zips and clean up temp files
	for _, g := range input.SelectedGames {
		if !g.Multi {
			continue
		}

		// Check if this multi-file ROM was successfully downloaded
		completed := slices.ContainsFunc(res.Completed, func(d gaba.Download) bool {
			return d.DisplayName == g.Name
		})
		if !completed {
			continue
		}

		// Get the platform for this game
		gamePlatform := input.Platform
		if input.Platform.ID == 0 && g.PlatformID != 0 {
			gamePlatform = romm.Platform{
				ID:   g.PlatformID,
				Slug: g.PlatformSlug,
				Name: g.PlatformDisplayName,
			}
		}

		// Extract the multi-file ROM with a progress message
		tmpZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("grout_multirom_%d.zip", g.ID))
		romDirectory := utils.GetPlatformRomDirectory(input.Config, gamePlatform)
		extractDir := filepath.Join(romDirectory, g.Name)

		_, err := gaba.ProcessMessage(
			fmt.Sprintf("Extracting %s...", g.Name),
			gaba.ProcessMessageOptions{ShowThemeBackground: true},
			func() (interface{}, error) {
				// Read the downloaded zip file
				zipData, err := os.ReadFile(tmpZipPath)
				if err != nil {
					logger.Error("Failed to read multi-file ROM zip", "game", g.Name, "error", err)
					return nil, err
				}

				logger.Debug("Extracting multi-file ROM", "game", g.Name, "dest", extractDir)

				// Extract the zip
				if err := utils.ExtractZip(zipData, extractDir); err != nil {
					logger.Error("Failed to extract multi-file ROM", "game", g.Name, "error", err)
					// Clean up the temp zip file even on error
					os.Remove(tmpZipPath)
					return nil, err
				}

				// Clean up the temp zip file
				if err := os.Remove(tmpZipPath); err != nil {
					logger.Warn("Failed to remove temp zip file", "path", tmpZipPath, "error", err)
				}

				logger.Info("Successfully extracted multi-file ROM", "game", g.Name, "dest", extractDir)
				return nil, nil
			},
		)

		if err != nil {
			continue
		}
	}

	downloadedGames := make([]romm.Rom, 0, len(res.Completed))
	for _, g := range input.SelectedGames {
		if slices.ContainsFunc(res.Completed, func(d gaba.Download) bool {
			return d.DisplayName == g.Name
		}) {
			downloadedGames = append(downloadedGames, g)
		}
	}

	output.DownloadedGames = downloadedGames
	return Success(output), nil
}

func (s *DownloadScreen) buildDownloads(config models.Config, host models.Host, platform romm.Platform, games []romm.Rom) []gaba.Download {
	downloads := make([]gaba.Download, 0, len(games))

	for _, g := range games {
		// For collections, use each game's platform info; for platforms, use the passed platform
		gamePlatform := platform
		if platform.ID == 0 && g.PlatformID != 0 {
			// Construct platform from game's platform info (happens when viewing collections)
			gamePlatform = romm.Platform{
				ID:   g.PlatformID,
				Slug: g.PlatformSlug,
				Name: g.PlatformDisplayName,
			}
		}

		romDirectory := utils.GetPlatformRomDirectory(config, gamePlatform)
		downloadLocation := ""

		sourceURL := ""

		if g.Multi {
			// For multi-file ROMs, download as zip to temp location
			// The zip will be extracted to a folder named after the game
			tmpDir := os.TempDir()
			downloadLocation = filepath.Join(tmpDir, fmt.Sprintf("grout_multirom_%d.zip", g.ID))
			sourceURL, _ = url.JoinPath(host.URL(), "/api/roms/", strconv.Itoa(g.ID), "content", g.Name)
		} else {
			downloadLocation = filepath.Join(romDirectory, g.Files[0].FileName)
			sourceURL, _ = url.JoinPath(host.URL(), "/api/roms/", strconv.Itoa(g.ID), "content", g.Files[0].FileName)
		}

		downloads = append(downloads, gaba.Download{
			URL:         sourceURL,
			Location:    downloadLocation,
			DisplayName: g.Name,
			Timeout:     config.DownloadTimeout,
		})
	}

	return downloads
}
