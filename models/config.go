package models

import (
	"time"
)

type Config struct {
	Hosts             Hosts                       `json:"hosts,omitempty"`
	DirectoryMappings map[string]DirectoryMapping `json:"directory_mappings,omitempty"`
	ApiTimeout        time.Duration               `json:"api_timeout"`
	DownloadTimeout   time.Duration               `json:"download_timeout"`
	UnzipDownloads    bool                        `json:"unzip_downloads,omitempty"`
	DownloadArt       bool                        `json:"download_art,omitempty"`
	ShowGameDetails   bool                        `json:"show_game_details"`
	LogLevel          string                      `json:"log_level,omitempty"`
}

func (c Config) ToLoggable() any {
	safeHosts := make([]map[string]any, len(c.Hosts))
	for i, host := range c.Hosts {
		safeHosts[i] = host.ToLoggable()
	}

	return map[string]any{
		"hosts":              safeHosts,
		"directory_mappings": c.DirectoryMappings,
		"api_timeout":        c.ApiTimeout,
		"download_timeout":   c.DownloadTimeout,
		"unzip_downloads":    c.UnzipDownloads,
		"download_art":       c.DownloadArt,
		"show_game_details":  c.ShowGameDetails,
		"log_level":          c.LogLevel,
	}
}
