package keymap

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	uiSetEvBit   = 0x40045564
	uiSetKeyBit  = 0x40045565
	uiDevSetup   = 0x405c5503
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502

	evSyn = 0x00
	evKey = 0x01
	evRep = 0x14

	maxKeyCode = 255
)

type uinputSetup struct {
	Bustype, Vendor, Product, Version uint16
	Name                              [80]byte
	FFEffectsMax                      uint32
}

type virtualKeyboard struct {
	f *os.File
}

func newVirtualKeyboard() (*virtualKeyboard, error) {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY|unix.O_NONBLOCK, 0)
	if os.IsNotExist(err) {
		// The system modprobe handles the compressed modules that the Go
		// library can't load.
		if out, merr := exec.Command("modprobe", "uinput").CombinedOutput(); merr != nil {
			return nil, fmt.Errorf("load uinput module: %w (%s)", merr, out)
		}

		f, err = os.OpenFile("/dev/uinput", os.O_WRONLY|unix.O_NONBLOCK, 0)
	}
	if err != nil {
		return nil, fmt.Errorf("open uinput device: %w", err)
	}

	fd := int(f.Fd())

	if err := unix.IoctlSetInt(fd, uiSetEvBit, evKey); err != nil {
		f.Close()
		return nil, fmt.Errorf("enable key events: %w", err)
	}
	if err := unix.IoctlSetInt(fd, uiSetEvBit, evRep); err != nil {
		f.Close()
		return nil, fmt.Errorf("enable key repeat: %w", err)
	}

	for code := 1; code <= maxKeyCode; code++ {
		if err := unix.IoctlSetInt(fd, uiSetKeyBit, code); err != nil {
			f.Close()
			return nil, fmt.Errorf("enable key code %d: %w", code, err)
		}
	}

	// Deliberately not the controller's vendor/product: tools that match
	// devices by ID (Steam, SDL) must not mistake the keyboard for the pad.
	setup := uinputSetup{
		Bustype: unix.BUS_VIRTUAL,
		Vendor:  0x0001,
		Product: 0x0001,
		Version: 1,
	}
	copy(setup.Name[:], "8bitdoctl keyboard")

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uiDevSetup, uintptr(unsafe.Pointer(&setup))); errno != 0 {
		f.Close()
		return nil, fmt.Errorf("setup uinput device: %w", errno)
	}

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uiDevCreate, 0); errno != 0 {
		f.Close()
		return nil, fmt.Errorf("create uinput device: %w", errno)
	}

	return &virtualKeyboard{f: f}, nil
}

func (k *virtualKeyboard) SendKey(code uint16, press bool) error {
	var value uint32
	if press {
		value = 1
	}

	// Two input_event structs: the key event followed by a SYN_REPORT. The
	// leading timeval of each is left zeroed; the input core stamps events on
	// delivery.
	tvSize := int(unsafe.Sizeof(unix.Timeval{}))
	evSize := tvSize + 8

	buf := make([]byte, 2*evSize)
	binary.LittleEndian.PutUint16(buf[tvSize:], evKey)
	binary.LittleEndian.PutUint16(buf[tvSize+2:], code)
	binary.LittleEndian.PutUint32(buf[tvSize+4:], value)
	binary.LittleEndian.PutUint16(buf[evSize+tvSize:], evSyn)

	_, err := k.f.Write(buf)
	return err
}

func (k *virtualKeyboard) Close() error {
	unix.Syscall(unix.SYS_IOCTL, k.f.Fd(), uiDevDestroy, 0)
	return k.f.Close()
}
