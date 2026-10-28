# 8BitDo Key Mapper

Maps **8BitDo** controller buttons to keyboard keys on Linux — including the back
buttons (L4, R4, PL, PR), which the firmware only lets you remap with the official
Windows software.

> Unofficial project, not affiliated with or endorsed by 8BitDo.

It follows the same design as [flydigictl](https://github.com/pipe01/flydigictl): a
daemon (`8bitdod`) reads the controller and types the keys through a virtual keyboard
(`/dev/uinput`); the CLI (`8bitdoctl`) and the Qt GUI talk to it over DBus.

Unlike flydigictl, there is **no proprietary protocol and no root**: in DInput mode the
kernel already exposes every button as a separate evdev event, and the daemon runs as
a user service (systemd `--user`, session bus).

Supported controllers:

- 8BitDo Ultimate 2 Wireless (`2dc8:6012`, DInput mode)

## Quick install

```
curl -fsSL https://raw.githubusercontent.com/PericlesSavio/8bitdo-keymap/main/get.sh | sh
```

Downloads the latest release (prebuilt binaries, x86_64 or aarch64) and runs its
`install.sh`. Works on Debian, Fedora, Arch, openSUSE and their derivatives (Ubuntu,
Mint, Pop!_OS, RHEL, Rocky, Manjaro, EndeavourOS…). Requires `sudo` and a systemd
desktop session.

The installer sets up the daemon, the CLI, the GUI (**8BitDo Key Mapper** in the
application menu), the udev rule that gives the logged-in user access to `/dev/uinput`,
and enables the `8bitdod` user service, which starts on login. The GUI needs PySide6:
it is installed from the distro packages, or into a private venv where none exists
(e.g. Ubuntu 24.04).

To remove everything: `8bitdoctl-uninstall`.

### From source

Requires [Go](https://go.dev/) 1.21+.

```
git clone https://github.com/PericlesSavio/8bitdo-keymap.git
cd 8bitdo-keymap
./install.sh
```

## Usage

Put the controller in **DInput mode** (turn it on holding **B + Home**), then open
**8BitDo Key Mapper** from the menu, or use the CLI:

```
8bitdoctl status                  # is the controller connected?
8bitdoctl mapping list            # buttons and keys
8bitdoctl mapping set L4 leftbrace
8bitdoctl mapping set PL kpplus
8bitdoctl mapping set L4 none     # remove a mapping
8bitdoctl mapping reset --all
8bitdoctl mapping keys            # accepted key names
8bitdoctl mapping identify        # prints the name of each button you press
```

Mappings are stored in `~/.config/8bitdoctl/keymap.json` and apply as soon as the
controller connects (the daemon reconnects by itself when it is turned off and on).

Buttons: `L4 R4 PL PR A B X Y LB RB LT RT SELECT START HOME LS RS UP DOWN LEFT RIGHT`.

| Button | evdev code |
|---|---|
| L4 | 704 (`BTN_TRIGGER_HAPPY1`) |
| R4 | 705 (`BTN_TRIGGER_HAPPY2`) |
| PL | 309 (`BTN_Z`) |
| PR | 306 (`BTN_C`) |

## Notes

- **DInput mode is required** (`2dc8:6012`). In XInput mode the `xpad` driver does not
  expose the extra buttons.
- A mapped button still works as a gamepad button — games that recognize it see both.
  Hiding it from games would require grabbing the whole controller and recreating it
  virtually (not implemented).
- The udev rule gives the logged-in user access to `/dev/uinput` (the same as Steam's
  `steam-devices` rules), which lets any program of that user inject input.

## Development

```
make build   # builds ./8bitdod and ./8bitdoctl
make test
./8bitdod --pretty-logs   # runs the daemon without installing
```

Pushing a `v*` tag builds the release archives (`.github/workflows/release.yml`).

## License

GPL-3.0 — the virtual keyboard code is derived from flydigictl.
