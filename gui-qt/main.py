#!/usr/bin/env python3
"""Qt (PySide6) interface for mapping 8BitDo controller buttons to keys.

Thin shell over the `8bitdoctl` CLI: every action becomes a call to the
binary, which talks to the 8bitdod daemon over DBus. All validation and
mapping logic lives in the CLI; this is presentation only.
"""

import os
import shutil
import sys

from PySide6.QtCore import QEvent, QEventLoop, QProcess, Qt, QTimer, Signal
from PySide6.QtGui import QColor, QIcon, QPalette
from PySide6.QtWidgets import (
    QAbstractItemView,
    QApplication,
    QHBoxLayout,
    QHeaderView,
    QLabel,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QTableWidget,
    QTableWidgetItem,
    QVBoxLayout,
    QWidget,
)

CTL = (
    os.environ.get("EIGHTBITDOCTL") or shutil.which("8bitdoctl") or "/usr/bin/8bitdoctl"
)

KEY_LABELS = {
    "minus": "minus ( - )", "equal": "equal ( = / + with shift )",
    "comma": "comma ( , )", "dot": "dot ( . )", "slash": "slash ( / )",
    "backslash": "backslash ( \\ )", "semicolon": "semicolon ( ; )",
    "apostrophe": "apostrophe ( ' )", "grave": "grave ( ` )",
    "leftbrace": "leftbrace ( [ )", "rightbrace": "rightbrace ( ] )",
    "up": "up ( ↑ )", "down": "down ( ↓ )", "left": "left ( ← )", "right": "right ( → )",
    "leftctrl": "left ctrl", "rightctrl": "right ctrl",
    "leftshift": "left shift", "rightshift": "right shift",
    "leftalt": "left alt", "rightalt": "altgr (right alt)",
    "leftmeta": "left super/win", "rightmeta": "right super/win",
    "kpplus": "kpplus ( numpad + )", "kpminus": "kpminus ( numpad - )",
    "kpasterisk": "kpasterisk ( numpad * )", "kpslash": "kpslash ( numpad / )",
    # Named after the US keypad; on the ABNT2 layout kpdot types "," and
    # kpcomma (the extra keypad key) types ".".
    "kpdot": "numpad ,  (kpdot)",
    "kpcomma": "numpad .  (kpcomma)", "kpenter": "numpad enter",
    "kpequal": "kpequal ( numpad = )",
    "sysrq": "print screen", "compose": "compose (menu)",
    "playpause": "play/pause", "nextsong": "next track",
    "previoussong": "previous track",
    "volumeup": "volume +", "volumedown": "volume -",
}
for _i in range(10):
    KEY_LABELS[f"kp{_i}"] = f"kp{_i} ( numpad {_i} )"

