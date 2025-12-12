package utils

import (
	"fmt"
	"grout/romm"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

type SaveSync struct {
	RomID    int
	Slug     string
	GameBase string
	Local    *LocalSave
	Remote   romm.Save
	Action   SyncAction
}

type SyncAction string

const (
	Download SyncAction = "DOWNLOAD"
	Upload              = "UPLOAD"
	Skip                = "SKIP"
)

func (s SaveSync) Execute(host romm.Host) error {
	switch s.Action {
	case Upload:
		return s.upload(host)
	case Download:
		if s.Local != nil {
			err := s.Local.Backup()
			if err != nil {
				return err
			}
		}
		return s.download(host)
	}

	return nil
}

func (s SaveSync) download(host romm.Host) error {
	logger := gaba.GetLogger()
	rc := GetRommClient(host)

	logger.Debug("Downloading save", "downloadPath", s.Remote.DownloadPath)

	// Download the save file using the DownloadPath from the API
	saveData, err := rc.DownloadSave(s.Remote.DownloadPath)
	if err != nil {
		return fmt.Errorf("failed to download save: %w", err)
	}

	// Determine the destination directory
	var destDir string
	if s.Local != nil {
		// Use the directory from existing local save
		destDir = filepath.Dir(s.Local.Path)
	} else {
		// Get the save directory for this platform
		saveFiles := FindSaveFiles(s.Slug)
		if len(saveFiles) > 0 {
			// Use the directory from any existing save file
			destDir = filepath.Dir(saveFiles[0].Path)
		} else {
			// No existing saves, need to determine the save directory
			return fmt.Errorf("cannot determine save location for slug %s: no existing save files", s.Slug)
		}
	}

	// Always use the full ROM name (with region tags) for the save filename
	ext := s.Remote.FileExtension
	filename := s.GameBase + ext
	destPath := filepath.Join(destDir, filename)

	// If we have a local save with a different name, remove it after backup
	if s.Local != nil && s.Local.Path != destPath {
		defer os.Remove(s.Local.Path)
	}

	// Write the downloaded file
	err = os.WriteFile(destPath, saveData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	return nil
}

func (s SaveSync) upload(host romm.Host) error {
	if s.Local == nil {
		return fmt.Errorf("cannot upload: no local save file")
	}

	rc := GetRommClient(host)

	// Create a temp file with the game's full name (includes region info)
	ext := filepath.Ext(s.Local.Path)
	filename := s.GameBase + ext
	tmp := filepath.Join(TempDir(), "uploads", filename)

	err := CopyFile(s.Local.Path, tmp)
	if err != nil {
		return err
	}

	uploadedSave, err := rc.UploadSave(s.RomID, tmp)
	if err != nil {
		return err
	}

	// Update local file's modification time to match server's UpdatedAt
	// This ensures future comparisons are accurate
	err = os.Chtimes(s.Local.Path, uploadedSave.UpdatedAt, uploadedSave.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update file timestamp: %w", err)
	}

	return nil
}

func FindSaveSyncs(host romm.Host) ([]SaveSync, error) {
	rc := GetRommClient(host)

	scanLocal := ScanAllRoms()

	plats, err := rc.GetPlatforms()
	if err != nil {
		gaba.GetLogger().Error("Could not retrieve platforms")
		return []SaveSync{}, err
	}

	for slug, localRoms := range scanLocal {
		idx := slices.IndexFunc(plats, func(p romm.Platform) bool {
			return p.Slug == slug
		})

		platform := plats[idx]

		remoteSaves, err := rc.GetSavesByRomForPlatform(platform.ID)
		if err != nil {
			gaba.GetLogger().Error("Could not retrieve remote saves", "platform", platform)
			continue
		}

		roms, err := rc.GetRoms(&romm.GetRomsOptions{PlatformID: &platform.ID})
		if err != nil {
			gaba.GetLogger().Error("Could not retrieve roms", "platform", platform)
			continue
		}

		for idx, localRom := range localRoms {
			hashMatchIdx := slices.IndexFunc(roms.Items, func(rom romm.Rom) bool {
				return rom.Sha1Hash == localRom.SHA1
			})
			if hashMatchIdx == -1 {
				continue
			}

			remoteRom := roms.Items[hashMatchIdx]
			scanLocal[slug][idx].RomID = remoteRom.ID
			scanLocal[slug][idx].RomName = remoteRom.Name

			if saves, ok := remoteSaves[remoteRom.ID]; ok {
				if len(*saves) > 0 {
					scanLocal[slug][idx].RemoteSaves = *saves
				}
			}
		}
	}

	var syncs []SaveSync

	for _, roms := range scanLocal {
		for _, r := range roms {
			action := r.SyncAction()
			switch action {
			case Upload, Download:
				base := strings.ReplaceAll(r.FileName, filepath.Ext(r.FileName), "")
				lastRemoteSave := r.LastRemoteSave()
				syncs = append(syncs, SaveSync{
					RomID:    r.RomID,
					Slug:     r.Slug,
					GameBase: base,
					Local:    r.SaveFile,
					Remote:   lastRemoteSave,
					Action:   action,
				})

			}
		}
	}

	return syncs, nil
}
