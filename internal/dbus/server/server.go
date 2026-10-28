package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	common "github.com/PericlesSavio/8bitdoctl/internal/dbus"
	"github.com/PericlesSavio/8bitdoctl/internal/gamepad"
	"github.com/PericlesSavio/8bitdoctl/internal/keymap"
	"github.com/PericlesSavio/8bitdoctl/internal/version"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/rs/zerolog/log"
)

const scanInterval = 2 * time.Second

// Server exposes the keyboard mappings over DBus and feeds the controller's
// button events into them. Every exported method returning *dbus.Error is a
// DBus method.
type Server struct {
	km *keymap.Manager

	mu     sync.Mutex
	device string
}

func New(km *keymap.Manager) *Server {
	return &Server{km: km}
}

func (s *Server) GetServerVersion() (string, *dbus.Error) {
	return version.Version, nil
}

// GetDevice returns the name of the connected controller, or "" if there's
// none.
func (s *Server) GetDevice() (string, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.device, nil
}

func (s *Server) SetKeyboardMapping(button, key string) *dbus.Error {
	return makeError(s.km.Set(button, key))
}

func (s *Server) RemoveKeyboardMapping(button string) *dbus.Error {
	return makeError(s.km.Remove(button))
}

func (s *Server) ClearKeyboardMappings() *dbus.Error {
	return makeError(s.km.RemoveAll())
}

func (s *Server) GetKeyboardMappings() (map[string]string, *dbus.Error) {
	return s.km.Mappings(), nil
}

func (s *Server) WaitButtonPress(timeoutMs uint32) (string, *dbus.Error) {
	return s.km.WaitPress(time.Duration(timeoutMs) * time.Millisecond), nil
}

// Listen exports the server on the session bus, then watches for the
// controller forever. The controller is only read once the bus name is
// owned, so that a second daemon can't type every key twice.
func (s *Server) Listen() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}

	if err := conn.Export(s, common.ObjectPath, common.InterfaceName); err != nil {
		return fmt.Errorf("export server: %w", err)
	}

	node := &introspect.Node{
		Name: common.ObjectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{Name: common.InterfaceName, Methods: introspect.Methods(s)},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), common.ObjectPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export introspection: %w", err)
	}

	reply, err := conn.RequestName(common.BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("request bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return errors.New("bus name already taken, is 8bitdod already running?")
	}

	log.Info().Str("name", common.BusName).Msg("listening on the session bus")

	s.watchController()
	return nil
}

func (s *Server) watchController() {
	for {
		dev, ok := gamepad.Find()
		if !ok {
			time.Sleep(scanInterval)
			continue
		}

		log.Info().Str("device", dev.Name).Str("path", dev.Path).Msg("controller connected")
		s.setDevice(dev.Name)

		err := dev.Run(s.km.HandleButton)

		dev.Close()
		s.km.ReleaseAll()
		s.setDevice("")

		log.Info().Err(err).Msg("controller disconnected")
	}
}

func (s *Server) setDevice(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.device = name
}

func makeError(err error) *dbus.Error {
	if err == nil {
		return nil
	}

	return dbus.NewError(common.ErrorFailed, []any{err.Error()})
}