# evdev key codes (linux/input-event-codes.h) -> names accepted by the CLI.
# Mirrors internal/keymap/keycodes.go.
KEY_NAMES = {
    1: "esc", 2: "1", 3: "2", 4: "3", 5: "4", 6: "5", 7: "6", 8: "7", 9: "8",
    10: "9", 11: "0", 12: "minus", 13: "equal", 14: "backspace", 15: "tab",
    16: "q", 17: "w", 18: "e", 19: "r", 20: "t", 21: "y", 22: "u", 23: "i",
    24: "o", 25: "p", 26: "leftbrace", 27: "rightbrace", 28: "enter",
    29: "leftctrl", 30: "a", 31: "s", 32: "d", 33: "f", 34: "g", 35: "h",
    36: "j", 37: "k", 38: "l", 39: "semicolon", 40: "apostrophe", 41: "grave",
    42: "leftshift", 43: "backslash", 44: "z", 45: "x", 46: "c", 47: "v",
    48: "b", 49: "n", 50: "m", 51: "comma", 52: "dot", 53: "slash",
    54: "rightshift", 55: "kpasterisk", 56: "leftalt", 57: "space",
    58: "capslock", 59: "f1", 60: "f2", 61: "f3", 62: "f4", 63: "f5",
    64: "f6", 65: "f7", 66: "f8", 67: "f9", 68: "f10", 69: "numlock",
    70: "scrolllock", 71: "kp7", 72: "kp8", 73: "kp9", 74: "kpminus",
    75: "kp4", 76: "kp5", 77: "kp6", 78: "kpplus", 79: "kp1", 80: "kp2",
    81: "kp3", 82: "kp0", 83: "kpdot", 87: "f11", 88: "f12", 96: "kpenter",
    97: "rightctrl", 98: "kpslash", 99: "sysrq", 100: "rightalt", 102: "home",
    103: "up", 104: "pageup", 105: "left", 106: "right", 107: "end",
    108: "down", 109: "pagedown", 110: "insert", 111: "delete", 113: "mute",
    114: "volumedown", 115: "volumeup", 117: "kpequal", 119: "pause",
    121: "kpcomma", 125: "leftmeta", 126: "rightmeta", 127: "compose",
    163: "nextsong", 164: "playpause", 165: "previoussong",
    **{183 + i: f"f{13 + i}" for i in range(12)},
}

# XKB keycodes on Linux (X11 and Wayland) are evdev codes offset by 8.
XKB_OFFSET = 8


class KeyCaptureButton(QPushButton):
    """Click, then press a keyboard key: emits the key's CLI name."""

    captured = Signal(str)

    def __init__(self, key, parent=None):
        super().__init__(parent)
        self.key = key
        self.capturing = False
        self.setToolTip("Click, then press the keyboard key to assign")
        self.clicked.connect(self.toggle)
        self.show_key()

    def show_key(self):
        self.setText(KEY_LABELS.get(self.key, self.key) if self.key else "— none —")

    def toggle(self):
        if self.capturing:
            self.stop()
            return
        self.capturing = True
        self.setText("press a key…  (click to cancel)")
        self.setFocus()
        self.grabKeyboard()

    def stop(self):
        self.capturing = False
        self.releaseKeyboard()
        self.show_key()

    def event(self, e):
        # Intercept every key (Tab, arrows, Esc…) before focus navigation.
        if self.capturing and e.type() == QEvent.Type.KeyPress:
            if e.isAutoRepeat():
                return True
            name = KEY_NAMES.get(e.nativeScanCode() - XKB_OFFSET)
            self.stop()
            if name:
                self.captured.emit(name)
            else:
                self.window().statusBar().showMessage(
                    f"key not supported (scan code {e.nativeScanCode()})", 5000
                )
            return True
        if self.capturing and e.type() == QEvent.Type.KeyRelease:
            return True
        return super().event(e)

    def focusOutEvent(self, e):
        if self.capturing:
            self.stop()
        super().focusOutEvent(e)


def run_ctl(args, timeout_ms=25000):
    """Run 8bitdoctl without freezing the UI (local event loop)."""
    proc = QProcess()
    proc.start(CTL, args)

    loop = QEventLoop()
    proc.finished.connect(loop.quit)

    timer = QTimer()
    timer.setSingleShot(True)
    timer.timeout.connect(loop.quit)
    timer.start(timeout_ms)

    if proc.state() != QProcess.ProcessState.NotRunning:
        loop.exec()

    if proc.state() != QProcess.ProcessState.NotRunning:
        proc.kill()
        proc.waitForFinished(1000)
        return False, "timed out talking to 8bitdoctl"

    out = bytes(proc.readAllStandardOutput()).decode(errors="replace")
    err = bytes(proc.readAllStandardError()).decode(errors="replace")

    if proc.exitStatus() != QProcess.ExitStatus.NormalExit or proc.exitCode() != 0:
        return False, (err or out).strip() or "failed to run 8bitdoctl"

    return True, out


