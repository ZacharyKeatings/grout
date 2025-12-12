package main

import (
	"fmt"
	"grout/romm"
	"grout/utils"
	"log/slog"
	"os"

	gaba "github.com/UncleJunVIP/gabagool/v2/pkg/gabagool"
)

func main() {
	gaba.Init(gaba.Options{
		WindowTitle: "CLI",
		LogFilename: "cli.log",
	})

	gaba.SetLogLevel(slog.LevelDebug)

	host := romm.Host{
		RootURI:  "http://192.168.1.20",
		Port:     1550,
		Username: os.Getenv("DEV_ROMM_USERNAME"),
		Password: os.Getenv("DEV_ROMM_PASSWORD"),
	}

	syncs, _ := utils.FindSaveSyncs(host)

	for _, s := range syncs {
		err := s.Execute(host)
		fmt.Println(err.Error())
	}
}
