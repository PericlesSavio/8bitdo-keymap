package client

import (
	"errors"
	"fmt"

	common "github.com/PericlesSavio/8bitdoctl/internal/dbus"
	"github.com/godbus/dbus/v5"
)

var ErrNotRunning = errors.New("8bitdod is not running (start it with: systemctl --user start 8bitdod)")

type Client struct {
	obj dbus.BusObject
}

func Dial() (*Client, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}

	return &Client{obj: conn.Object(common.BusName, common.ObjectPath)}, nil
}

func (c *Client) call(methodName string, args []any, retvalues ...any) error {
	err := c.obj.Call(common.InterfaceName+"."+methodName, 0, args...).Store(retvalues...)

	var dberr dbus.Error
	if errors.As(err, &dberr) && dberr.Name == "org.freedesktop.DBus.Error.ServiceUnknown" {
		return ErrNotRunning
	}

	return err
}

func (c *Client) GetServerVersion() (string, error) {
	var version string
	err := c.call("GetServerVersion", nil, &version)
	return version, err
}

func (c *Client) GetDevice() (string, error) {
	var device string
	err := c.call("GetDevice", nil, &device)
	return device, err
}

func (c *Client) SetKeyboardMapping(button, key string) error {
	return c.call("SetKeyboardMapping", []any{button, key})
}

func (c *Client) RemoveKeyboardMapping(button string) error {
	return c.call("RemoveKeyboardMapping", []any{button})
}

func (c *Client) ClearKeyboardMappings() error {
	return c.call("ClearKeyboardMappings", nil)
}

func (c *Client) GetKeyboardMappings() (map[string]string, error) {
	var mappings map[string]string
	err := c.call("GetKeyboardMappings", nil, &mappings)
	return mappings, err
}

func (c *Client) WaitButtonPress(timeoutMs uint32) (string, error) {
	var button string
	err := c.call("WaitButtonPress", []any{timeoutMs}, &button)
	return button, err
}
