package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/PericlesSavio/8bitdoctl/internal/dbus/server"
	"github.com/PericlesSavio/8bitdoctl/internal/keymap"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	prettyLogging := flag.Bool("pretty-logs", false, "Enable human-readable colored logs")
	configPath := flag.String("config", "", "Keyboard mappings file (default: $XDG_CONFIG_HOME/8bitdoctl/keymap.json)")
	flag.Parse()

	if *prettyLogging {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	if *configPath == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to find the config directory")
		}

		*configPath = filepath.Join(dir, "8bitdoctl", "keymap.json")
	}

	km := keymap.NewManager(*configPath)

	if err := server.New(km).Listen(); err != nil {
		log.Fatal().Err(err).Msg("failed to start dbus server")
	}
}
