package main

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/claes/cec"
)

type CEC struct {
	adapter    string
	retries    int
	deviceName string

	conn       CECConnection
	realConn   *cec.Connection // Keep reference for reopening
	connMu     sync.RWMutex
	cecOpener  func(string, string) (*cec.Connection, error)

	keyPresses chan *cec.KeyPress
}

func NewCEC(adapter string, deviceName string, connectionRetries int, keyPresses chan *cec.KeyPress) (*CEC, error) {
	return NewCECWithOpener(adapter, deviceName, connectionRetries, keyPresses, cec.Open)
}

func NewCECWithOpener(adapter string, deviceName string, connectionRetries int, keyPresses chan *cec.KeyPress, opener func(string, string) (*cec.Connection, error)) (*CEC, error) {
	if connectionRetries < 1 {
		slog.Warn("Connection retries must be at least 1, setting to 1")
		connectionRetries = 1
	}

	c, err := opener(adapter, deviceName)
	if err != nil {
		return nil, err
	}

	c.KeyPresses = keyPresses

	return &CEC{
		conn:       &CECConnectionWrapper{conn: c},
		realConn:   c,
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
		c.realConn = nil
	}

	for i := 0; i < c.retries; i++ {
		var err error
		c.realConn, err = c.cecOpener(c.adapter, c.deviceName)
		if err != nil {
			slog.Error("Failed to open CEC connection", "attempt", i+1, "error", err)
			continue
		}

		// Here we are literally hoping nobody reads this value concurrently we have no choice
		c.realConn.KeyPresses = c.keyPresses
		c.conn = &CECConnectionWrapper{conn: c.realConn}
		slog.Info("CEC connection re-established")
		return nil
	}

	return fmt.Errorf("failed to open CEC connection after %d attempts", c.retries)
}

func (c *CEC) powerCall(powerFunc func(int) error, address int) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return powerFunc(address)
}

func (c *CEC) power(powerFunc func(int) error, addresses ...int) error {
	for _, addr := range addresses {
		if powerFunc(addr) == nil { // error values are inverted in this lib for this function
			// Error is nil on failure
			if err := c.reopen(); err != nil {
				return err
			}

			if powerFunc(addr) == nil {
				return fmt.Errorf("failed to send PowerOn to address %d after reopening connection", addr)
			}
		}
	}

	return nil
}

func (c *CEC) PowerOn(addresses ...int) error {
	return c.power(c.conn.PowerOn, addresses...)
}

func (c *CEC) Standby(addresses ...int) error {
	return c.power(c.conn.Standby, addresses...)
}

func (c *CEC) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
