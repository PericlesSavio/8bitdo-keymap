package gamepad

import (
	"fmt"
	"strings"
)

// The D-pad is reported as a hat axis instead of buttons, so its directions
// get codes above KEY_MAX (0x2ff) to share the button code space.
const (
	codeDpadUp uint16 = 0x300 + iota
	codeDpadDown
	codeDpadLeft
	codeDpadRight
)

type Button struct {
	Name string
	Code uint16 // evdev key code, from linux/input-event-codes.h
}

// Buttons lists the Ultimate 2 Wireless buttons in DInput mode, in display
// order. The codes match SDL's gamecontrollerdb entry for the controller, and
// the extra buttons were confirmed by pressing them.
var Buttons = []Button{
	{"L4", 704}, {"R4", 705}, {"PL", 309}, {"PR", 306},
	{"A", 304}, {"B", 305}, {"X", 307}, {"Y", 308},
	{"LB", 310}, {"RB", 311}, {"LT", 312}, {"RT", 313},
	{"SELECT", 314}, {"START", 315}, {"HOME", 316},
	{"LS", 317}, {"RS", 318},
	{"UP", codeDpadUp}, {"DOWN", codeDpadDown}, {"LEFT", codeDpadLeft}, {"RIGHT", codeDpadRight},
}

// ButtonByName looks up a button by its name, case-insensitively.
func ButtonByName(name string) (Button, bool) {
	for _, b := range Buttons {
		if strings.EqualFold(b.Name, name) {
			return b, true
		}
	}

	return Button{}, false
}

// ButtonName returns the name of a button code, or "#<code>" for codes that
// aren't in the button list.
func ButtonName(code uint16) string {
	for _, b := range Buttons {
		if b.Code == code {
			return b.Name
		}
	}

	return fmt.Sprintf("#%d", code)
}

func ButtonNames() []string {
	names := make([]string, len(Buttons))
	for i, b := range Buttons {
		names[i] = b.Name
	}

	return names
}
