package main

import (
	"fmt"
	"grout/romm"
	"grout/utils"
	"os"
	"slices"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

func main() {
	gaba.Init(gaba.Options{
		WindowTitle: "CLI",
		LogFilename: "cli.log",
	})

	rc := romm.NewClient("http://192.168.1.20:1550",
		romm.WithBasicAuth(os.Getenv("DEV_ROMM_USERNAME"),
			os.Getenv("DEV_ROMM_PASSWORD")))

	scanLocal := utils.ScanAllRoms()

	plats, _ := rc.GetPlatforms()

	for slug, localRoms := range scanLocal {
		idx := slices.IndexFunc(plats, func(p romm.Platform) bool {
			return p.Slug == slug
		})

		platform := plats[idx]

		remoteSaves, _ := rc.GetSavesByRomForPlatform(platform.ID)

		roms, _ := rc.GetRoms(&romm.GetRomsOptions{PlatformID: &platform.ID})
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

	for slug, roms := range scanLocal {
		fmt.Println(slug + "\n------------")
		for _, r := range roms {
			hasLocalSave := r.SaveFile != nil
			hasRemoteSave := len(r.RemoteSaves) > 0
			fmt.Println(fmt.Sprintf("Game: %s\n\tLocal Save: %t\n\tRemote Save: %t\n\tSync Action: %s\n",
				r.RomName,
				hasLocalSave,
				hasRemoteSave,
				r.SyncAction()))
		}
	}

}
