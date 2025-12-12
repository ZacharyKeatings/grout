package romm

import (
	"os"
	"time"
)

type Save struct {
	ID             int       `json:"id"`
	RomID          int       `json:"rom_id"`
	UserID         int       `json:"user_id"`
	FileName       string    `json:"file_name"`
	FileNameNoTags string    `json:"file_name_no_tags"`
	FileNameNoExt  string    `json:"file_name_no_ext"`
	FileExtension  string    `json:"file_extension"`
	FilePath       string    `json:"file_path"`
	FileSizeBytes  int       `json:"file_size_bytes"`
	FullPath       string    `json:"full_path"`
	DownloadPath   string    `json:"download_path"`
	MissingFromFs  bool      `json:"missing_from_fs"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Emulator       string    `json:"emulator"`
	Screenshot     struct {
		ID             int       `json:"id"`
		RomID          int       `json:"rom_id"`
		UserID         int       `json:"user_id"`
		FileName       string    `json:"file_name"`
		FileNameNoTags string    `json:"file_name_no_tags"`
		FileNameNoExt  string    `json:"file_name_no_ext"`
		FileExtension  string    `json:"file_extension"`
		FilePath       string    `json:"file_path"`
		FileSizeBytes  int       `json:"file_size_bytes"`
		FullPath       string    `json:"full_path"`
		DownloadPath   string    `json:"download_path"`
		MissingFromFs  bool      `json:"missing_from_fs"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	} `json:"screenshot"`
}

type SaveQuery struct {
	RomID      int `json:"rom_id,omitempty"`
	PlatformID int `json:"platform_id,omitempty"`
}

func (sq SaveQuery) Valid() bool {
	return sq.RomID != 0 || sq.PlatformID != 0
}

func (c *Client) GetSaves(query SaveQuery) (*[]Save, error) {
	var saves []Save
	err := c.doRequest("GET", EndpointSaves, query, nil, &saves)
	return &saves, err
}

func (c *Client) GetSavesByRomForPlatform(platformID int) (map[int]*[]Save, error) {
	saves, err := c.GetSaves(SaveQuery{PlatformID: platformID})
	if err != nil {
		return nil, err
	}

	res := make(map[int]*[]Save)

	for _, save := range *saves {
		ptr := res[save.RomID]
		if ptr == nil {
			var s []Save
			res[save.RomID] = &s
			ptr = res[save.RomID]
		}
		*ptr = append(*ptr, save)
	}

	return res, nil
}

func (c *Client) UploadSave(romID int, savePath string) (*Save, error) {
	file, err := os.Open(savePath)
	if err != nil {
		return nil, err
	}

	var res *Save
	err = c.doMultipartRequest("POST", EndpointSaves, SaveQuery{RomID: romID}, file, "", res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
