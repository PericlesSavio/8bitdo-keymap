package keymap

import (
	"path/filepath"
	"testing"
	"time"
)

type fakeKeyboard struct {
	events []struct {
		code  uint16
		press bool
	}
}

func (f *fakeKeyboard) SendKey(code uint16, press bool) error {
	f.events = append(f.events, struct {
		code  uint16
		press bool
	}{code, press})
	return nil
}

func (f *fakeKeyboard) Close() error { return nil }

func newTestManager(t *testing.T, kb keyboardWriter) *Manager {
	return &Manager{
		path:     filepath.Join(t.TempDir(), "keymap.json"),
		keyNames: make(map[uint16]string),
		keyCodes: make(map[uint16]uint16),
		held:     make(map[uint16]uint16),
		kb:       kb,
	}
}

const (
	btnA  = 304
	btnL4 = 704

	keyF5        = 63
	keyLeftBrace = 26
)

func TestHandleButton(t *testing.T) {
	kb := &fakeKeyboard{}
	m := newTestManager(t, kb)

	if err := m.Set("L4", "f5"); err != nil {
		t.Fatalf("set mapping: %s", err)
	}

	m.HandleButton(btnL4, true)  // press L4
	m.HandleButton(btnL4, false) // release L4
	m.HandleButton(btnA, true)   // press A, which has no mapping
	m.HandleButton(btnA, false)

	if len(kb.events) != 2 {
		t.Fatalf("got %d keyboard events, want 2: %v", len(kb.events), kb.events)
	}
	if kb.events[0].code != keyF5 || !kb.events[0].press {
		t.Errorf("event 0 = %+v, want press of code %d", kb.events[0], keyF5)
	}
	if kb.events[1].code != keyF5 || kb.events[1].press {
		t.Errorf("event 1 = %+v, want release of code %d", kb.events[1], keyF5)
	}
}

func TestReleaseSendsPressedKey(t *testing.T) {
	kb := &fakeKeyboard{}
	m := newTestManager(t, kb)

	if err := m.Set("L4", "f5"); err != nil {
		t.Fatalf("set mapping: %s", err)
	}

	m.HandleButton(btnL4, true)
	if err := m.Set("L4", "leftbrace"); err != nil {
		t.Fatalf("change mapping: %s", err)
	}
	m.HandleButton(btnL4, false)

	if len(kb.events) != 2 || kb.events[1].code != keyF5 || kb.events[1].press {
		t.Errorf("events = %v, want the release to be of the original key %d", kb.events, keyF5)
	}
}

func TestReleaseAll(t *testing.T) {
	kb := &fakeKeyboard{}
	m := newTestManager(t, kb)

	if err := m.Set("L4", "leftbrace"); err != nil {
		t.Fatalf("set mapping: %s", err)
	}

	m.HandleButton(btnL4, true)
	m.ReleaseAll()
	m.HandleButton(btnL4, false) // the release after reconnecting sends nothing

	if len(kb.events) != 2 || kb.events[1].code != keyLeftBrace || kb.events[1].press {
		t.Errorf("events = %v, want a press and a single release of code %d", kb.events, keyLeftBrace)
	}
}

func TestSetPersistsAndReloads(t *testing.T) {
	m := newTestManager(t, &fakeKeyboard{})

	if err := m.Set("pl", "kpplus"); err != nil {
		t.Fatalf("set mapping: %s", err)
	}

	if got := m.Mappings()["PL"]; got != "kpplus" {
		t.Errorf(`mapping = %q, want "kpplus" under the canonical button name`, got)
	}

	reloaded := NewManager(m.path)
	if got := reloaded.Mappings()["PL"]; got != "kpplus" {
		t.Errorf(`reloaded mapping = %q, want "kpplus"`, got)
	}

	if err := m.Remove("PL"); err != nil {
		t.Fatalf("remove mapping: %s", err)
	}
	if m.HasMappings() {
		t.Error("mappings should be empty after removal")
	}
}

func TestWaitPress(t *testing.T) {
	m := newTestManager(t, &fakeKeyboard{})

	done := make(chan string, 1)
	go func() {
		done <- m.WaitPress(2 * time.Second)
	}()

	// Give WaitPress time to register its waiter before pressing.
	time.Sleep(50 * time.Millisecond)
	m.HandleButton(btnA, true)

	if got := <-done; got != "A" {
		t.Errorf("WaitPress = %q, want A", got)
	}

	if got := m.WaitPress(50 * time.Millisecond); got != "" {
		t.Errorf("WaitPress after timeout = %q, want empty", got)
	}
}

func TestWaitPressPicksUpRecentPress(t *testing.T) {
	m := newTestManager(t, &fakeKeyboard{})

	m.HandleButton(btnL4, true) // press L4 with nobody waiting

	if got := m.WaitPress(50 * time.Millisecond); got != "L4" {
		t.Errorf("WaitPress = %q, want the recent press L4", got)
	}

	// The remembered press is consumed by the call above.
	if got := m.WaitPress(50 * time.Millisecond); got != "" {
		t.Errorf("WaitPress after consuming = %q, want empty", got)
	}
}
