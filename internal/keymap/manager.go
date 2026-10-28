package keymap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PericlesSavio/8bitdoctl/internal/gamepad"
	"github.com/rs/zerolog/log"
)

type keyboardWriter interface {
	SendKey(code uint16, press bool) error
	Close() error
}

// Manager holds the button-to-keyboard mappings, persists them to disk and
// translates gamepad button events into virtual keyboard events.
type Manager struct {
	mu sync.Mutex

	path     string
	keyNames map[uint16]string // button code -> keyboard key name
	keyCodes map[uint16]uint16 // button code -> KEY_* code

	kb keyboardWriter

	held map[uint16]uint16 // pressed button code -> KEY_* code it sent

	waiters     []chan uint16
	lastPress   uint16
	lastPressAt time.Time
}

func NewManager(path string) *Manager {
	m := &Manager{
		path:     path,
		keyNames: make(map[uint16]string),
		keyCodes: make(map[uint16]uint16),
		held:     make(map[uint16]uint16),
	}

	if err := m.load(); err != nil && !os.IsNotExist(err) {
		log.Err(err).Str("path", path).Msg("failed to load keyboard mappings")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.keyCodes) > 0 {
		if err := m.ensureKeyboard(); err != nil {
			log.Err(err).Msg("failed to create virtual keyboard")
		}
	}

	return m
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	var mappings map[string]string
	if err := json.Unmarshal(data, &mappings); err != nil {
		return fmt.Errorf("parse mappings: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for buttonName, key := range mappings {
		button, ok := gamepad.ButtonByName(buttonName)
		if !ok {
			log.Warn().Str("button", buttonName).Msg("ignoring mapping for unknown button")
			continue
		}

		name, code, ok := KeyCodeByName(key)
		if !ok {
			log.Warn().Str("key", key).Msg("ignoring mapping to unknown key")
			continue
		}

		m.keyNames[button.Code] = name
		m.keyCodes[button.Code] = code
	}

	return nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.mappings(), "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	return os.WriteFile(m.path, data, 0644)
}

func (m *Manager) ensureKeyboard() error {
	if m.kb != nil {
		return nil
	}

	kb, err := newVirtualKeyboard()
	if err != nil {
		return err
	}

	m.kb = kb
	return nil
}

// Set maps a physical button to a keyboard key, both given by name.
func (m *Manager) Set(buttonName, key string) error {
	button, ok := gamepad.ButtonByName(buttonName)
	if !ok {
		return fmt.Errorf("unknown button %q", buttonName)
	}

	name, code, ok := KeyCodeByName(key)
	if !ok {
		return fmt.Errorf("unknown keyboard key %q", key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureKeyboard(); err != nil {
		return fmt.Errorf("create virtual keyboard: %w", err)
	}

	m.keyNames[button.Code] = name
	m.keyCodes[button.Code] = code

	return m.save()
}

// Remove clears the keyboard mapping for a button. Removing a button that
// isn't mapped is a no-op.
func (m *Manager) Remove(buttonName string) error {
	button, ok := gamepad.ButtonByName(buttonName)
	if !ok {
		return fmt.Errorf("unknown button %q", buttonName)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.keyNames, button.Code)
	delete(m.keyCodes, button.Code)

	return m.save()
}

func (m *Manager) RemoveAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clear(m.keyNames)
	clear(m.keyCodes)

	return m.save()
}

// Mappings returns the current mappings as button name -> key name.
func (m *Manager) Mappings() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.mappings()
}

func (m *Manager) mappings() map[string]string {
	mappings := make(map[string]string, len(m.keyNames))
	for code, key := range m.keyNames {
		mappings[gamepad.ButtonName(code)] = key
	}

	return mappings
}

func (m *Manager) HasMappings() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.keyCodes) > 0
}

// HandleButton receives a button state change from the gamepad and emits the
// keyboard event for it, if the button is mapped.
func (m *Manager) HandleButton(button uint16, pressed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pressed {
		if len(m.waiters) > 0 {
			for _, ch := range m.waiters {
				select {
				case ch <- button:
				default:
				}
			}
			m.waiters = nil
		} else {
			// Nobody is waiting right now; remember the press so that a
			// caller that is still connecting can pick it up.
			m.lastPress = button
			m.lastPressAt = time.Now()
		}
	}

	if m.kb == nil {
		return
	}

	// A release sends the key that the press sent, even if the mapping
	// changed in between, so no key is left stuck down.
	var code uint16
	if pressed {
		var ok bool
		if code, ok = m.keyCodes[button]; !ok {
			return
		}
		m.held[button] = code
	} else {
		var ok bool
		if code, ok = m.held[button]; !ok {
			return
		}
		delete(m.held, button)
	}

	if err := m.kb.SendKey(code, pressed); err != nil {
		log.Err(err).Str("button", gamepad.ButtonName(button)).Msg("failed to send keyboard event")
	}
}

// ReleaseAll releases every key still held down, for when the controller
// disconnects in the middle of a press.
func (m *Manager) ReleaseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for button, code := range m.held {
		if err := m.kb.SendKey(code, false); err != nil {
			log.Err(err).Str("button", gamepad.ButtonName(button)).Msg("failed to release key")
		}
	}
	clear(m.held)
}

// WaitPress blocks until any button is pressed, returning its name, or until
// the timeout elapses, returning "". A press from the last few seconds is
// returned immediately, so that presses made while the caller was still
// connecting aren't lost.
func (m *Manager) WaitPress(timeout time.Duration) string {
	m.mu.Lock()

	if !m.lastPressAt.IsZero() && time.Since(m.lastPressAt) < 3*time.Second {
		button := m.lastPress
		m.lastPressAt = time.Time{}
		m.mu.Unlock()

		return gamepad.ButtonName(button)
	}

	ch := make(chan uint16, 1)
	m.waiters = append(m.waiters, ch)
	m.mu.Unlock()

	select {
	case button := <-ch:
		return gamepad.ButtonName(button)

	case <-time.After(timeout):
		m.mu.Lock()
		for i, c := range m.waiters {
			if c == ch {
				m.waiters = append(m.waiters[:i], m.waiters[i+1:]...)
				break
			}
		}
		m.mu.Unlock()

		return ""
	}
}
