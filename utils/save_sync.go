package utils

import (
	"grout/romm"
	"path/filepath"
	"slices"
	"strings"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

type SaveSync struct {
	RomID    int
	GameBase string
	Local    LocalSave
	Remote   romm.Save
	Action   SyncAction
}

type SyncAction string

const (
	Download SyncAction = "DOWNLOAD"
	Upload              = "UPLOAD"
	Skip                = "SKIP"
)

func (s SaveSync) Execute() error {
	switch s.Action {
	case Upload:
		return s.upload()
	case Download:
		err := s.Local.Backup()
		if err == nil {
			return s.download()
		}
	}

	return nil
}

func (s SaveSync) download() error {

	return nil
}

func (s SaveSync) upload() error {

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
			switch r.SyncAction() {
			case Upload, Download:
				base := strings.ReplaceAll(r.FileName, filepath.Ext(r.FileName), "")
				lastRemoteSave := r.LastRemoteSave()
				syncs = append(syncs, SaveSync{
					RomID:    r.RomID,
					GameBase: base,
					Local:    *r.SaveFile,
					Remote:   lastRemoteSave,
					Action:   r.SyncAction(),
				})

			}
		}
	}

	return syncs, nil
}
