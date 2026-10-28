package commands

import (
	"os"

	"github.com/PericlesSavio/8bitdoctl/internal/dbus/client"
	"github.com/spf13/cobra"
)

var terseOutput bool

var rootCmd = &cobra.Command{
	Use:          "8bitdoctl",
	Short:        "Map 8BitDo controller buttons to keyboard keys",
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&terseOutput, "terse", "t", false, "output only the queried values with no decoration")
}

var dbusClient *client.Client

// daemon connects to the session bus on first use, so that commands that
// don't need the daemon (completion, keys) work without a bus.
func daemon() (*client.Client, error) {
	if dbusClient != nil {
		return dbusClient, nil
	}

	cl, err := client.Dial()
	if err != nil {
		return nil, err
	}

	dbusClient = cl
	return cl, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
