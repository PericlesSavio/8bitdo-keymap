package commands

import (
	"fmt"

	"github.com/PericlesSavio/8bitdoctl/internal/version"
	"github.com/spf13/cobra"
)

var statusCommand = &cobra.Command{
	Use:   "status",
	Short: "Show whether the controller is connected (terse: its name, or an empty line)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl, err := daemon()
		if err != nil {
			return err
		}

		device, err := cl.GetDevice()
		if err != nil {
			return fmt.Errorf("get device: %w", err)
		}

		switch {
		case terseOutput:
			fmt.Println(device)
		case device == "":
			fmt.Println("no controller connected (it must be in DInput mode)")
		default:
			fmt.Printf("connected: %s\n", device)
		}
		return nil
	},
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Show version information for 8bitdoctl and 8bitdod",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("8bitdoctl version %s\n", version.Version)

		ver := ""
		cl, err := daemon()
		if err == nil {
			ver, err = cl.GetServerVersion()
		}

		if err != nil {
			fmt.Println("8bitdod not running")
		} else {
			fmt.Printf("8bitdod version %s\n", ver)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCommand)
	rootCmd.AddCommand(versionCommand)
}