class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("8BitDo — Keyboard Mapping")
        self.setWindowIcon(QIcon.fromTheme("input-gaming"))
        self.resize(640, 640)

        central = QWidget()
        layout = QVBoxLayout(central)

        top = QHBoxLayout()
        self.identify_btn = QPushButton("Detect button")
        self.identify_btn.setToolTip("Press a button on the controller to locate it in the list")
        self.reload_btn = QPushButton("Reload")
        self.about_btn = QPushButton("About")
        top.addWidget(self.identify_btn)
        top.addWidget(self.reload_btn)
        top.addStretch()
        top.addWidget(self.about_btn)
        layout.addLayout(top)

        self.warning = QLabel(
            "⚠  Controller not detected. Turn it on in DInput mode — mappings "
            "can still be edited and apply as soon as it connects."
        )
        self.warning.setWordWrap(True)
        self.warning.setStyleSheet(
            "background: #33342e; border: 1px solid #75715e; padding: 8px;"
        )
        self.warning.hide()
        layout.addWidget(self.warning)

        self.table = QTableWidget(0, 3)
        self.table.setHorizontalHeaderLabels(["Button", "Keyboard key", ""])
        header = self.table.horizontalHeader()
        header.setSectionResizeMode(0, QHeaderView.ResizeMode.ResizeToContents)
        header.setSectionResizeMode(1, QHeaderView.ResizeMode.Stretch)
        header.setSectionResizeMode(2, QHeaderView.ResizeMode.ResizeToContents)
        self.table.verticalHeader().setVisible(False)
        self.table.setSelectionBehavior(QAbstractItemView.SelectionBehavior.SelectRows)
        self.table.setEditTriggers(QAbstractItemView.EditTrigger.NoEditTriggers)
        layout.addWidget(self.table)

        self.setCentralWidget(central)
        self.statusBar().showMessage("loading…")

        self.reload_btn.clicked.connect(self.reload)
        self.identify_btn.clicked.connect(self.identify)
        self.about_btn.clicked.connect(self.show_about)

        QTimer.singleShot(0, self.reload)

    def set_busy(self, busy, message=""):
        self.centralWidget().setEnabled(not busy)
        if message:
            self.statusBar().showMessage(message)
        if busy:
            QApplication.setOverrideCursor(Qt.CursorShape.WaitCursor)
        else:
            QApplication.restoreOverrideCursor()

    def reload(self):
        self.set_busy(True, "querying daemon…")

        ok, out = run_ctl(["mapping", "list", "-t"])
        if not ok:
            self.table.setRowCount(0)
            self.set_busy(False, "daemon not available")
            QMessageBox.warning(
                self, "No connection",
                f"Could not talk to the 8bitdod daemon:\n\n{out}",
            )
            return

        mappings = []
        for line in out.splitlines():
            fields = line.split()
            if len(fields) != 2:
                continue
            mappings.append({
                "button": fields[0],
                "key": "" if fields[1] == "-" else fields[1],
            })

        _, device_out = run_ctl(["status", "-t"])
        device = device_out.strip()

        self.table.setRowCount(0)
        for m in mappings:
            self.add_row(m)

        self.warning.setVisible(not device)
        self.identify_btn.setEnabled(bool(device))

        mapped = sum(1 for m in mappings if m["key"])
        status = f"{device or 'controller not connected'} · {mapped} buttons mapped"
        self.set_busy(False, status)

    def add_row(self, m):
        row = self.table.rowCount()
        self.table.insertRow(row)

        item = QTableWidgetItem(m["button"])
        item.setData(Qt.ItemDataRole.UserRole, m["button"])
        self.table.setItem(row, 0, item)

        button = m["button"]

        capture = KeyCaptureButton(m["key"])
        # Defer past the signal handler: reload destroys the emitting widget.
        capture.captured.connect(lambda key: QTimer.singleShot(0, lambda: self.apply(
            ["mapping", "set", button, key], "saving key…"
        )))
        self.table.setCellWidget(row, 1, capture)

        clear = QPushButton("✕")
        clear.setToolTip("Remove mapping")
        clear.setEnabled(bool(m["key"]))
        clear.clicked.connect(lambda: QTimer.singleShot(0, lambda: self.apply(
            ["mapping", "set", button, "none"], "removing key…"
        )))
        self.table.setCellWidget(row, 2, clear)

    def apply(self, args, message):
        self.set_busy(True, message)
        ok, out = run_ctl(args)
        self.set_busy(False)

        if not ok:
            QMessageBox.warning(self, "Failed to apply", out)

        self.reload()

    def show_about(self):
        QMessageBox.about(
            self,
            "About",
            """<h3>8BitDo Key Mapper</h3>
            <p>Maps the buttons of <b>8BitDo</b> controllers to keyboard keys
            on Linux, including the back buttons (L4, R4, PL, PR).</p>
            <p>Péricles S. — <a href="https://github.com/PericlesSavio">@PericlesSavio</a></p>
            <p>GPL-3.0 license · keyboard injection derived from
            <a href="https://github.com/pipe01/flydigictl">flydigictl</a></p>
            <p style="color: #75715e">To uninstall everything, run:<br>
            <code>/usr/bin/8bitdoctl-uninstall</code></p>""",
        )

    def identify(self):
        self.set_busy(True, "press a button on the controller…")
        ok, out = run_ctl(["mapping", "identify", "--once", "-t"], timeout_ms=15000)
        self.set_busy(False)

        button = out.strip()
        if not ok or not button:
            self.statusBar().showMessage("no button detected", 4000)
            return

        for row in range(self.table.rowCount()):
            if self.table.item(row, 0).data(Qt.ItemDataRole.UserRole) == button:
                self.table.selectRow(row)
                self.table.scrollToItem(
                    self.table.item(row, 0),
                    QAbstractItemView.ScrollHint.PositionAtCenter,
                )
                break

        self.statusBar().showMessage(f"button: {button}", 4000)


