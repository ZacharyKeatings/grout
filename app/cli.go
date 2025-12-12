package main

import (
	"grout/romm"
	"grout/utils"
	"os"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

func main() {
	gaba.Init(gaba.Options{
		WindowTitle: "CLI",
		LogFilename: "cli.log",
	})

	syncs, _ := utils.FindSaveSyncs(romm.Host{
		RootURI:  "http://192.168.1.20",
		Port:     1550,
		Username: os.Getenv("DEV_ROMM_USERNAME"),
		Password: os.Getenv("DEV_ROMM_PASSWORD"),
	})

	for _, s := range syncs {
		if s.Action == utils.Download {
			s.Local.Backup()
		}
	}
}
