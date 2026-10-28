package gamepad

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	vendorID = 0x2dc8

	evKey = 0x01
	evAbs = 0x03

	absHat0X = 0x10
	absHat0Y = 0x11

	btnSouth = 304

	eviocgid     = 0x80084502 // EVIOCGID
	eviocgbitKey = 0x80604521 // EVIOCGBIT(EV_KEY, 96)
)

// Product IDs of supported controllers, in DInput mode.
var models = map[uint16]string{
	0x6012: "8BitDo Ultimate 2 Wireless",
}

type inputID struct {
	Bustype, Vendor, Product, Version uint16
}

// Device is an open evdev node of a supported controller.
type Device struct {
	Name string
	Path string

	f          *os.File
	hatX, hatY int32
}

// Find opens the first supported controller among the evdev devices. Nodes
// that can't be opened (usually for lack of permission) are skipped.
func Find() (*Device, bool) {
	paths, _ := filepath.Glob("/dev/input/event*")

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		if name, ok := identify(f); ok {
			return &Device{Name: name, Path: path, f: f}, true
		}

		f.Close()
	}

	return nil, false
}

func identify(f *os.File) (string, bool) {
	var id inputID
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), eviocgid, uintptr(unsafe.Pointer(&id))); errno != 0 {
		return "", false
	}

	name, ok := models[id.Product]
	if id.Vendor != vendorID || !ok {
		return "", false
	}

	// A controller can expose more than one node (motion sensors, for
	// instance); the buttons live in the one that reports BTN_SOUTH.
	var keys [96]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), eviocgbitKey, uintptr(unsafe.Pointer(&keys))); errno != 0 {
		return "", false
	}

	return name, keys[btnSouth/8]&(1<<(btnSouth%8)) != 0
}

// Run reads events until the device fails (for example, when it's unplugged
// or turned off), reporting button state changes to fn.
func (d *Device) Run(fn func(code uint16, pressed bool)) error {
	tvSize := int(unsafe.Sizeof(unix.Timeval{}))
	evSize := tvSize + 8

	buf := make([]byte, 64*evSize)

	for {
		n, err := d.f.Read(buf)
		if err != nil {
			return err
		}

		for off := 0; off+evSize <= n; off += evSize {
			ev := buf[off+tvSize : off+evSize]
			typ := binary.LittleEndian.Uint16(ev)
			code := binary.LittleEndian.Uint16(ev[2:])
			value := int32(binary.LittleEndian.Uint32(ev[4:]))

			switch {
			case typ == evKey && value != 2: // 2 is autorepeat
				fn(code, value == 1)
			case typ == evAbs && code == absHat0X:
				hat(&d.hatX, value, codeDpadLeft, codeDpadRight, fn)
			case typ == evAbs && code == absHat0Y:
				hat(&d.hatY, value, codeDpadUp, codeDpadDown, fn)
			}
		}
	}
}

// hat turns a hat axis change into release/press events for the two
// directions of the axis.
func hat(last *int32, value int32, neg, pos uint16, fn func(code uint16, pressed bool)) {
	if *last == value {
		return
	}

	if *last < 0 {
		fn(neg, false)
	} else if *last > 0 {
		fn(pos, false)
	}

	if value < 0 {
		fn(neg, true)
	} else if value > 0 {
		fn(pos, true)
	}

	*last = value
}

func (d *Device) Close() error {
	return d.f.Close()
}
