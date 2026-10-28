package keymap

import "strings"

// Key codes from linux/input-event-codes.h.
var keyCodes = map[string]uint16{
	"esc": 1, "1": 2, "2": 3, "3": 4, "4": 5, "5": 6, "6": 7, "7": 8, "8": 9,
	"9": 10, "0": 11, "minus": 12, "equal": 13, "backspace": 14, "tab": 15,
	"q": 16, "w": 17, "e": 18, "r": 19, "t": 20, "y": 21, "u": 22, "i": 23,
	"o": 24, "p": 25, "leftbrace": 26, "rightbrace": 27, "enter": 28,
	"leftctrl": 29, "a": 30, "s": 31, "d": 32, "f": 33, "g": 34, "h": 35,
	"j": 36, "k": 37, "l": 38, "semicolon": 39, "apostrophe": 40, "grave": 41,
	"leftshift": 42, "backslash": 43, "z": 44, "x": 45, "c": 46, "v": 47,
	"b": 48, "n": 49, "m": 50, "comma": 51, "dot": 52, "slash": 53,
	"rightshift": 54, "kpasterisk": 55, "leftalt": 56, "space": 57,
	"capslock": 58, "f1": 59, "f2": 60, "f3": 61, "f4": 62, "f5": 63,
	"f6": 64, "f7": 65, "f8": 66, "f9": 67, "f10": 68, "numlock": 69,
	"scrolllock": 70, "kp7": 71, "kp8": 72, "kp9": 73, "kpminus": 74,
	"kp4": 75, "kp5": 76, "kp6": 77, "kpplus": 78, "kp1": 79, "kp2": 80,
	"kp3": 81, "kp0": 82, "kpdot": 83, "f11": 87, "f12": 88, "kpenter": 96,
	"rightctrl": 97, "kpslash": 98, "sysrq": 99, "rightalt": 100, "home": 102,
	"up": 103, "pageup": 104, "left": 105, "right": 106, "end": 107,
	"down": 108, "pagedown": 109, "insert": 110, "delete": 111, "mute": 113,
	"volumedown": 114, "volumeup": 115, "kpequal": 117, "pause": 119, "kpcomma": 121,
	"leftmeta": 125, "rightmeta": 126, "compose": 127, "nextsong": 163,
	"playpause": 164, "previoussong": 165, "f13": 183, "f14": 184, "f15": 185,
	"f16": 186, "f17": 187, "f18": 188, "f19": 189, "f20": 190, "f21": 191,
	"f22": 192, "f23": 193, "f24": 194,
}

var keyAliases = map[string]string{
	"ctrl": "leftctrl", "shift": "leftshift", "alt": "leftalt",
	"altgr": "rightalt", "meta": "leftmeta", "super": "leftmeta",
	"win": "leftmeta", "return": "enter", "del": "delete", "ins": "insert",
	"pgup": "pageup", "pgdn": "pagedown", "printscreen": "sysrq",
}

// KeyCodeByName looks up a keyboard key code by its name, case-insensitively.
// It returns the canonical name alongside the code.
func KeyCodeByName(name string) (canonical string, code uint16, ok bool) {
	name = strings.ToLower(name)
	if alias, ok := keyAliases[name]; ok {
		name = alias
	}

	code, ok = keyCodes[name]
	return name, code, ok
}

// KeyNames returns the canonical names of every supported keyboard key.
func KeyNames() []string {
	names := make([]string, 0, len(keyCodes))
	for name := range keyCodes {
		names = append(names, name)
	}

	return names
}
