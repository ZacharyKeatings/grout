package romm

import (
	"fmt"
	"time"
)

type Platform struct {
	ID                  int        `json:"id"`
	Slug                string     `json:"slug"`
	FSSlug              string     `json:"fs_slug"`
	Name                string     `json:"name"`
	ShortName           string     `json:"short_name"`
	LogoPath            string     `json:"logo_path"`
	ROMCount            int        `json:"rom_count"`
	Firmware            []Firmware `json:"Firmware"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Manufacturer        string     `json:"manufacturer"`
	Generation          int        `json:"generation"`
	Type                string     `json:"type"`
	HasBIOS             bool       `json:"has_bios"`
	SupportedExtensions []string   `json:"supported_extensions"`
}

func (c *Client) GetPlatforms() ([]Platform, error) {
	var platforms []Platform
	err := c.doRequest("GET", endpointPlatforms, nil, nil, &platforms)
	return platforms, err
}

func (c *Client) GetPlatform(id int) (Platform, error) {
	var platform Platform
	path := fmt.Sprintf(endpointPlatformByID, id)
	err := c.doRequest("GET", path, nil, nil, &platform)
	return platform, err
}
