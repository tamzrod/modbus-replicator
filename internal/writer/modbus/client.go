// internal/writer/modbus/client.go
package modbus

import (
	"errors"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

// EndpointClient is a single TCP connection to one MMA endpoint.
// It serializes requests because it mutates SlaveId per memory write.
type EndpointClient struct {
	mu      sync.Mutex
	handler *modbus.TCPClientHandler
	client  modbus.Client
}

type Config struct {
	Endpoint string
	Timeout  time.Duration
}

func NewEndpointClient(cfg Config) (*EndpointClient, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("writer modbus: endpoint required")
	}

	h := modbus.NewTCPClientHandler(cfg.Endpoint)
	h.Timeout = cfg.Timeout

	if err := h.Connect(); err != nil {
		return nil, err
	}

	return &EndpointClient{
		handler: h,
		client:  modbus.NewClient(h),
	}, nil
}

func (c *EndpointClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.handler.Close()
}

func (c *EndpointClient) WriteCoils(unitID uint8, addr uint16, bits []bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handler.SlaveId = unitID

	qty := uint16(len(bits))
	payload := packBits(bits)

	_, err := c.client.WriteMultipleCoils(addr, qty, payload)
	return err
}

func (c *EndpointClient) WriteRegisters(unitID uint8, addr uint16, regs []uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handler.SlaveId = unitID

	qty := uint16(len(regs))
	payload := packRegisters(regs)

	_, err := c.client.WriteMultipleRegisters(addr, qty, payload)
	return err
}

func packBits(bits []bool) []byte {
	n := (len(bits) + 7) / 8
	out := make([]byte, n)
	for i, v := range bits {
		if v {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

func packRegisters(regs []uint16) []byte {
	out := make([]byte, len(regs)*2)
	for i, r := range regs {
		out[2*i] = byte(r >> 8)
		out[2*i+1] = byte(r)
	}
	return out
}
