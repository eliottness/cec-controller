package main

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/claes/cec"
)

// CEC device type constants for SetActiveSource.
// These correspond to the CEC logical device types defined in the spec.
const (
	CECDeviceTypeTV          = 0
	CECDeviceTypeRecording   = 1
	CECDeviceTypeTuner       = 3
	CECDeviceTypePlayback    = 4 // most appropriate for a PC/media player
	CECDeviceTypeAudioSystem = 5
)

type CEC struct {
	adapter    string
	retries    int
	deviceName string

	conn      CECConnection
	connMu    sync.RWMutex
	cecOpener func(string, string) (CECConnection, error)

	keyPresses chan *cec.KeyPress
}

func NewCEC(adapter string, deviceName string, connectionRetries int, keyPresses chan *cec.KeyPress) (*CEC, error) {
	return newCECWithOpener(adapter, deviceName, connectionRetries, keyPresses, func(adapter, deviceName string) (CECConnection, error) {
		conn, err := cec.Open(adapter, deviceName)
		if err != nil {
			return nil, err
		}
		return &CECConnectionWrapper{Connection: conn}, nil
	})
}

func newCECWithOpener(adapter string, deviceName string, connectionRetries int, keyPresses chan *cec.KeyPress, opener func(string, string) (CECConnection, error)) (*CEC, error) {
	if connectionRetries < 1 {
		slog.Warn("Connection retries must be at least 1, setting to 1")
		connectionRetries = 1
	}

	conn, err := opener(adapter, deviceName)
	if err != nil {
		return nil, err
	}

	conn.SetKeyPressesChan(keyPresses)

	return &CEC{
		conn:       conn,
		adapter:    adapter,
		retries:    connectionRetries,
		deviceName: deviceName,
		keyPresses: keyPresses,
		cecOpener:  opener,
	}, nil
}

func (c *CEC) reopen() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		slog.Warn("CEC Connection lost, reopening...")
		c.conn.Close()
		c.conn = nil
	}

	for i := 0; i < c.retries; i++ {
		conn, err := c.cecOpener(c.adapter, c.deviceName)
		if err != nil {
			slog.Error("Failed to open CEC connection", "attempt", i+1, "error", err)
			continue
		}

		// Here we are literally hoping nobody reads this value concurrently we have no choice
		c.conn = conn
		c.conn.SetKeyPressesChan(c.keyPresses)
		slog.Info("CEC connection re-established")
		return nil
	}

	return fmt.Errorf("failed to open CEC connection after %d attempts", c.retries)
}

// powerCall calls the appropriate power function while holding the read lock,
// ensuring the connection is not replaced concurrently by reopen().
func (c *CEC) powerCall(isPowerOn bool, address int) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	if isPowerOn {
		return c.conn.PowerOn(address)
	}
	return c.conn.Standby(address)
}

func (c *CEC) power(isPowerOn bool, addresses ...int) error {
	for _, addr := range addresses {
		if err := c.powerCall(isPowerOn, addr); err != nil {
			if err := c.reopen(); err != nil {
				return err
			}
			if err := c.powerCall(isPowerOn, addr); err != nil {
				return fmt.Errorf("failed to send power command to address %d after reopening: %w", addr, err)
			}
		}
	}
	return nil
}

func (c *CEC) PowerOn(addresses ...int) error {
	return c.power(true, addresses...)
}

func (c *CEC) Standby(addresses ...int) error {
	return c.power(false, addresses...)
}

// SetActiveSource broadcasts to the CEC network that this device is the active
// source, causing the TV to switch its input accordingly.
func (c *CEC) SetActiveSource(deviceType int) bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn.SetActiveSource(deviceType)
}

func (c *CEC) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
