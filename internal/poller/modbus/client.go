// internal/poller/modbus/client.go
package modbus

import (
	"errors"
	"time"

	"github.com/goburrow/modbus"
)

// Client implements poller.Client using Modbus TCP.
type Client struct {
	handler *modbus.TCPClientHandler
	client  modbus.Client
}

// Config is minimal transport config.
type Config struct {
	Endpoint  string
	UnitID    uint8
	Timeout   time.Duration
}

// New creates a connected Modbus TCP client.
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("modbus client: endpoint required")
	}

	h := modbus.NewTCPClientHandler(cfg.Endpoint)
	h.Timeout = cfg.Timeout
	h.SlaveId = cfg.UnitID

	if err := h.Connect(); err != nil {
		return nil, err
	}

	return &Client{
		handler: h,
		client:  modbus.NewClient(h),
	}, nil
}

// Close closes the TCP connection.
func (c *Client) Close() error {
	return c.handler.Close()
}

// ---- poller.Client interface ----

func (c *Client) ReadCoils(addr, qty uint16) ([]bool, error) {
	b, err := c.client.ReadCoils(addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackBits(b, int(qty)), nil
}

func (c *Client) ReadDiscreteInputs(addr, qty uint16) ([]bool, error) {
	b, err := c.client.ReadDiscreteInputs(addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackBits(b, int(qty)), nil
}

func (c *Client) ReadHoldingRegisters(addr, qty uint16) ([]uint16, error) {
	b, err := c.client.ReadHoldingRegisters(addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackRegisters(b), nil
}

func (c *Client) ReadInputRegisters(addr, qty uint16) ([]uint16, error) {
	b, err := c.client.ReadInputRegisters(addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackRegisters(b), nil
}

// ---- helpers (pure geometry) ----

func unpackBits(data []byte, count int) []bool {
	out := make([]bool, count)
	for i := 0; i < count; i++ {
		byteIdx := i / 8
		bitIdx := i % 8
		out[i] = (data[byteIdx]&(1<<bitIdx) != 0)
	}
	return out
}

func unpackRegisters(data []byte) []uint16 {
	n := len(data) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
	}
	return out
}
