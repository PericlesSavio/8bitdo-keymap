package commands

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/PericlesSavio/8bitdoctl/internal/gamepad"
	"github.com/PericlesSavio/8bitdoctl/internal/keymap"
	"github.com/spf13/cobra"
)

var mappingCommand = &cobra.Command{
	Use:   "mapping",
	Short: "Manage keyboard mappings",
}

var mappingListCommand = &cobra.Command{
	Use:   "list",
	Short: "List the buttons and the keyboard key each one types",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl, err := daemon()
		if err != nil {
			return err
		}

		mappings, err := cl.GetKeyboardMappings()
		if err != nil {
			return fmt.Errorf("get keyboard mappings: %w", err)
		}

		if terseOutput {
			for _, b := range gamepad.Buttons {
				fmt.Println(b.Name, orDash(mappings[b.Name]))
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "BUTTON\tKEYBOARD")

		for _, b := range gamepad.Buttons {
			fmt.Fprintf(w, "%s\t%s\n", b.Name, orDash(mappings[b.Name]))
		}

		return w.Flush()
	},
}

var mappingSetCommand = &cobra.Command{
	Use:   "set <button> <key>",
	Short: `Make a button type a keyboard key ("f5" or "key:f5"), or "none" to stop it`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		button, ok := gamepad.ButtonByName(args[0])
		if !ok {
			return fmt.Errorf("unknown button %q, valid buttons are: %s", args[0], strings.Join(gamepad.ButtonNames(), ", "))
		}

		cl, err := daemon()
		if err != nil {
			return err
		}

		keyName := strings.TrimPrefix(strings.ToLower(args[1]), "key:")

		if keyName == "none" {
			if err := cl.RemoveKeyboardMapping(button.Name); err != nil {
				return fmt.Errorf("remove keyboard mapping: %w", err)
			}

			if !terseOutput {
				fmt.Printf("%s no longer types a keyboard key\n", button.Name)
			}
			return nil
		}

		canonical, _, ok := keymap.KeyCodeByName(keyName)
		if !ok {
			return fmt.Errorf(`unknown keyboard key %q, run "8bitdoctl mapping keys" to list them`, keyName)
		}

		if err := cl.SetKeyboardMapping(button.Name, canonical); err != nil {
			return fmt.Errorf("set keyboard mapping: %w", err)
		}

		if !terseOutput {
			fmt.Printf("%s now types the %q key\n", button.Name, canonical)
		}
		return nil
	},
}

var mappingResetAll bool

var mappingResetCommand = &cobra.Command{
	Use:   "reset [button]",
	Short: "Clear the keyboard mapping of a button (or of all buttons, with --all)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if (len(args) == 0) == !mappingResetAll {
			return errors.New("specify either a button or --all")
		}

		cl, err := daemon()
		if err != nil {
			return err
		}

		if mappingResetAll {
			return cl.ClearKeyboardMappings()
		}

		button, ok := gamepad.ButtonByName(args[0])
		if !ok {
			return fmt.Errorf("unknown button %q", args[0])
		}

		return cl.RemoveKeyboardMapping(button.Name)
	},
}

var mappingKeysCommand = &cobra.Command{
	Use:   "keys",
	Short: `List the keyboard key names accepted by "mapping set"`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		names := keymap.KeyNames()
		sort.Strings(names)

		for _, name := range names {
			fmt.Println(name)
		}
	},
}

var mappingIdentifyOnce bool

var mappingIdentifyCommand = &cobra.Command{
	Use:   "identify",
	Short: "Print the name of each button as you press it (Ctrl+C to stop)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl, err := daemon()
		if err != nil {
			return err
		}

		if mappingIdentifyOnce {
			button, err := cl.WaitButtonPress(10000)
			if err != nil {
				return fmt.Errorf("wait for button press: %w", err)
			}
			if button == "" {
				return errors.New("no button pressed")
			}

			fmt.Println(button)
			return nil
		}

		if !terseOutput {
			fmt.Println("press buttons on the controller to identify them, Ctrl+C to stop")
		}

		for {
			button, err := cl.WaitButtonPress(60000)
			if err != nil {
				return fmt.Errorf("wait for button press: %w", err)
			}

			if button != "" {
				fmt.Println(button)
			}
		}
	},
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}

func init() {
	mappingIdentifyCommand.Flags().BoolVar(&mappingIdentifyOnce, "once", false, "wait for a single button press (up to 10s) and exit")
	mappingResetCommand.Flags().BoolVar(&mappingResetAll, "all", false, "reset all buttons")

	mappingCommand.AddCommand(mappingListCommand)
	mappingCommand.AddCommand(mappingSetCommand)
	mappingCommand.AddCommand(mappingResetCommand)
	mappingCommand.AddCommand(mappingKeysCommand)
	mappingCommand.AddCommand(mappingIdentifyCommand)

	rootCmd.AddCommand(mappingCommand)
}