def apply_dark_theme(app):
    """Dark theme in Monokai-style gray tones, no vivid colors."""
    app.setStyle("Fusion")

    bg = QColor(0x1E, 0x1F, 0x1C)
    panel = QColor(0x27, 0x28, 0x22)
    base = QColor(0x23, 0x24, 0x1F)
    text = QColor(0xD6, 0xD6, 0xCF)
    disabled = QColor(0x75, 0x71, 0x5E)
    accent = QColor(0x4E, 0x4F, 0x47)

    p = QPalette()
    p.setColor(QPalette.ColorRole.Window, bg)
    p.setColor(QPalette.ColorRole.WindowText, text)
    p.setColor(QPalette.ColorRole.Base, base)
    p.setColor(QPalette.ColorRole.AlternateBase, panel)
    p.setColor(QPalette.ColorRole.Text, text)
    p.setColor(QPalette.ColorRole.Button, panel)
    p.setColor(QPalette.ColorRole.ButtonText, text)
    p.setColor(QPalette.ColorRole.ToolTipBase, panel)
    p.setColor(QPalette.ColorRole.ToolTipText, text)
    p.setColor(QPalette.ColorRole.Highlight, accent)
    p.setColor(QPalette.ColorRole.HighlightedText, QColor(0xE8, 0xE8, 0xE2))
    for role in (QPalette.ColorRole.Text, QPalette.ColorRole.ButtonText,
                 QPalette.ColorRole.WindowText):
        p.setColor(QPalette.ColorGroup.Disabled, role, disabled)
    app.setPalette(p)


def main():
    app = QApplication(sys.argv)
    app.setApplicationName("8bitdoctl-qt")
    app.setDesktopFileName("8bitdoctl-qt")

    apply_dark_theme(app)

    win = MainWindow()
    win.show()

    sys.exit(app.exec())


if __name__ == "__main__":
    main()
